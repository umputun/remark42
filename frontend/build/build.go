package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
)

// entries are the files the widget is loaded as. Each is a list of imports, bundled in order into
// one script named after the entry, with the styles they reach in a stylesheet of the same name.
// remark and last-comments start with their custom properties, so those come first in their
// stylesheets and every rule after them can read one. last-comments renders into the site's own
// page, so its palette sits on the widget's node instead of :root, where it would replace the
// page's variables of the same names.
//
// packages says an entry may bundle what is installed in node_modules. Only remark does: it renders
// preact components, so a development build also loads preact/debug into it, ahead of everything
// else, for the Preact DevTools extension. The other entries are written without packages, and a
// build in which one of them reaches into node_modules fails.
var entries = []struct {
	name     string
	imports  []string
	packages bool
}{
	{name: "embed", imports: []string{"./app/embed.ts"}},
	{name: "deleteme", imports: []string{"./app/deleteme.ts"}},
	{name: "last-comments", imports: []string{"./app/styles/last-comments-properties.css", "./app/last-comments.js"}},
	{name: "remark", imports: []string{"./app/styles/custom-properties.css", "./app/remark.tsx"}, packages: true},
}

// engines is the oldest version of each browser the scripts and styles are written down to. It is
// what the query "defaults, not IE 11, not samsung 12" resolved to when the build moved off
// browserslist in November 2026; Opera Mobile 80 and Samsung Internet 29 from that list run newer
// Chromium than the Chrome named here. Raising a version is a decision to drop its users.
var engines = []api.Engine{
	{Name: api.EngineChrome, Version: "109"},
	{Name: api.EngineEdge, Version: "149"},
	{Name: api.EngineFirefox, Version: "140"},
	{Name: api.EngineIOS, Version: "18.5"},
	{Name: api.EngineOpera, Version: "127"},
	{Name: api.EngineSafari, Version: "26.3"},
}

// lowered lists syntax every engine above reads and the stylesheets still go out without, as they
// always have: nested rules and range media queries are rewritten into their older form, and four
// offsets are not folded into inset. A browser older than the list that still runs the scripts
// keeps its layout, at a cost of a few hundred bytes. A nested rule is also written out as one
// selector per parent instead of one :is() over all of them, which matters beyond old browsers:
// :is() takes the weight of its heaviest argument, so the rule would start winning over rules it
// has always lost to.
var lowered = map[string]bool{
	"nesting":         false,
	"is-pseudo-class": false,
	"inset-property":  false,
	"media-range":     false,
}

const (
	entryNamespace  = "entry"
	moduleNamespace = "css-module"

	// classMarker starts the name every CSS module is bundled under in a production build, which
	// makes it the start of every class and animation name esbuild derives from that module:
	// "<marker><n>_<name>". renameClasses finds the names by it and replaces them. It has to be a
	// word that occurs nowhere else in the widget.
	classMarker = "r42cssmodule"
)

var classToken = regexp.MustCompile(classMarker + `\d+_[A-Za-z0-9_-]+`)

// a string of class names of which at least one is a module's, which is how a script holds them:
// one name, or several joined by spaces where a class composes others, a global one among them
var classString = regexp.MustCompile(`"(?:[A-Za-z0-9_-]+ )*` + classToken.String() + `(?: [A-Za-z0-9_-]+)*"`)

// a class in a selector, or an animation where it is defined
var globalName = regexp.MustCompile(`(?:\.|@keyframes )(-?[A-Za-z_][A-Za-z0-9_-]*)`)

// newModuleNames starts the record for one build of appDir. esbuild hands files over by their real
// paths, so the directory they are told apart from has to be the real one as well.
func newModuleNames(appDir string) *moduleNames {
	if resolved, err := filepath.EvalSymlinks(appDir); err == nil {
		appDir = resolved
	}
	return &moduleNames{appDir: appDir}
}

// moduleNames remembers which CSS module each marker stands for
type moduleNames struct {
	mu     sync.Mutex
	appDir string
	stems  map[string]string // file → the name it is bundled under
	files  map[string]string // that name → the file, relative to the app directory
}

func (m *moduleNames) stem(file string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stem, ok := m.stems[file]; ok {
		return stem
	}
	if m.stems == nil {
		m.stems, m.files = map[string]string{}, map[string]string{}
	}
	stem := fmt.Sprintf("%s%d", classMarker, len(m.stems))
	rel, err := filepath.Rel(m.appDir, file)
	if err != nil {
		rel = file
	}
	m.stems[file], m.files[stem] = stem, filepath.ToSlash(rel)
	return stem
}

// bundledAs returns the path a CSS module is bundled under: its own, with the marker put in front
// of the file name. The marker is what the class names are derived from, and the rest is what
// esbuild prints when it has something to say about the file.
func (m *moduleNames) bundledAs(file string) string {
	stem := m.stem(file)
	m.mu.Lock()
	defer m.mu.Unlock()
	rel := m.files[stem]
	return path.Join(path.Dir(rel), stem+"."+path.Base(rel))
}

// options describes one build of the widget from appDir. A development build is not minified,
// carries source maps and preact/debug, and reads NODE_ENV as "development", which makes the
// widget take the address it was loaded from as its instance instead of the placeholder the server
// fills in. Nothing is written: the caller takes the files from the result.
func options(appDir string, dev bool, names *moduleNames) api.BuildOptions {
	env := "production"
	if dev {
		env = "development"
	}

	points := make([]api.EntryPoint, 0, len(entries))
	for _, e := range entries {
		points = append(points, api.EntryPoint{InputPath: entryNamespace + ":" + e.name, OutputPath: e.name})
	}

	plugins := []api.Plugin{entryPlugin(appDir, dev)}
	sourcemap := api.SourceMapLinked
	if !dev {
		plugins = append(plugins, modulePlugin(names))
		sourcemap = api.SourceMapNone
	}

	return api.BuildOptions{
		EntryPointsAdvanced: points,
		Plugins:             plugins,
		AbsWorkingDir:       appDir,
		Outdir:              filepath.Join(appDir, "public"),
		// the bundles are loaded as modules and, under their legacy .js names, as classic scripts,
		// so they carry no module syntax and keep their variables to themselves
		Format:       api.FormatIIFE,
		OutExtension: map[string]string{".js": ".mjs"},
		Bundle:       true,
		Platform:     api.PlatformBrowser,
		Engines:      engines,
		Supported:    lowered,
		Tsconfig:     filepath.Join(appDir, "tsconfig.json"),
		Loader: map[string]api.Loader{
			".module.css": api.LoaderLocalCSS,
			".css":        api.LoaderGlobalCSS,
		},
		Define:           map[string]string{"process.env.NODE_ENV": fmt.Sprintf("%q", env)},
		MinifyWhitespace: !dev,
		MinifySyntax:     !dev,
		// never here: esbuild's short names for CSS module classes start at one letter, and a
		// class named "p" or "w" is also what a code highlighter calls a token and what a host
		// page may call anything. A production build keeps the long names and compile replaces
		// them, then shortens the scripts' own identifiers in a pass of its own.
		MinifyIdentifiers: false,
		Sourcemap:         sourcemap,
		LogLevel:          api.LogLevelWarning,
		Write:             false,
		// read by refusePackages
		Metafile: true,
	}
}

// entryPlugin serves each entry as a module made of its imports, resolved from the app directory
func entryPlugin(appDir string, dev bool) api.Plugin {
	return api.Plugin{
		Name: "entries",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: "^" + entryNamespace + ":"},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{Path: strings.TrimPrefix(args.Path, entryNamespace+":"), Namespace: entryNamespace}, nil
				})
			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: entryNamespace},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					for _, e := range entries {
						if e.name != args.Path {
							continue
						}
						contents := entrySource(e.imports, dev && e.packages)
						return api.OnLoadResult{Contents: &contents, ResolveDir: appDir, Loader: api.LoaderJS}, nil
					}
					return api.OnLoadResult{}, fmt.Errorf("unknown entry %q", args.Path)
				})
		},
	}
}

func entrySource(imports []string, debug bool) string {
	var b strings.Builder
	if debug {
		b.WriteString("import 'preact/debug';\n")
	}
	for _, path := range imports {
		fmt.Fprintf(&b, "import %q;\n", path)
	}
	return b.String()
}

// modulePlugin bundles each CSS module under a name starting with classMarker. esbuild names a
// module's classes after its file, so this is what lets renameClasses tell them from every other
// word in the output.
func modulePlugin(names *moduleNames) api.Plugin {
	// marks the lookup this plugin makes itself, which has to reach esbuild's own resolver
	type passThrough struct{}

	return api.Plugin{
		Name: "css-module-names",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `\.module\.css$`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					if _, own := args.PluginData.(passThrough); own {
						return api.OnResolveResult{}, nil
					}
					found := build.Resolve(args.Path, api.ResolveOptions{
						Importer: args.Importer, Namespace: args.Namespace, ResolveDir: args.ResolveDir,
						Kind: args.Kind, PluginData: passThrough{},
					})
					if len(found.Errors) > 0 {
						return api.OnResolveResult{Errors: found.Errors}, nil
					}
					return api.OnResolveResult{Path: names.bundledAs(found.Path), Namespace: moduleNamespace, PluginData: found.Path}, nil
				})
			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: moduleNamespace},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					file, _ := args.PluginData.(string)
					source, err := os.ReadFile(file) //nolint:gosec // a path esbuild resolved from the widget's own imports
					if err != nil {
						return api.OnLoadResult{}, err
					}
					contents := string(source)
					return api.OnLoadResult{Contents: &contents, ResolveDir: filepath.Dir(file), Loader: api.LoaderLocalCSS}, nil
				})
		},
	}
}

// compile builds the widget and returns its files by name
func compile(appDir string, dev bool) (map[string][]byte, error) {
	names := newModuleNames(appDir)
	opts := options(appDir, dev, names)
	result := api.Build(opts)
	if len(result.Errors) > 0 {
		// esbuild has printed each error with its location already
		return nil, fmt.Errorf("build failed with %d error(s)", len(result.Errors))
	}
	if err := refusePackages(result.Metafile); err != nil {
		return nil, err
	}

	files, err := outputs(opts.Outdir, result.OutputFiles)
	if err != nil || dev {
		return files, err
	}
	if err := renameClasses(files, names); err != nil {
		return nil, err
	}
	return files, minifyScripts(files)
}

// refusePackages fails a build in which an entry written without packages bundles a file from
// node_modules, in its script or in its stylesheet. metafile is esbuild's description of the
// build, which lists the files that went into each output.
func refusePackages(metafile string) error {
	var built struct {
		Outputs map[string]struct {
			Inputs map[string]json.RawMessage `json:"inputs"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(metafile), &built); err != nil {
		return fmt.Errorf("reading what the build bundled: %w", err)
	}

	allowed := map[string]bool{}
	for _, e := range entries {
		allowed[e.name] = e.packages
	}

	var found []string
	for output, described := range built.Outputs {
		entry := strings.TrimSuffix(path.Base(output), path.Ext(output))
		if strings.HasSuffix(output, ".map") || allowed[entry] {
			continue
		}
		for input := range described.Inputs {
			if slices.Contains(strings.Split(input, "/"), "node_modules") {
				found = append(found, fmt.Sprintf("%s bundles %s", path.Base(output), input))
			}
		}
	}
	if len(found) == 0 {
		return nil
	}
	sort.Strings(found)
	return fmt.Errorf("an entry written without packages reaches into node_modules: %s", strings.Join(found, "; "))
}

// outputs indexes what a build produced by file name
func outputs(outDir string, produced []api.OutputFile) (map[string][]byte, error) {
	files := make(map[string][]byte, len(produced))
	for _, f := range produced {
		name, err := filepath.Rel(outDir, f.Path)
		if err != nil || strings.HasPrefix(name, "..") {
			return nil, fmt.Errorf("the build produced %s, outside %s", f.Path, outDir)
		}
		files[filepath.ToSlash(name)] = f.Contents
	}
	return files, nil
}

// renameClasses gives every CSS module class and animation its shipped name: the module's letters,
// an underscore, the name's letters, as in "c_B". The names mean nothing on purpose, since they
// are not an interface and move whenever a class or a module is added. The underscore is what
// keeps them apart from everything else a page carries: the single words a code highlighter marks
// tokens with, which arrive inside comments, and the class names of the host page, which
// last-comments shares a document with.
func renameClasses(files map[string][]byte, names *moduleNames) error {
	locals := map[string]map[string]bool{} // module → the names found under it
	collect := func(token string) {
		module, _, _ := strings.Cut(token, "_")
		if locals[module] == nil {
			locals[module] = map[string]bool{}
		}
		locals[module][token] = true
	}
	for name, content := range files {
		switch filepath.Ext(name) {
		case ".css":
			for _, token := range classToken.FindAllString(string(content), -1) {
				collect(token)
			}
		case ".mjs":
			for _, quoted := range classString.FindAllString(string(content), -1) {
				for _, token := range classToken.FindAllString(quoted, -1) {
					collect(token)
				}
			}
		}
	}

	// modules in the order of their paths and names in their own order, so a build of the same
	// sources always ships the same names
	modules := make([]string, 0, len(locals))
	for module := range locals {
		if _, known := names.files[module]; !known {
			return fmt.Errorf("class names under %q belong to no CSS module of this build", module)
		}
		modules = append(modules, module)
	}
	sort.Slice(modules, func(i, j int) bool { return names.files[modules[i]] < names.files[modules[j]] })

	// the names written as global, which stay as they are: in the stylesheets, and in the scripts
	// beside the module classes that compose them. A shipped name equal to one of them would make
	// two unrelated rules one, and nothing would show it but the page. Case is set aside, as a
	// host page without a doctype sets it aside.
	global := map[string]string{} // in lower case → as written
	for name, content := range files {
		switch filepath.Ext(name) {
		case ".css":
			for _, m := range globalName.FindAllStringSubmatch(string(content), -1) {
				global[strings.ToLower(m[1])] = m[1]
			}
		case ".mjs":
			for _, quoted := range classString.FindAllString(string(content), -1) {
				for _, word := range strings.Fields(strings.Trim(quoted, `"`)) {
					global[strings.ToLower(word)] = word
				}
			}
		}
	}

	shipped := map[string]string{}
	for m, module := range modules {
		tokens := make([]string, 0, len(locals[module]))
		for token := range locals[module] {
			tokens = append(tokens, token)
		}
		sort.Strings(tokens)
		for t, token := range tokens {
			shipped[token] = letters(m) + "_" + letters(t)
			if written, taken := global[shipped[token]]; taken {
				return fmt.Errorf("the global name %q has the form of the shipped name %q, which %s takes: rename the global one",
					written, shipped[token], names.files[module])
			}
		}
	}

	rename := func(token string) string { return shipped[token] }
	for name, content := range files {
		switch filepath.Ext(name) {
		case ".css":
			files[name] = []byte(classToken.ReplaceAllStringFunc(string(content), rename))
		case ".mjs":
			files[name] = []byte(classString.ReplaceAllStringFunc(string(content), func(quoted string) string {
				return classToken.ReplaceAllStringFunc(quoted, rename)
			}))
		}
	}
	return nil
}

// letters counts in the alphabet: a to z, then aa. Lower case only, because a host page without a
// doctype matches class names whatever their case, and two names apart by case alone would then
// be one.
func letters(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	name := ""
	for {
		name = string(alphabet[n%len(alphabet)]) + name
		if n = n/len(alphabet) - 1; n < 0 {
			return name
		}
	}
}

// minifyScripts shortens the identifiers of each script, which the build left long for the sake
// of the class names, and fails if a marker is still in anything that ships: a class name the
// rename did not reach would be a class no stylesheet has
func minifyScripts(files map[string][]byte) error {
	for name, content := range files {
		if filepath.Ext(name) == ".mjs" {
			result := api.Transform(string(content), api.TransformOptions{
				Loader: api.LoaderJS, Sourcefile: name, Engines: engines, LogLevel: api.LogLevelWarning,
				MinifyIdentifiers: true, MinifyWhitespace: true, MinifySyntax: true,
			})
			if len(result.Errors) > 0 {
				return fmt.Errorf("minifying %s failed with %d error(s)", name, len(result.Errors))
			}
			files[name] = result.Code
		}
		if strings.Contains(string(files[name]), classMarker) {
			return fmt.Errorf("%s still carries a class name under its build-time form %q", name, classMarker)
		}
	}
	return nil
}

// build writes one production build of the widget into outDir, in place of whatever was there
func build(appDir, outDir string) error {
	files, err := compile(appDir, false)
	if err != nil {
		return err
	}
	// what a previous build left goes first, so a file this one no longer writes is not left
	// behind to be served or embedded
	if err := emptyDir(outDir); err != nil {
		return fmt.Errorf("can't empty %s: %w", outDir, err)
	}
	for name, content := range files {
		target := filepath.Join(outDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(target, content, 0o644); err != nil { //nolint:gosec // served to browsers as is
			return err
		}
	}
	return nil
}

func emptyDir(dir string) error {
	names, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(dir, 0o750)
	}
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := os.RemoveAll(filepath.Join(dir, name.Name())); err != nil {
			return err
		}
	}
	return nil
}
