// Command build compiles the comment widget with esbuild: TypeScript and JSX into scripts, CSS
// modules into stylesheets, both written down to the browsers in engines and minified. It replaces
// the JavaScript bundler, so building the widget needs go and the installed runtime packages and
// nothing else.
//
//	go run .          a production build into the app's public directory
//	go run . -serve   a development build, kept in memory, rebuilt on every change and served
//	                  on -addr with the API proxied to -api-base
package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

const (
	defaultPort    = "9000"
	defaultAPIBase = "http://127.0.0.1:8080"
)

func main() {
	log.SetFlags(0)

	app := flag.String("app", "../apps/remark42", "the widget's directory, holding app, tsconfig.json and node_modules")
	serving := flag.Bool("serve", false, "serve a development build on -addr, rebuilt on change: unminified, with source maps and preact/debug")
	addr := flag.String("addr", "", "address the development server listens on (default 127.0.0.1:$PORT, port "+defaultPort+")")
	apiBase := flag.String("api-base", "", "backend the development server proxies /api and /auth to (default $REMARK_API_BASE_URL, "+defaultAPIBase+")")
	flag.Parse()

	if flag.NArg() > 0 {
		log.Fatalf("unexpected argument %q", flag.Arg(0))
	}

	appDir, err := filepath.Abs(*app)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(appDir, "tsconfig.json")); err != nil {
		log.Fatalf("%s is not the widget's directory: %v", appDir, err)
	}

	if !*serving {
		// always the app's own public directory: the build empties where it writes
		if err = build(appDir, filepath.Join(appDir, "public")); err != nil {
			log.Fatal(err)
		}
		return
	}

	// a local .env may set the two variables read below; one already in the environment wins
	env, err := readEnvFile(filepath.Join(appDir, ".env"))
	if err != nil {
		log.Fatal(err)
	}
	lookup := func(name, fallback string) string {
		if v := os.Getenv(name); v != "" {
			return v
		}
		if v := env[name]; v != "" {
			return v
		}
		return fallback
	}
	if *addr == "" {
		*addr = "127.0.0.1:" + lookup("PORT", defaultPort)
	}
	if *apiBase == "" {
		*apiBase = lookup("REMARK_API_BASE_URL", defaultAPIBase)
	}
	backend, err := url.Parse(*apiBase)
	if err != nil || backend.Scheme == "" || backend.Host == "" {
		log.Fatalf("-api-base %q is not an address like %s", *apiBase, defaultAPIBase)
	}

	assetsDir := filepath.Join(appDir, "..", "..", "..", "backend", "app", "webassets", "assets")
	if err = serve(appDir, assetsDir, *addr, backend); err != nil {
		log.Fatal(fmt.Errorf("development server: %w", err))
	}
}
