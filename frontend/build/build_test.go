package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

// fixture is the smallest app the builder accepts: the four entries, the custom properties, a css
// module written the way the widget's are, and a stand-in for preact/debug
var fixture = map[string]string{
	"tsconfig.json":                    `{"compilerOptions": {"baseUrl": "app", "target": "es6", "strict": true}}`,
	"app/embed.ts":                     `console.log('embed marker');`,
	"app/deleteme.ts":                  `console.log('deleteme marker');`,
	"app/last-comments.js":             `import { title } from 'shared'; console.log('last-comments marker', title);`,
	"app/styles/custom-properties.css": `:root { --fixture-color: #123456; }`,
	"node_modules/preact/package.json": `{"name": "preact", "exports": {"./debug": "./debug.js"}}`,
	"node_modules/preact/debug.js":     `console.log('preact debug marker');`,
	// imported through baseUrl, as the widget imports its own modules
	"app/shared.ts": `import styles from './shared.module.css';
import other from './other/shared.module.css';
export const title: string = [styles.fixtureTitle, styles.fixtureComposed, styles.fixtureThemed, other.fixtureTitle].join(' ');`,
	"app/other/shared.module.css": `.fixtureTitle { animation: fixtureSpin 1s; }
@keyframes fixtureSpin { to { opacity: 0; } }`,
	"app/remark.tsx": `import { title } from 'shared';
const instance = process.env.NODE_ENV === 'development' ? 'development-only marker' : '{% REMARK_URL %}';
console.log('remark marker', title, instance);`,
	"app/shared.module.css": `.fixtureTitle {
  color: var(--fixture-color);
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;

  &:hover { color: red; }

  @media (width >= 448px) { color: blue; }
}

.fixtureFirst,
.fixtureSecond.fixtureWide {
  & .fixtureInner { color: green; }
}

:global(.dark) .fixtureTitle { color: white; }

.fixtureComposed { composes: fixtureFirst; margin: 0; }

.fixtureThemed { composes: dark from global; padding: 1px; }

/* what a code highlighter calls punctuation, and a name a host page is free to use */
:global(.p) { color: gray; }
`,
}

func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range fixture {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // a file the test has just built
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func listed(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func mustContain(t *testing.T, name, body string, wanted ...string) {
	t.Helper()
	for _, w := range wanted {
		if !strings.Contains(body, w) {
			t.Errorf("%s has no %q:\n%s", name, w, body)
		}
	}
}

func mustNotContain(t *testing.T, name, body string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(body, u) {
			t.Errorf("%s contains %q:\n%s", name, u, body)
		}
	}
}

// shippedClass is the form every CSS module class and animation name ships in
var shippedClass = regexp.MustCompile(`^[a-z]+_[a-z]+$`)

// classSelectors returns the class names a stylesheet's selectors use
func classSelectors(css string) []string {
	seen := map[string]bool{}
	var names []string
	for _, m := range regexp.MustCompile(`\.([A-Za-z_][A-Za-z0-9_-]*)`).FindAllStringSubmatch(css, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	return names
}

// classOf returns the class of the one rule carrying the declaration
func classOf(t *testing.T, css, declaration string) string {
	t.Helper()
	m := regexp.MustCompile(`\.([A-Za-z0-9_-]+)\{[^{}]*`+regexp.QuoteMeta(declaration)).FindAllStringSubmatch(css, -1)
	if len(m) != 1 {
		t.Fatalf("want one rule with %q, found %d in:\n%s", declaration, len(m), css)
	}
	return m[0][1]
}

func TestBuildProduction(t *testing.T) {
	app := writeFixture(t)
	out := filepath.Join(app, "public")
	// what an earlier build left has to go, or it is served and embedded with the new one
	if err := os.MkdirAll(filepath.Join(out, "old"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "stale.mjs"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := build(app, out); err != nil {
		t.Fatal(err)
	}

	want := []string{"deleteme.mjs", "embed.mjs", "last-comments.css", "last-comments.mjs", "remark.css", "remark.mjs"}
	if got := listed(t, out); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("the build wrote %v, want exactly %v", got, want)
	}

	t.Run("each script carries its own entry and nothing of a development build", func(t *testing.T) {
		for _, name := range []string{"embed", "deleteme", "last-comments", "remark"} {
			js := read(t, filepath.Join(out, name+".mjs"))
			mustContain(t, name+".mjs", js, name+" marker")
			mustNotContain(t, name+".mjs", js, "process.env", "development-only marker", "preact debug marker", "sourceMappingURL")
		}
	})

	t.Run("the server's placeholder survives for it to fill in", func(t *testing.T) {
		mustContain(t, "remark.mjs", read(t, filepath.Join(out, "remark.mjs")), "{% REMARK_URL %}")
	})

	t.Run("a script is a classic script: no module syntax, and its variables stay its own", func(t *testing.T) {
		declares := regexp.MustCompile(`^"use strict";\s*(var|let|const|function|class)\b`)
		for _, name := range []string{"embed", "deleteme", "last-comments", "remark"} {
			js := read(t, filepath.Join(out, name+".mjs"))
			if !strings.HasPrefix(js, `"use strict";`) || declares.MatchString(js) {
				t.Errorf("%s.mjs declares names in the page's own scope: %.80s", name, js)
			}
			mustNotContain(t, name+".mjs", js, "import ", "import{", "export ", "export{")
		}
		// the one fixture script with variables of its own keeps them inside a function
		if js := read(t, filepath.Join(out, "remark.mjs")); !strings.HasPrefix(js, `"use strict";(()=>{`) {
			t.Errorf("remark.mjs is not wrapped in a function: %.80s", js)
		}
	})

	t.Run("a script's own names are shortened", func(t *testing.T) {
		mustNotContain(t, "remark.mjs", read(t, filepath.Join(out, "remark.mjs")), "instance", "title")
	})

	t.Run("the stylesheet is written down to old browsers, one selector per parent", func(t *testing.T) {
		for _, name := range []string{"remark", "last-comments"} {
			css := read(t, filepath.Join(out, name+".css"))
			mustNotContain(t, name+".css", css, "&", ":is(", "inset:", "width>=", "width >=")
			mustContain(t, name+".css", css, "top:0;right:0;bottom:0;left:0", "(min-width:448px)", ":hover{color:red}")
			// two parents, two selectors, each with its own weight
			if strings.Count(css, "{color:green}") != 1 || !regexp.MustCompile(`[a-z_]+ \.[a-z_]+,\.[a-z_]+\.[a-z_]+ \.[a-z_]+\{color:green\}`).MatchString(css) {
				t.Errorf("%s.css does not list the nested rule once per parent:\n%s", name, css)
			}
		}
	})

	t.Run("the custom properties come first", func(t *testing.T) {
		css := read(t, filepath.Join(out, "remark.css"))
		if !strings.HasPrefix(css, ":root{--fixture-color:") {
			t.Errorf("remark.css does not start with the custom properties:\n%s", css)
		}
	})

}

func TestBuildClassNames(t *testing.T) {
	app := writeFixture(t)
	out := filepath.Join(app, "public")
	if err := build(app, out); err != nil {
		t.Fatal(err)
	}

	t.Run("module classes ship under names nothing else on a page uses", func(t *testing.T) {
		for _, name := range []string{"remark", "last-comments"} {
			css, js := read(t, filepath.Join(out, name+".css")), read(t, filepath.Join(out, name+".mjs"))
			mustNotContain(t, name+".css", css, "fixtureTitle", "fixtureInner", "fixtureSpin", "fixtureComposed", classMarker)
			mustNotContain(t, name+".mjs", js, "fixtureInner\"", "_fixture", classMarker)

			for _, class := range classSelectors(css) {
				switch {
				case class == "dark" || class == "p":
					// written as global in the source and left alone
				case !shippedClass.MatchString(class):
					t.Errorf("%s.css ships the class %q: a module class has to be letters, an underscore and letters", name, class)
				case !strings.Contains(js, class):
					t.Errorf("%s.css ships the class %q, which %s.mjs never names", name, class, name)
				}
			}
		}
	})

	t.Run("a class a highlighter or a host page uses stays theirs", func(t *testing.T) {
		css := read(t, filepath.Join(out, "last-comments.css"))
		if classOf(t, css, "color:gray") != "p" || strings.Count(css, ".p{") != 1 || strings.Count(css, ".p ") != 0 {
			t.Errorf("the global .p is not the only rule for that name:\n%s", css)
		}
	})

	t.Run("the same class name in two modules ships as two names", func(t *testing.T) {
		css := read(t, filepath.Join(out, "remark.css"))
		colored, animated := classOf(t, css, "color:var(--fixture-color)"), classOf(t, css, "animation:")
		if colored == animated {
			t.Errorf("both fixtureTitle classes ship as %q", colored)
		}
	})

	t.Run("an animation is renamed where it is defined and where it is used", func(t *testing.T) {
		css := read(t, filepath.Join(out, "remark.css"))
		m := regexp.MustCompile(`@keyframes ([a-z_]+)\{`).FindStringSubmatch(css)
		if len(m) < 2 || !shippedClass.MatchString(m[1]) || !regexp.MustCompile(`animation:[^;}]*\b`+m[1]+`\b`).MatchString(css) {
			t.Errorf("the animation is not under one shipped name:\n%s", css)
		}
	})

	t.Run("a composed class is handed to the script as both names", func(t *testing.T) {
		css, js := read(t, filepath.Join(out, "remark.css")), read(t, filepath.Join(out, "remark.mjs"))
		composed := classOf(t, css, "margin:0")
		if !regexp.MustCompile(`"[a-z]+_[a-z]+ ` + composed + `"|"` + composed + ` [a-z]+_[a-z]+"`).MatchString(js) {
			t.Errorf("remark.mjs does not pair %q with the class it composes:\n%s", composed, js)
		}
	})

	t.Run("a class composing a global one is handed over with the global name as written", func(t *testing.T) {
		css, js := read(t, filepath.Join(out, "remark.css")), read(t, filepath.Join(out, "remark.mjs"))
		themed := classOf(t, css, "padding:1px")
		if !strings.Contains(js, `"dark `+themed+`"`) && !strings.Contains(js, `"`+themed+` dark"`) {
			t.Errorf("remark.mjs does not pair %q with the global class dark:\n%s", themed, js)
		}
	})

	t.Run("the same sources build the same files", func(t *testing.T) {
		again := filepath.Join(t.TempDir(), "public")
		if err := build(app, again); err != nil {
			t.Fatal(err)
		}
		for _, name := range listed(t, out) {
			if read(t, filepath.Join(out, name)) != read(t, filepath.Join(again, name)) {
				t.Errorf("%s differs between two builds of the same sources", name)
			}
		}
	})
}

func TestBuildRefusesAGlobalNameWithAShippedForm(t *testing.T) {
	// the first name of the first module ships as a_a, and the second module's start at b_a
	for name, tt := range map[string]struct{ file, extra string }{
		"a class in a selector":          {"app/shared.module.css", ":global(.a_a) { color: blue; }"},
		"in another case":                {"app/shared.module.css", ":global(.A_A) { color: blue; }"},
		"a class only composed":          {"app/shared.module.css", ".fixtureExtra { composes: a_a from global; color: pink; }"},
		"composed, in another case":      {"app/shared.module.css", ".fixtureExtra { composes: A_A from global; color: pink; }"},
		"a class of the second module":   {"app/shared.module.css", ":global(.b_a) { color: blue; }"},
		"an animation of a global sheet": {"app/styles/custom-properties.css", "@keyframes a_a { to { opacity: 1; } }"},
		"a class of a global sheet":      {"app/styles/custom-properties.css", ".a_b { color: blue; }"},
	} {
		t.Run(name, func(t *testing.T) {
			app := writeFixture(t)
			path := filepath.Join(app, filepath.FromSlash(tt.file))
			if err := os.WriteFile(path, []byte(fixture[tt.file]+"\n"+tt.extra+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			err := build(app, filepath.Join(app, "public"))
			if err == nil || !strings.Contains(err.Error(), "has the form of the shipped name") {
				t.Fatalf("built with: %v", err)
			}
			// the name as the source has it, which is what there is to search for
			if written := regexp.MustCompile(`[aAb]_[aAb]`).FindString(tt.extra); !strings.Contains(err.Error(), `"`+written+`"`) {
				t.Errorf("the refusal does not name %q: %v", written, err)
			}
		})
	}
}

// what esbuild has to say about a CSS module has to name the module's own file, although a
// production build bundles it under another name
func TestBuildMessagesNameTheSourceFile(t *testing.T) {
	app := writeFixture(t)
	path := filepath.Join(app, "app", "other", "shared.module.css")
	if err := os.WriteFile(path, []byte(fixture["app/other/shared.module.css"]+"\n.broken {{ }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// reached through a link, as a checkout under a linked directory is: esbuild reports the file
	// by its real path either way
	link := filepath.Join(t.TempDir(), "widget")
	if err := os.Symlink(app, link); err != nil {
		t.Fatal(err)
	}
	app = link

	opts := options(app, false, newModuleNames(app))
	opts.LogLevel = api.LogLevelSilent
	result := api.Build(opts)
	if len(result.Warnings) == 0 || result.Warnings[0].Location == nil {
		t.Fatalf("no located warning for a stylesheet that does not parse: %+v", result.Warnings)
	}
	file := result.Warnings[0].Location.File
	if !strings.Contains(file, "app/other/") || !strings.HasSuffix(file, ".shared.module.css") || strings.Contains(file, "..") {
		t.Errorf("the warning is located in %q, which does not say app/other/shared.module.css", file)
	}
}

func TestBundledAs(t *testing.T) {
	names := &moduleNames{appDir: filepath.FromSlash("/widget")}
	first := names.bundledAs(filepath.FromSlash("/widget/app/components/auth/auth.module.css"))
	second := names.bundledAs(filepath.FromSlash("/widget/app/other/auth.module.css"))
	// the marker leads the file name, where the class names are taken from, and the directory
	// stays, so a message about the file still says which one
	if first != "app/components/auth/"+classMarker+"0.auth.module.css" || second != "app/other/"+classMarker+"1.auth.module.css" {
		t.Errorf("got %q and %q", first, second)
	}
	if again := names.bundledAs(filepath.FromSlash("/widget/app/components/auth/auth.module.css")); again != first {
		t.Errorf("the same file is bundled as %q and as %q", first, again)
	}
}

func TestLetters(t *testing.T) {
	for n, want := range map[int]string{0: "a", 25: "z", 26: "aa", 27: "ab", 51: "az", 52: "ba", 701: "zz", 702: "aaa"} {
		if got := letters(n); got != want {
			t.Errorf("letters(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestBuildDevelopment(t *testing.T) {
	files, err := compile(writeFixture(t), true)
	if err != nil {
		t.Fatal(err)
	}

	remark := string(files["remark.mjs"])
	mustContain(t, "remark.mjs", remark, "preact debug marker", "sourceMappingURL=remark.mjs.map")
	// NODE_ENV read as "development": the comparison is settled at build time, in favor of the
	// branch that takes the address the page was loaded from
	mustContain(t, "remark.mjs", remark, `true ? "development-only marker"`)
	mustNotContain(t, "remark.mjs", remark, "process.env")
	if strings.Index(remark, "preact debug marker") > strings.Index(remark, "remark marker") {
		t.Error("preact/debug has to run before the widget does")
	}

	// only an entry with packages gets it
	mustNotContain(t, "last-comments.mjs", string(files["last-comments.mjs"]), "preact debug marker")
	mustNotContain(t, "embed.mjs", string(files["embed.mjs"]), "preact debug marker")
	mustNotContain(t, "deleteme.mjs", string(files["deleteme.mjs"]), "preact debug marker")

	// readable names, after the file and the class
	mustContain(t, "remark.css", string(files["remark.css"]), "shared_fixtureTitle")
	if len(files["remark.mjs.map"]) == 0 {
		t.Errorf("no source map among %d files", len(files))
	}
}

func TestBuildFailsOnBrokenSource(t *testing.T) {
	app := writeFixture(t)
	out := filepath.Join(app, "public")
	if err := build(app, out); err != nil {
		t.Fatal(err)
	}
	before := read(t, filepath.Join(out, "embed.mjs"))

	if err := os.WriteFile(filepath.Join(app, "app", "embed.ts"), []byte("const = ;"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := build(app, out); err == nil {
		t.Fatal("a source that does not parse built without an error")
	}
	if read(t, filepath.Join(out, "embed.mjs")) != before {
		t.Error("a build that failed replaced the files of the one before it")
	}
}

func TestBuildRefusesAPackageInAnEntryWrittenWithoutThem(t *testing.T) {
	// the same import in each entry, which only remark may make
	const imported = "import 'preact/debug';\n"

	for _, tc := range []struct {
		entry, file string
		refused     bool
	}{
		{"last-comments", "app/last-comments.js", true},
		{"embed", "app/embed.ts", true},
		{"deleteme", "app/deleteme.ts", true},
		{"remark", "app/remark.tsx", false},
	} {
		for _, dev := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s, development build %v", tc.entry, dev), func(t *testing.T) {
				app := writeFixture(t)
				file := filepath.Join(app, filepath.FromSlash(tc.file))
				if err := os.WriteFile(file, []byte(imported+read(t, file)), 0o600); err != nil {
					t.Fatal(err)
				}

				var err error
				if dev {
					err = devBuild(t, app, tc.entry+".mjs")
				} else {
					_, err = compile(app, false)
				}
				if !tc.refused {
					if err != nil {
						t.Fatalf("an entry with packages did not build: %v", err)
					}
					return
				}
				if err == nil {
					t.Fatal("the build bundled a package into an entry written without them")
				}
				want := tc.entry + ".mjs bundles node_modules/preact/debug.js"
				if !strings.Contains(err.Error(), want) || strings.Count(err.Error(), " bundles ") != 1 {
					t.Errorf("the error has to name the one output and the file, %q, got: %v", want, err)
				}
			})
		}
	}
}

// devBuild builds app the way the development server does, after a first build of the fixture as
// it is, and returns what esbuild reported about the second. A build the server refuses must
// leave the first one being served, which is checked on the script named watched.
func devBuild(t *testing.T, app, watched string) error {
	t.Helper()
	clean := writeFixture(t)
	latest := newBuilt()

	first := options(clean, true, nil)
	first.Plugins = append(first.Plugins, servePlugin(latest, first.Outdir))
	first.LogLevel = api.LogLevelSilent
	if result := api.Build(first); len(result.Errors) > 0 {
		t.Fatalf("the fixture did not build: %v", result.Errors)
	}
	served, ok := latest.file(watched)
	if !ok {
		t.Fatalf("the first build served no %s", watched)
	}

	second := options(app, true, nil)
	second.Plugins = append(second.Plugins, servePlugin(latest, second.Outdir))
	second.LogLevel = api.LogLevelSilent
	result := api.Build(second)
	if len(result.Errors) == 0 {
		return nil
	}
	if now, _ := latest.file(watched); !bytes.Equal(now, served) {
		t.Error("a refused build replaced what the server serves")
	}
	texts := make([]string, 0, len(result.Errors))
	for _, m := range result.Errors {
		texts = append(texts, m.Text)
	}
	return errors.New(strings.Join(texts, "; "))
}

func TestDevHandler(t *testing.T) {
	var seenHost, seenPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHost, seenPath = r.Host, r.URL.RequestURI()
		_, _ = io.WriteString(w, "from the backend")
	}))
	defer backend.Close()
	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	assets := t.TempDir()
	for name, content := range map[string]string{"both.html": "from the assets", "index.html": "demo page", "iframe.html": "widget document"} {
		if err := os.WriteFile(filepath.Join(assets, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	latest := newBuilt()
	srv := httptest.NewServer(devHandler(latest, assets, backendURL))
	defer srv.Close()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()

	get := func(t *testing.T, path string) (int, string, http.Header) {
		t.Helper()
		resp, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close() //nolint:errcheck // test
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, string(body), resp.Header
	}

	t.Run("a request for the widget waits for the first build", func(t *testing.T) {
		answered := make(chan string, 1)
		go func() {
			resp, err := client.Get(srv.URL + "/web/remark.mjs")
			if err != nil {
				answered <- err.Error()
				return
			}
			defer resp.Body.Close() //nolint:errcheck // test
			body, _ := io.ReadAll(resp.Body)
			answered <- string(body)
		}()

		select {
		case body := <-answered:
			t.Fatalf("answered %q before any build had finished", body)
		case <-time.After(100 * time.Millisecond):
		}

		latest.take(map[string][]byte{"remark.mjs": []byte("built bundle"), "both.html": []byte("from the build")}, false)
		if body := <-answered; body != "built bundle" {
			t.Errorf("answered %q once the build was there", body)
		}
	})

	tests := []struct {
		name, path string
		code       int
		body       string
	}{
		{"the build is served under /web/", "/web/remark.mjs", http.StatusOK, "built bundle"},
		{"a name in both comes from the build", "/web/both.html", http.StatusOK, "from the build"},
		{"a name the build lacks comes from the assets", "/web/iframe.html", http.StatusOK, "widget document"},
		{"the directory answers with the demo page", "/web/", http.StatusOK, "demo page"},
		{"a missing name is not found", "/web/missing.html", http.StatusNotFound, "404 page not found\n"},
		{"the api goes to the backend", "/api/v1/config?site=remark", http.StatusOK, "from the backend"},
		{"auth goes to the backend", "/auth/status", http.StatusOK, "from the backend"},
		{"anything else is not found", "/other", http.StatusNotFound, "404 page not found\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, body, _ := get(t, tt.path)
			if code != tt.code || body != tt.body {
				t.Errorf("GET %s = %d %q, want %d %q", tt.path, code, body, tt.code, tt.body)
			}
		})
	}

	t.Run("a path cannot climb out of the served directories", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(filepath.Dir(assets), "outside.txt"), []byte("outside the assets"), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/web/../outside.txt", "/web/%2e%2e/outside.txt", "/web/..%2foutside.txt"} {
			if code, body, _ := get(t, path); code == http.StatusOK || strings.Contains(body, "outside the assets") {
				t.Errorf("GET %s = %d %q", path, code, body)
			}
		}
	})

	t.Run("the backend is asked under its own name, with the path and query intact", func(t *testing.T) {
		get(t, "/api/v1/find?site=remark&url=x")
		if seenHost != backendURL.Host || seenPath != "/api/v1/find?site=remark&url=x" {
			t.Errorf("the backend saw host %q and %q, want %q and the request as sent", seenHost, seenPath, backendURL.Host)
		}
	})

	t.Run("the root leads to the demo page", func(t *testing.T) {
		code, _, header := get(t, "/")
		if code != http.StatusFound || header.Get("Location") != "/web/" {
			t.Errorf("GET / = %d to %q, want a redirect to /web/", code, header.Get("Location"))
		}
	})

	t.Run("a widget file can be read from another origin and is revalidated", func(t *testing.T) {
		_, _, header := get(t, "/web/remark.mjs")
		if header.Get("Access-Control-Allow-Origin") != "*" || header.Get("Cache-Control") != "no-cache" {
			t.Errorf("headers %v", header)
		}
	})

}

func TestDevHandlerRebuilds(t *testing.T) {
	assets := t.TempDir()
	if err := os.WriteFile(filepath.Join(assets, "both.html"), []byte("from the assets"), 0o600); err != nil {
		t.Fatal(err)
	}
	latest := newBuilt()
	latest.take(map[string][]byte{"remark.mjs": []byte("built bundle"), "both.html": []byte("from the build")}, false)
	srv := httptest.NewServer(devHandler(latest, assets, &url.URL{Scheme: "http", Host: "127.0.0.1:1"}))
	defer srv.Close()
	client := &http.Client{}
	defer client.CloseIdleConnections()

	get := func(t *testing.T, path string) (int, string, http.Header) {
		t.Helper()
		resp, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close() //nolint:errcheck // test
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, string(body), resp.Header
	}

	t.Run("a reload is never told a built file has not changed", func(t *testing.T) {
		// two builds fit in one second, which is as fine as a modification time gets
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/web/remark.mjs", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("If-Modified-Since", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close() //nolint:errcheck // test
		if resp.StatusCode != http.StatusOK || resp.Header.Get("Last-Modified") != "" {
			t.Errorf("GET with If-Modified-Since = %d, Last-Modified %q", resp.StatusCode, resp.Header.Get("Last-Modified"))
		}
	})

	t.Run("a change that does not compile leaves the last build being served", func(t *testing.T) {
		latest.take(nil, true)
		if code, body, _ := get(t, "/web/remark.mjs"); code != http.StatusOK || body != "built bundle" {
			t.Errorf("GET /web/remark.mjs = %d %q after a failed rebuild", code, body)
		}
	})

	t.Run("a rebuild replaces what is served", func(t *testing.T) {
		latest.take(map[string][]byte{"remark.mjs": []byte("rebuilt bundle")}, false)
		if _, body, _ := get(t, "/web/remark.mjs"); body != "rebuilt bundle" {
			t.Errorf("GET /web/remark.mjs = %q after a rebuild", body)
		}
		if _, body, _ := get(t, "/web/both.html"); body != "from the assets" {
			t.Errorf("GET /web/both.html = %q once the build no longer has it", body)
		}
	})
}

func TestReadEnvFile(t *testing.T) {
	t.Run("a missing file is empty", func(t *testing.T) {
		env, err := readEnvFile(filepath.Join(t.TempDir(), ".env"))
		if err != nil || len(env) != 0 {
			t.Errorf("got %v, %v", env, err)
		}
	})

	t.Run("values, comments and quotes", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".env")
		content := strings.Join([]string{
			"# local settings",
			"PORT=9001 # the port the demo page is on",
			"",
			`export REMARK_API_BASE_URL="http://127.0.0.1:8081"`,
			"QUOTED='a # b' # and a comment",
			"TICKED=`9002` # as node reads it",
			"not a pair",
			" SPACED = x ",
			"EMPTY=",
		}, "\n")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		env, err := readEnvFile(path)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{"PORT": "9001", "REMARK_API_BASE_URL": "http://127.0.0.1:8081", "QUOTED": "a # b", "TICKED": "9002", "SPACED": "x", "EMPTY": ""}
		if len(env) != len(want) {
			t.Fatalf("got %v, want %v", env, want)
		}
		for k, v := range want {
			if env[k] != v {
				t.Errorf("%s = %q, want %q", k, env[k], v)
			}
		}
	})
}
