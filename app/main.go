package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/bbolt"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"github.com/umputun/remark/app/store/engine"
	"github.com/umputun/remark/app/store/service"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/rest/api"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/rest/proxy"
)

// Opts with command line flags and env
type Opts struct {
	BoltPath  string   `long:"bolt" env:"BOLTDB_PATH" default:"./var" description:"parent dir for bolt files"`
	Sites     []string `long:"site" env:"SITE" default:"remark" description:"site names" env-delim:","`
	RemarkURL string   `long:"url" env:"REMARK_URL" default:"https://remark42.com" description:"url to remark"`
	Admins    []string `long:"admin" env:"ADMIN" description:"admin(s) names" env-delim:","`

	DevPasswd string `long:"dev-passwd" env:"DEV_PASSWD" default:"" description:"development mode password"`
	Dbg       bool   `long:"dbg" env:"DEBUG" description:"debug mode"`

	BackupLocation string `long:"backup" env:"BACKUP_PATH" default:"./var/backup" description:"backups location"`
	MaxBackupFiles int    `long:"max-back" env:"MAX_BACKUP_FILES" default:"10" description:"max backups to keep"`
	AvatarStore    string `long:"avatars" env:"AVATAR_STORE" default:"./var/avatars" description:"avatars location"`
	ImageProxy     bool   `long:"img-proxy" env:"IMG_PROXY" description:"enable image proxy"`

	MaxCommentSize int    `long:"max-comment" env:"MAX_COMMENT_SIZE" default:"2048" description:"max comment size"`
	MaxCachedItems int    `long:"max-cache-items" env:"MAX_CACHE_ITEMS" default:"1000" description:"max cached items"`
	MaxCachedValue int    `long:"max-cache-value" env:"MAX_CACHE_VALUE" default:"65536" description:"max size of cached value"`
	SecretKey      string `long:"secret" env:"SECRET" required:"true" description:"secret key"`
	LowScore       int    `long:"low-score" env:"LOW_SCORE" default:"-5" description:"low score threshold"`
	CriticalScore  int    `long:"critical-score" env:"CRITICAL_SCORE" default:"-10" description:"critical score threshold"`

	GoogleCID    string `long:"google-cid" env:"REMARK_GOOGLE_CID" description:"Google OAuth client ID"`
	GoogleCSEC   string `long:"google-csec" env:"REMARK_GOOGLE_CSEC" description:"Google OAuth client secret"`
	GithubCID    string `long:"github-cid" env:"REMARK_GITHUB_CID" description:"Github OAuth client ID"`
	GithubCSEC   string `long:"github-csec" env:"REMARK_GITHUB_CSEC" description:"Github OAuth client secret"`
	FacebookCID  string `long:"facebook-cid" env:"REMARK_FACEBOOK_CID" description:"Facebook OAuth client ID"`
	FacebookCSEC string `long:"facebook-csec" env:"REMARK_FACEBOOK_CSEC" description:"Facebook OAuth client secret"`
	DisqusCID    string `long:"disqus-cid" env:"REMARK_DISQUS_CID" description:"Disqus OAuth client ID"`
	DisqusCSEC   string `long:"disqus-csec" env:"REMARK_DISQUS_CSEC" description:"Disqus OAuth client secret"`

	Port    int    `long:"port" env:"REMARK_PORT" default:"8080" description:"port"`
	WebRoot string `long:"web-root" env:"REMARK_WEB_ROOT" default:"./web" description:"web root directory"`
}

var opts Opts
var revision = "unknown"

// Application holds all active objects
type Application struct {
	Opts
	srv      api.Rest
	importer api.Import
	exporter migrator.Exporter
}

func main() {
	fmt.Printf("remark %s\n", revision)
	p := flags.NewParser(&opts, flags.Default)
	if _, e := p.ParseArgs(os.Args[1:]); e != nil {
		os.Exit(1)
	}
	log.Print("[INFO] started remark")

	ctx, cancel := context.WithCancel(context.Background())

	// catch signal and invoke graceful termination
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Print("[WARN] interrupt signal")
		cancel()
	}()

	app, err := New(opts)
	if err != nil {
		log.Fatalf("[ERROR] failed to setup application, %+v", err)
	}
	log.Printf("[INFO] remark terminated %s", app.Run(ctx))
}

// New prepares application and return it with all active parts
// doesn't start anything
func New(opts Opts) (*Application, error) {
	setupLog(opts.Dbg)

	if err := makeDirs(opts.BoltPath, opts.BackupLocation, opts.AvatarStore); err != nil {
		return nil, err
	}

	dataService := service.DataStore{
		Interface:      makeBoltStore(opts.Sites, opts.BoltPath),
		EditDuration:   5 * time.Minute,
		Secret:         opts.SecretKey,
		MaxCommentSize: opts.MaxCommentSize,
	}

	cache := rest.NewLoadingCache(rest.MaxValSize(opts.MaxCachedValue), rest.MaxKeys(opts.MaxCachedItems),
		rest.PostFlushFn(postFlushFn))

	jwtService := auth.NewJWT(opts.SecretKey, strings.HasPrefix(opts.RemarkURL, "https://"), 7*24*time.Hour)

	avatarProxy := &proxy.Avatar{
		StorePath: opts.AvatarStore,
		RoutePath: "/api/v1/avatar",
		RemarkURL: strings.TrimSuffix(opts.RemarkURL, "/"),
	}

	exporter := &migrator.Remark{DataStore: &dataService}

	importer := api.Import{
		Version:        revision,
		Cache:          cache,
		NativeImporter: &migrator.Remark{DataStore: &dataService},
		DisqusImporter: &migrator.Disqus{DataStore: &dataService},
		SecretKey:      opts.SecretKey,
	}

	srv := api.Rest{
		Version:     revision,
		DataService: dataService,
		Exporter:    exporter,
		WebRoot:     opts.WebRoot,
		ImageProxy:  proxy.Image{Enabled: opts.ImageProxy, RoutePath: "/api/v1/img", RemarkURL: opts.RemarkURL},
		Authenticator: auth.Authenticator{
			JWTService:  jwtService,
			Admins:      opts.Admins,
			Providers:   makeAuthProviders(jwtService, avatarProxy),
			AvatarProxy: avatarProxy,
			DevPasswd:   opts.DevPasswd,
		},
		Cache: cache,
	}
	srv.ScoreThresholds.Low, srv.ScoreThresholds.Critical = opts.LowScore, opts.CriticalScore
	return &Application{srv: srv, importer: importer, exporter: exporter, Opts: opts}, nil
}

// Run all application objects
func (a *Application) Run(ctx context.Context) error {
	if a.DevPasswd != "" {
		log.Printf("[WARN] running in dev mode")
	}

	a.activateBackup(ctx) // runs in goroutine for each site
	go a.importer.Run(a.Port + 1)
	go a.srv.Run(opts.Port)

	// shutdown on context cancellation
	<-ctx.Done()
	a.srv.Shutdown()
	a.importer.Shutdown()
	return ctx.Err()
}

// activateBackup runs background backups for each site
func (a *Application) activateBackup(ctx context.Context) {
	for _, siteID := range a.Sites {
		backup := migrator.AutoBackup{
			Exporter:       a.exporter,
			BackupLocation: a.BackupLocation,
			SiteID:         siteID,
			KeepMax:        a.MaxBackupFiles,
			Duration:       24 * time.Hour,
		}
		go backup.Do(ctx)
	}
}

// makeBoltStore creates store for all sites
func makeBoltStore(siteNames []string, path string) engine.Interface {
	sites := []engine.BoltSite{}
	for _, site := range siteNames {
		sites = append(sites, engine.BoltSite{SiteID: site, FileName: fmt.Sprintf("%s/%s.db", path, site)})
	}
	result, err := engine.NewBoltDB(bolt.Options{Timeout: 30 * time.Second}, sites...)
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

func makeAuthProviders(jwtService *auth.JWT, avatarProxy *proxy.Avatar) (providers []auth.Provider) {

	makeParams := func(cid, secret string) auth.Params {
		return auth.Params{
			JwtService:  jwtService,
			AvatarProxy: avatarProxy,
			RemarkURL:   opts.RemarkURL,
			Cid:         cid,
			Csecret:     secret,
		}
	}

	if opts.GoogleCID != "" && opts.GoogleCSEC != "" {
		providers = append(providers, auth.NewGoogle(makeParams(opts.GoogleCID, opts.GoogleCSEC)))
	}
	if opts.GithubCID != "" && opts.GithubCSEC != "" {
		providers = append(providers, auth.NewGithub(makeParams(opts.GithubCID, opts.GithubCSEC)))
	}
	if opts.FacebookCID != "" && opts.FacebookCSEC != "" {
		providers = append(providers, auth.NewFacebook(makeParams(opts.FacebookCID, opts.FacebookCSEC)))
	}
	if opts.DisqusCID != "" && opts.DisqusCSEC != "" {
		providers = append(providers, auth.NewDisqus(makeParams(opts.DisqusCID, opts.DisqusCSEC)))
	}
	if len(providers) == 0 {
		log.Printf("[WARN] no auth providers defined")
	}
	return providers
}

// post-flush callback invoked by cache after each flush in async way
func postFlushFn() {

	// list of heavy urls for pre-heating on cache change
	urls := []string{
		"http://localhost:%d/api/v1/list?site=%s",
		"http://localhost:%d/api/v1/last/50?site=%s",
	}

	for _, site := range opts.Sites {
		for _, u := range urls {
			resp, err := http.Get(fmt.Sprintf(u, opts.Port, site))
			if err != nil {
				log.Printf("[WARN] failed to refresh cached list for %s, %s", site, err)
				return
			}
			if err = resp.Body.Close(); err != nil {
				log.Printf("[WARN] failed to close response body, %s", err)
			}
		}
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
