package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/rest/avatar"
	"github.com/umputun/remark/app/rest/common"
	"github.com/umputun/remark/app/store"
)

var opts struct {
	BoltPath  string   `long:"bolt" env:"BOLTDB_PATH" default:"./var" description:"parent dir for bolt files"`
	Sites     []string `long:"site" env:"SITE" default:"remark" description:"site names" env-delim:","`
	RemarkURL string   `long:"url" env:"REMARK_URL" default:"https://remark42.com" description:"url to remark"`
	Admins    []string `long:"admin" env:"ADMIN" description:"admin(s) names" env-delim:","`

	DevMode bool `long:"dev" env:"DEV" description:"development mode, no auth enforced"`
	Dbg     bool `long:"dbg" env:"DEBUG" description:"debug mode"`

	BackupLocation string `long:"backup" env:"BACKUP_PATH" default:"./var" description:"backups location"`
	MaxBackupFiles int    `long:"max-back" env:"MAX_BACKUP_FILES" default:"10" description:"max backups to keep"`

	ServerCommand struct {
		SessionStore string `long:"session" env:"SESSION_STORE" default:"./var" description:"path to session store directory"`
		StoreKey     string `long:"store-key" env:"STORE_KEY" default:"secure-store-key" description:"store key"`

		GoogleCID    string `long:"google-cid" env:"REMARK_GOOGLE_CID" description:"Google OAuth client ID"`
		GoogleCSEC   string `long:"google-csec" env:"REMARK_GOOGLE_CSEC" description:"Google OAuth client secret"`
		GithubCID    string `long:"github-cid" env:"REMARK_GITHUB_CID" description:"Github OAuth client ID"`
		GithubCSEC   string `long:"github-csec" env:"REMARK_GITHUB_CSEC" description:"Github OAuth client secret"`
		FacebookCID  string `long:"facebook-cid" env:"REMARK_FACEBOOK_CID" description:"Facebook OAuth client ID"`
		FacebookCSEC string `long:"facebook-csec" env:"REMARK_FACEBOOK_CSEC" description:"Facebook OAuth client secret"`

		AvatarStore string `long:"avatars" env:"AVATAR_STORE" default:"./var/avatars" description:"path to avatars directory"`
		Port        int    `long:"port" env:"REMARK_PORT" default:"8080" description:"port"`
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

	if err := makeDirs(opts.BoltPath, opts.ServerCommand.SessionStore, opts.BackupLocation, opts.ServerCommand.AvatarStore); err != nil {
		log.Fatalf("[ERROR] can't create directories, %+v", err)
	}

	dataStore := makeBoltStore(opts.Sites)

	if p.Active != nil && p.Command.Find("import") == p.Active {
		// import mode
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

	dataService := store.Service{Interface: dataStore, EditDuration: 5 * time.Minute}
	sessionStore := func() sessions.Store {
		sess := sessions.NewFilesystemStore(opts.ServerCommand.SessionStore, []byte(opts.ServerCommand.StoreKey))
		sess.Options.HttpOnly = true
		sess.Options.Secure = true
		sess.Options.MaxAge = 3600 * 24 * 365
		return sess
	}()

	exporter := migrator.Remark{DataStore: dataStore}

	srv := rest.Server{
		Version:       revision,
		DataService:   dataService,
		SessionStore:  sessionStore,
		Admins:        opts.Admins,
		DevMode:       opts.DevMode,
		Exporter:      &exporter,
		AuthProviders: makeAuthProviders(sessionStore),
		Cache:         common.NewLoadingCache(4*time.Hour, 15*time.Minute, postFlushFn),
		AvatarProxy:   avatar.Proxy{StorePath: opts.ServerCommand.AvatarStore, RoutePath: "/api/v1/avatar"},
	}

	if opts.DevMode {
		log.Printf("[WARN] running in dev mode, no auth!")
	}

	for _, siteID := range opts.Sites {
		go migrator.AutoBackup{
			Exporter:       &exporter,
			BackupLocation: opts.BackupLocation,
			SiteID:         siteID,
			KeepMax:        opts.MaxBackupFiles,
			Duration:       24 * time.Hour,
		}.Do()
	}

	srv.Run(opts.ServerCommand.Port)
}

// makeBoltStore creates store for all sites
func makeBoltStore(siteNames []string) store.Interface {
	sites := []store.BoltSite{}
	for _, site := range siteNames {
		sites = append(sites, store.BoltSite{SiteID: site, FileName: fmt.Sprintf("%s/%s.db", opts.BoltPath, site)})
	}
	result, err := store.NewBoltDB(sites...)
	if err != nil {
		log.Fatalf("[ERROR] can't initialize data store, %+v", err)
	}
	return result
}

// mkdir -p for all dirs
func makeDirs(dirs ...string) error {

	// exists returns whether the given file or directory exists or not
	exists := func(path string) (bool, error) {
		_, err := os.Stat(path)
		if err == nil {
			return true, nil
		}
		if os.IsNotExist(err) {
			return false, nil
		}
		return true, err
	}

	for _, dir := range dirs {
		ex, err := exists(dir)
		if err != nil {
			return errors.Wrapf(err, "can't check directory status for %s", dir)
		}
		if !ex {
			if e := os.MkdirAll(dir, 0700); e != nil {
				return errors.Wrapf(err, "can't make directory %s", dir)
			}
		}
	}
	return nil
}

func makeAuthProviders(sessionStore sessions.Store) []auth.Provider {
	providers := []auth.Provider{}
	srvOpts := opts.ServerCommand
	if srvOpts.GoogleCID != "" && srvOpts.GoogleCSEC != "" {
		providers = append(providers, auth.NewGoogle(auth.Params{
			Cid: srvOpts.GoogleCID, Csecret: srvOpts.GoogleCSEC, SessionStore: sessionStore, RemarkURL: opts.RemarkURL}))
	}
	if srvOpts.GithubCID != "" && srvOpts.GithubCSEC != "" {
		providers = append(providers, auth.NewGithub(auth.Params{
			Cid: srvOpts.GithubCID, Csecret: srvOpts.GithubCSEC, SessionStore: sessionStore, RemarkURL: opts.RemarkURL}))
	}
	if srvOpts.FacebookCID != "" && srvOpts.FacebookCSEC != "" {
		providers = append(providers, auth.NewFacebook(auth.Params{
			Cid: srvOpts.FacebookCID, Csecret: srvOpts.FacebookCSEC, SessionStore: sessionStore, RemarkURL: opts.RemarkURL}))
	}
	if len(providers) == 0 {
		log.Printf("[WARN] no auth providers defined")
	}
	return providers
}

// post-flush callback invoked by cache after each flush in async way
func postFlushFn() {
	for _, site := range opts.Sites {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/list?site=%s", opts.ServerCommand.Port, site))
		if err != nil {
			log.Printf("[WARN] failed to refresh cached list for %s, %s", site, err)
			return
		}
		_ = resp.Body.Close()
	}
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
