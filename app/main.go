package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gorilla/sessions"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/store"

	"github.com/umputun/remark/app/rest"
)

var opts struct {
	DBFile string `long:"db" env:"BOLTDB_FILE" default:"/tmp/remark.db" description:"bolt file name"`

	SessionStore string `long:"session" env:"SESSION_STORE" default:"/tmp" description:"path to session store directory"`
	StoreKey     string `long:"store-key" env:"STORE_KEY" default:"secure-store-key" description:"store key"`

	GoogleCID  string `long:"google-cid" env:"REMARK_GOOGLE_CID" description:"Google OAuth client ID"`
	GoogleCSEC string `long:"google-csec" env:"REMARK_GOOGLE_CSEC" description:"Google OAuth client secret"`
	GithubCID  string `long:"github-cid" env:"REMARK_GITHUB_CID" description:"Github OAuth client ID"`
	GithubCSEC string `long:"github-csec" env:"REMARK_GITHUB_CSEC" description:"Github OAuth client secret"`

	Admins  []string `long:"admin" env:"ADMIN" default:"umputun@gmail.com" description:"admin(s) names" env-delim:","`
	DevMode bool     `long:"dev" env:"DEV" description:"dev mode mode"`
	Dbg     bool     `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "unknown"

func main() {
	fmt.Printf("remark %s\n", revision)
	if _, err := flags.Parse(&opts); err != nil {
		log.Fatal(err)
	}
	setupLog(opts.Dbg)
	log.Print("[INFO] started remark")

	if opts.DevMode {
		log.Printf("[WARN] running in dev mode, no auth!")
	}

	dataStore, err := store.NewBoltDB(opts.DBFile)
	if err != nil {
		log.Fatalf("[ERROR] can't initialize data store, %+v", err)
	}

	sessionStore := sessions.NewFilesystemStore(opts.SessionStore, []byte(opts.StoreKey))
	srv := rest.Server{
		Version:      revision,
		Store:        dataStore,
		SessionStore: sessionStore,
		Admins:       opts.Admins,
		DevMode:      opts.DevMode,
		AuthGoogle: auth.NewGoogle(auth.Params{
			Cid:          opts.GoogleCID,
			Csecret:      opts.GoogleCSEC,
			SessionStore: sessionStore,
		}),
		AuthGithub: auth.NewGithub(auth.Params{
			Cid:          opts.GithubCID,
			Csecret:      opts.GithubCSEC,
			SessionStore: sessionStore,
		}),
	}

	srv.Run()
}

func setupLog(dbg bool) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stdout,
	}

	log.SetFlags(log.Ldate | log.Ltime)

	if dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		filter.MinLevel = logutils.LogLevel("DEBUG")
	}
	log.SetOutput(filter)
}
