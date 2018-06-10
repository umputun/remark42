package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	"github.com/umputun/remark/app/rest/api"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/rest/cache"
	"github.com/umputun/remark/app/rest/proxy"
)

// Opts with command line flags and env
// nolint:maligned
type Opts struct {
	BoltPath   string   `long:"bolt" env:"BOLTDB_PATH" default:"./var" description:"parent dir for bolt files"`
	Sites      []string `long:"site" env:"SITE" default:"remark" description:"site names" env-delim:","`
	RemarkURL  string   `long:"url" env:"REMARK_URL" required:"true" description:"url to remark"`
	Admins     []string `long:"admin" env:"ADMIN" description:"admin(s) names" env-delim:","`
	AdminEmail string   `long:"admin-email" env:"ADMIN_EMAIL" default:"" description:"admin email"`
	DevPasswd  string   `long:"dev-passwd" env:"DEV_PASSWD" default:"" description:"development mode password"`
	Dbg        bool     `long:"dbg" env:"DEBUG" description:"debug mode"`

	BackupLocation string `long:"backup" env:"BACKUP_PATH" default:"./var/backup" description:"backups location"`
	MaxBackupFiles int    `long:"max-back" env:"MAX_BACKUP_FILES" default:"10" description:"max backups to keep"`
	AvatarStore    string `long:"avatars" env:"AVATAR_STORE" default:"./var/avatars" description:"avatars location"`
	ImageProxy     bool   `long:"img-proxy" env:"IMG_PROXY" description:"enable image proxy"`

	MaxCommentSize int `long:"max-comment" env:"MAX_COMMENT_SIZE" default:"2048" description:"max comment size"`
	MaxCachedItems int `long:"max-cache-items" env:"MAX_CACHE_ITEMS" default:"1000" description:"max cached items"`
	MaxCachedValue int `long:"max-cache-value" env:"MAX_CACHE_VALUE" default:"65536" description:"max size of cached value"`
	MaxCacheSize   int `long:"max-cache-size" env:"MAX_CACHE_SIZE" default:"50000000" description:"max size of total cache"`

	SecretKey     string `long:"secret" env:"SECRET" required:"true" description:"secret key"`
	LowScore      int    `long:"low-score" env:"LOW_SCORE" default:"-5" description:"low score threshold"`
	CriticalScore int    `long:"critical-score" env:"CRITICAL_SCORE" default:"-10" description:"critical score threshold"`
	ReadOnlyAge   int    `long:"read-age" env:"READONLY_AGE" default:"0" description:"read-only age of comments"`

	Port    int    `long:"port" env:"REMARK_PORT" default:"8080" description:"port"`
	WebRoot string `long:"web-root" env:"REMARK_WEB_ROOT" default:"./web" description:"web root directory"`

	Auth struct {
		Google   AuthGroup `group:"google" namespace:"google" env-namespace:"GOOGLE" description:"Google OAuth"`
		Github   AuthGroup `group:"github" namespace:"github" env-namespace:"GITHUB" description:"Github OAuth"`
		Facebook AuthGroup `group:"facebook" namespace:"facebook" env-namespace:"FACEBOOK" description:"Facebook OAuth"`
		Yandex   AuthGroup `group:"yandex" namespace:"yandex" env-namespace:"YANDEX" description:"Yandex OAuth"`
	} `group:"auth" namespace:"auth" env-namespace:"AUTH"`
}

// AuthGroup defines options group for auth params
type AuthGroup struct {
	CID  string `long:"cid" env:"CID" description:"OAuth client ID"`
	CSEC string `long:"csec" env:"CSEC" description:"OAuth client secret"`
}

var revision = "unknown"

// Application holds all active objects
type Application struct {
	Opts
	restSrv     *api.Rest
	migratorSrv *api.Migrator
	exporter    migrator.Exporter
	terminated  chan struct{}
}

func main() {
	fmt.Printf("remark %s\n", revision)

	var opts Opts
	p := flags.NewParser(&opts, flags.Default)
	if _, e := p.ParseArgs(os.Args[1:]); e != nil {
		os.Exit(1)
	}
	log.Print("[INFO] started remark")
	resetEnv("SECRET", "AUTH_GOOGLE_CSEC", "AUTH_GITHUB_CSEC", "AUTH_FACEBOOK_CSEC", "AUTH_YANDEX_CSEC")
	ctx, cancel := context.WithCancel(context.Background())
	go func() { // catch signal and invoke graceful termination
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
	err = app.Run(ctx)
	log.Printf("[INFO] remark terminated %s", err)
}

// New prepares application and return it with all active parts
// doesn't start anything
func New(opts Opts) (*Application, error) {
	setupLog(opts.Dbg)

	if err := makeDirs(opts.BoltPath, opts.BackupLocation, opts.AvatarStore); err != nil {
		return nil, err
	}

	if !strings.HasPrefix(opts.RemarkURL, "http://") && !strings.HasPrefix(opts.RemarkURL, "https://") {
		return nil, errors.Errorf("invalid remark42 url %s", opts.RemarkURL)
	}

	boltStore, err := makeBoltStore(opts.Sites, opts.BoltPath)
	if err != nil {
		return nil, err
	}
	dataService := &service.DataStore{
		Interface:      boltStore,
		EditDuration:   5 * time.Minute,
		Secret:         opts.SecretKey,
		MaxCommentSize: opts.MaxCommentSize,
	}

	loadingCache, err := cache.NewLoadingCache(cache.MaxValSize(opts.MaxCachedValue), cache.MaxKeys(opts.MaxCachedItems),
		cache.PostFlushFn(postFlushFn(opts.Sites, opts.Port)))
	if err != nil {
		return nil, err
	}

	jwtService := auth.NewJWT(opts.SecretKey, strings.HasPrefix(opts.RemarkURL, "https://"), 7*24*time.Hour)

	avatarProxy := &proxy.Avatar{
		Store:     proxy.NewFSAvatarStore(opts.AvatarStore),
		RoutePath: "/api/v1/avatar",
		RemarkURL: strings.TrimSuffix(opts.RemarkURL, "/"),
	}

	exporter := &migrator.Remark{DataStore: dataService}

	migr := &api.Migrator{
		Version:        revision,
		Cache:          loadingCache,
		NativeImporter: &migrator.Remark{DataStore: dataService},
		DisqusImporter: &migrator.Disqus{DataStore: dataService},
		NativeExported: &migrator.Remark{DataStore: dataService},
		SecretKey:      opts.SecretKey,
	}

	srv := &api.Rest{
		Version:     revision,
		DataService: dataService,
		Exporter:    exporter,
		WebRoot:     opts.WebRoot,
		RemarkURL:   opts.RemarkURL,
		ImageProxy:  &proxy.Image{Enabled: opts.ImageProxy, RoutePath: "/api/v1/img", RemarkURL: opts.RemarkURL},
		AvatarProxy: avatarProxy,
		ReadOnlyAge: opts.ReadOnlyAge,
		Authenticator: auth.Authenticator{
			JWTService: jwtService,
			Admins:     opts.Admins,
			AdminEmail: opts.AdminEmail,
			Providers:  makeAuthProviders(jwtService, avatarProxy, dataService, opts),
			DevPasswd:  opts.DevPasswd,
		},
		Cache: loadingCache,
	}

	// no admin email, use admin@domain
	if srv.Authenticator.AdminEmail == "" {
		if u, err := url.Parse(opts.RemarkURL); err == nil {
			srv.Authenticator.AdminEmail = "admin@" + u.Host
		}
	}

	srv.ScoreThresholds.Low, srv.ScoreThresholds.Critical = opts.LowScore, opts.CriticalScore
	tch := make(chan struct{})
	return &Application{restSrv: srv, migratorSrv: migr, exporter: exporter, Opts: opts, terminated: tch}, nil
}

// Run all application objects
func (a *Application) Run(ctx context.Context) error {
	if a.DevPasswd != "" {
		log.Printf("[WARN] running in dev mode")
	}

	go func() {
		// shutdown on context cancellation
		<-ctx.Done()
		a.restSrv.Shutdown()
		a.migratorSrv.Shutdown()
	}()
	a.activateBackup(ctx) // runs in goroutine for each site
	go a.migratorSrv.Run(a.Port + 1)
	a.restSrv.Run(a.Port)
	close(a.terminated)
	return nil
}

// Wait for application completion (termination)
func (a *Application) Wait() {
	<-a.terminated
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
func makeBoltStore(siteNames []string, path string) (engine.Interface, error) {
	sites := []engine.BoltSite{}
	for _, site := range siteNames {
		sites = append(sites, engine.BoltSite{SiteID: site, FileName: fmt.Sprintf("%s/%s.db", path, site)})
	}
	result, err := engine.NewBoltDB(bolt.Options{Timeout: 30 * time.Second}, sites...)
	if err != nil {
		return nil, errors.Wrap(err, "can't initialize data store")
	}
	return result, nil
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

func makeAuthProviders(jwtService *auth.JWT, avatarProxy *proxy.Avatar, ds *service.DataStore, opts Opts) []auth.Provider {

	makeParams := func(cid, secret string) auth.Params {
		return auth.Params{
			JwtService:   jwtService,
			AvatarProxy:  avatarProxy,
			RemarkURL:    opts.RemarkURL,
			Cid:          cid,
			Csecret:      secret,
			Admins:       opts.Admins,
			SecretKey:    opts.SecretKey,
			IsVerifiedFn: ds.IsVerifiedFn(),
		}
	}

	providers := []auth.Provider{}
	if opts.Auth.Google.CID != "" && opts.Auth.Google.CSEC != "" {
		providers = append(providers, auth.NewGoogle(makeParams(opts.Auth.Google.CID, opts.Auth.Google.CSEC)))
	}
	if opts.Auth.Github.CID != "" && opts.Auth.Github.CSEC != "" {
		providers = append(providers, auth.NewGithub(makeParams(opts.Auth.Github.CID, opts.Auth.Github.CSEC)))
	}
	if opts.Auth.Facebook.CID != "" && opts.Auth.Facebook.CSEC != "" {
		providers = append(providers, auth.NewFacebook(makeParams(opts.Auth.Facebook.CID, opts.Auth.Facebook.CSEC)))
	}
	if opts.Auth.Yandex.CID != "" && opts.Auth.Yandex.CSEC != "" {
		providers = append(providers, auth.NewYandex(makeParams(opts.Auth.Yandex.CID, opts.Auth.Yandex.CSEC)))
	}
	if len(providers) == 0 {
		log.Printf("[WARN] no auth providers defined")
	}
	return providers
}

// post-flush callback invoked by cache after each flush in async way
func postFlushFn(sites []string, port int) func() {

	return func() {
		// list of heavy urls for pre-heating on cache change
		urls := []string{
			"http://localhost:%d/api/v1/list?site=%s",
			"http://localhost:%d/api/v1/last/50?site=%s",
		}

		for _, site := range sites {
			for _, u := range urls {
				resp, err := http.Get(fmt.Sprintf(u, port, site))
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
}

func resetEnv(envs ...string) {
	for _, env := range envs {
		if err := os.Unsetenv(env); err != nil {
			log.Printf("[WARN] can't unset env %s, %s", env, err)
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
