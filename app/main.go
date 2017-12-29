package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gorilla/sessions"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/store"
)

var opts struct {
	BoltPath  string   `long:"bolt" env:"BOLTDB_PATH" default:"/tmp" description:"parent dir for bolt files"`
	Sites     []string `long:"site" env:"SITE" default:"remark" description:"site names" env-delim:","`
	RemarkURL string   `long:"url" env:"REMARK_URL" default:"https://remark42.com" description:"url to remark"`
	Admins    []string `long:"admin" env:"ADMIN" default:"umputun@gmail.com" description:"admin(s) names" env-delim:","`

	DevMode bool `long:"dev" env:"DEV" description:"development mode, no auth enforced"`
	Dbg     bool `long:"dbg" env:"DEBUG" description:"debug mode"`

	BackupLocation string `long:"backup" env:"BACKUP_PATH" default:"/tmp" description:"backups location"`

	ServerCommand struct {
		SessionStore string `long:"session" env:"SESSION_STORE" default:"/tmp" description:"path to session store directory"`
		StoreKey     string `long:"store-key" env:"STORE_KEY" default:"secure-store-key" description:"store key"`

		GoogleCID    string `long:"google-cid" env:"REMARK_GOOGLE_CID" description:"Google OAuth client ID"`
		GoogleCSEC   string `long:"google-csec" env:"REMARK_GOOGLE_CSEC" description:"Google OAuth client secret"`
		GithubCID    string `long:"github-cid" env:"REMARK_GITHUB_CID" description:"Github OAuth client ID"`
		GithubCSEC   string `long:"github-csec" env:"REMARK_GITHUB_CSEC" description:"Github OAuth client secret"`
		FacebookCID  string `long:"facebook-cid" env:"REMARK_FACEBOOK_CID" description:"Facebook OAuth client ID"`
		FacebookCSEC string `long:"facebook-csec" env:"REMARK_FACEBOOK_CSEC" description:"Facebook OAuth client secret"`
	} `command:"server" description:"run server"`

	ImportCommand struct {
		Provider  string `long:"provider" default:"disqus" description:"provider type"`
		SiteID    string `long:"site" default:"remark" description:"site ID"`
		InputFile string `long:"file" default:"disqus.xml" description:"input file"`
	} `command:"import" description:"import comments from external sources"`
}

var revision = "unknown"

func main() {
	fmt.Printf("remark %s\n", revision)
	p := flags.NewParser(&opts, flags.Default)
	if _, e := p.ParseArgs(os.Args[1:]); e != nil {
		os.Exit(1)
	}

	setupLog(opts.Dbg)
	log.Print("[INFO] started remark")

	makeDirs()

	dataStore := makeBoltStore()

	if p.Active != nil && p.Command.Find("import") == p.Active {
		params := migrator.ImportParams{
			DataStore: dataStore,
			InputFile: opts.ImportCommand.InputFile,
			Provider:  opts.ImportCommand.Provider,
			SiteID:    opts.ImportCommand.SiteID,
		}
		if err := migrator.ImportComments(params); err != nil {
			log.Fatalf("[ERROR] failed to import, %+v", err)
		}
		return
	}

	dataService := store.Service{Interface: dataStore}
	sessionStore := sessions.NewFilesystemStore(opts.ServerCommand.SessionStore, []byte(opts.ServerCommand.StoreKey))
	exporter := migrator.Remark{DataStore: dataStore}

	srv := rest.Server{
		Version:      revision,
		DataService:  dataService,
		SessionStore: sessionStore,
		Admins:       opts.Admins,
		DevMode:      opts.DevMode,
		Exporter:     &exporter,
		AuthGoogle: auth.NewGoogle(auth.Params{
			Cid:          opts.ServerCommand.GoogleCID,
			Csecret:      opts.ServerCommand.GoogleCSEC,
			SessionStore: sessionStore,
			RemarkURL:    opts.RemarkURL,
		}),
		AuthGithub: auth.NewGithub(auth.Params{
			Cid:          opts.ServerCommand.GithubCID,
			Csecret:      opts.ServerCommand.GithubCSEC,
			SessionStore: sessionStore,
			RemarkURL:    opts.RemarkURL,
		}),
		AuthFacebook: auth.NewFacebook(auth.Params{
			Cid:          opts.ServerCommand.FacebookCID,
			Csecret:      opts.ServerCommand.FacebookCSEC,
			SessionStore: sessionStore,
			RemarkURL:    opts.RemarkURL,
		}),
	}

	if opts.DevMode {
		log.Printf("[WARN] running in dev mode, no auth!")
	}

	go migrator.AutoBackup(&exporter, opts.BackupLocation)
	srv.Run()
}

// makeBoltStore creates store for all sites
func makeBoltStore() store.Interface {
	sites := []store.BoltSite{}
	for _, site := range opts.Sites {
		sites = append(sites, store.BoltSite{SiteID: site, FileName: fmt.Sprintf("%s/%s.db", opts.BoltPath, site)})
	}
	result, err := store.NewBoltDB(sites...)
	if err != nil {
		log.Fatalf("[ERROR] can't initialize data store, %+v", err)
	}
	return result
}

func makeDirs() {
	_ = os.MkdirAll(opts.BoltPath, 700)
	_ = os.MkdirAll(opts.BackupLocation, 700)
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
