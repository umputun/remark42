package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/umputun/remark/backend/app/store/avatar"

	"github.com/coreos/bbolt"
	"github.com/globalsign/mgo"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/migrator"
	"github.com/umputun/remark/backend/app/rest/api"
	"github.com/umputun/remark/backend/app/rest/auth"
	"github.com/umputun/remark/backend/app/rest/cache"
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/backend/app/store/engine/mongo"
	"github.com/umputun/remark/backend/app/store/service"
)

// Opts with command line flags and env
// nolint:maligned
type Opts struct {
	SecretKey string `long:"secret" env:"SECRET" required:"true" description:"secret key"`
	RemarkURL string `long:"url" env:"REMARK_URL" required:"true" description:"url to remark"`

	Store  StoreGroup  `group:"store" namespace:"store" env-namespace:"STORE"`
	Avatar AvatarGroup `group:"avatar" namespace:"avatar" env-namespace:"AVATAR"`
	Cache  CacheGroup  `group:"cache" namespace:"cache" env-namespace:"CACHE"`

	Sites          []string      `long:"site" env:"SITE" default:"remark" description:"site names" env-delim:","`
	Admins         []string      `long:"admin" env:"ADMIN" description:"admin(s) names" env-delim:","`
	AdminEmail     string        `long:"admin-email" env:"ADMIN_EMAIL" default:"" description:"admin email"`
	DevPasswd      string        `long:"dev-passwd" env:"DEV_PASSWD" default:"" description:"development mode password"`
	BackupLocation string        `long:"backup" env:"BACKUP_PATH" default:"./var/backup" description:"backups location"`
	MaxBackupFiles int           `long:"max-back" env:"MAX_BACKUP_FILES" default:"10" description:"max backups to keep"`
	ImageProxy     bool          `long:"img-proxy" env:"IMG_PROXY" description:"enable image proxy"`
	MaxCommentSize int           `long:"max-comment" env:"MAX_COMMENT_SIZE" default:"2048" description:"max comment size"`
	LowScore       int           `long:"low-score" env:"LOW_SCORE" default:"-5" description:"low score threshold"`
	CriticalScore  int           `long:"critical-score" env:"CRITICAL_SCORE" default:"-10" description:"critical score threshold"`
	ReadOnlyAge    int           `long:"read-age" env:"READONLY_AGE" default:"0" description:"read-only age of comments"`
	EditDuration   time.Duration `long:"edit-time" env:"EDIT_TIME" default:"5m" description:"edit window"`
	Port           int           `long:"port" env:"REMARK_PORT" default:"8080" description:"port"`
	WebRoot        string        `long:"web-root" env:"REMARK_WEB_ROOT" default:"./web" description:"web root directory"`
	Dbg            bool          `long:"dbg" env:"DEBUG" description:"debug mode"`

	Auth struct {
		TTL struct {
			JWT    time.Duration `long:"jwt" env:"JWT" default:"5m" description:"jwt TTL"`
			Cookie time.Duration `long:"cookie" env:"COOKIE" default:"200h" description:"auth cookie TTL"`
		} `group:"ttl" namespace:"ttl" env-namespace:"TTL"`
		Google   AuthGroup `group:"google" namespace:"google" env-namespace:"GOOGLE" description:"Google OAuth"`
		Github   AuthGroup `group:"github" namespace:"github" env-namespace:"GITHUB" description:"Github OAuth"`
		Facebook AuthGroup `group:"facebook" namespace:"facebook" env-namespace:"FACEBOOK" description:"Facebook OAuth"`
		Yandex   AuthGroup `group:"yandex" namespace:"yandex" env-namespace:"YANDEX" description:"Yandex OAuth"`
		Dev      bool      `long:"dev" env:"DEV" description:"enable dev (local) oauth2"`
	} `group:"auth" namespace:"auth" env-namespace:"AUTH"`
}

// AuthGroup defines options group for auth params
type AuthGroup struct {
	CID  string `long:"cid" env:"CID" description:"OAuth client ID"`
	CSEC string `long:"csec" env:"CSEC" description:"OAuth client secret"`
}

// StoreGroup defines options group for store params
type StoreGroup struct {
	Type string `long:"type" env:"TYPE" description:"type of storage" choice:"bolt" choice:"mongo" default:"bolt"`
	Bolt struct {
		Path    string        `long:"path" env:"PATH" default:"./var" description:"parent dir for bolt files"`
		Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"30s" description:"bolt timeout"`
	} `group:"bolt" namespace:"bolt" env-namespace:"BOLT"`
	Mongo MongoOpts `group:"mongo" namespace:"mongo" env-namespace:"MONGO"`
}

// AvatarGroup defines options group for avatar params
type AvatarGroup struct {
	Type string `long:"type" env:"TYPE" description:"type of avatar storage" choice:"fs" choice:"mongo" default:"fs"`
	FS   struct {
		Path string `long:"path" env:"PATH" default:"./var/avatars" description:"avatars location"`
	} `group:"fs" namespace:"fs" env-namespace:"FS"`
	Mongo  MongoOpts `group:"mongo" namespace:"mongo" env-namespace:"MONGO"`
	RszLmt int       `long:"rsz-lmt" env:"RESIZE" default:"0" description:"max image size for resizing avatars on save"`
}

// CacheGroup defines options group for cache params
type CacheGroup struct {
	Type string `long:"type" env:"TYPE" description:"type of cache" choice:"mem" choice:"redis" default:"mem"`
	Max  struct {
		Items int   `long:"items" env:"ITEMS" default:"1000" description:"max cached items"`
		Value int   `long:"value" env:"VALUE" default:"65536" description:"max size of cached value"`
		Size  int64 `long:"size" env:"SIZE" default:"50000000" description:"max size of total cache"`
	} `group:"max" namespace:"max" env-namespace:"MAX"`
}

// MongoOpts holds all mongo params
type MongoOpts struct {
	URL    string   `long:"url" env:"URL" description:"mongo url"`
	Server []string `long:"server" env:"SERVER" description:"mongo host:port" env-delim:","`
	DB     string   `long:"db" env:"DB" default:"remark42" description:"mongo database"`
	User   string   `long:"user" env:"USER" default:"" description:"mongo user"`
	Passwd string   `long:"password" env:"PASSWD" default:"" description:"mongo pssword"`
	SSL    bool     `long:"ssl" env:"SSL" description:"connect to mongo with ssl"`
	Dbg    bool     `long:"dbg" env:"DEBUG" description:"enable mongo debug"`
}

var revision = "unknown"

// Application holds all active objects
type Application struct {
	Opts
	restSrv     *api.Rest
	migratorSrv *api.Migrator
	exporter    migrator.Exporter
	devAuth     *auth.DevAuthServer
	dataService *service.DataStore
	terminated  chan struct{}
}

func main() {
	fmt.Printf("remark %s\n", revision)

	var opts Opts
	p := flags.NewParser(&opts, flags.Default)
	if _, e := p.ParseArgs(os.Args[1:]); e != nil {
		os.Exit(1)
	}

	setupLog(opts.Dbg)
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

	if err := makeDirs(opts.BackupLocation); err != nil {
		return nil, err
	}

	if !strings.HasPrefix(opts.RemarkURL, "http://") && !strings.HasPrefix(opts.RemarkURL, "https://") {
		return nil, errors.Errorf("invalid remark42 url %s", opts.RemarkURL)
	}

	storeEngine, err := makeDataStore(opts.Store, opts.Sites)
	if err != nil {
		return nil, err
	}

	dataService := &service.DataStore{
		Interface:      storeEngine,
		EditDuration:   opts.EditDuration,
		Secret:         opts.SecretKey,
		MaxCommentSize: opts.MaxCommentSize,
		Admins:         opts.Admins,
	}

	loadingCache, err := cache.NewMemoryCache(cache.MaxCacheSize(opts.Cache.Max.Size), cache.MaxValSize(opts.Cache.Max.Value),
		cache.MaxKeys(opts.Cache.Max.Items))
	if err != nil {
		return nil, err
	}

	// token TTL is 5 minutes, inactivity interval 7+ days by default
	jwtService := auth.NewJWT(opts.SecretKey, strings.HasPrefix(opts.RemarkURL, "https://"),
		opts.Auth.TTL.JWT, opts.Auth.TTL.Cookie)

	avatarStore, err := makeAvatarStore(opts.Avatar)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make avatar store")
	}
	avatarProxy := &proxy.Avatar{
		Store:     avatarStore,
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

	authProviders := makeAuthProviders(jwtService, avatarProxy, dataService, opts)

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
			JWTService:        jwtService,
			AdminEmail:        opts.AdminEmail,
			Providers:         authProviders,
			DevPasswd:         opts.DevPasswd,
			PermissionChecker: dataService,
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

	var devAuth *auth.DevAuthServer
	if opts.Auth.Dev {
		devAuth = &auth.DevAuthServer{Provider: authProviders[len(authProviders)-1]}
	}

	tch := make(chan struct{})
	return &Application{restSrv: srv, migratorSrv: migr, exporter: exporter, devAuth: devAuth, dataService: dataService,
		Opts: opts, terminated: tch}, nil
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
		if a.devAuth != nil {
			a.devAuth.Shutdown()
		}
		if e := a.dataService.Close(); e != nil {
			log.Printf("[WARN] failed to close store, %s", e)
		}

	}()
	a.activateBackup(ctx)            // runs in goroutine for each site
	go a.migratorSrv.Run(a.Port + 1) // migrator server runs on +1, localhost only
	if a.Auth.Dev {
		go a.devAuth.Run() // dev oauth2 server on :8084
	}
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

// makeDataStore creates store for all sites
func makeDataStore(group StoreGroup, siteNames []string) (result engine.Interface, err error) {
	switch group.Type {
	case "bolt":
		if err = makeDirs(group.Bolt.Path); err != nil {
			return nil, errors.Wrap(err, "failed to create bolt store")
		}
		sites := []engine.BoltSite{}
		for _, site := range siteNames {
			sites = append(sites, engine.BoltSite{SiteID: site, FileName: fmt.Sprintf("%s/%s.db", group.Bolt.Path, site)})
		}
		result, err = engine.NewBoltDB(bolt.Options{Timeout: group.Bolt.Timeout}, sites...)
	case "mongo":
		mgServer, e := makeMongo(group.Mongo)
		if e != nil {
			return result, errors.Wrap(err, "failed to create mongo server")
		}
		conn := mongo.NewConnection(mgServer, group.Mongo.DB, "")
		result, err = engine.NewMongo(conn, 500, 100*time.Millisecond)
	default:
		return nil, errors.Errorf("unsupported store type %s", group.Type)
	}
	return result, errors.Wrap(err, "can't initialize data store")
}

func makeAvatarStore(group AvatarGroup) (avatar.Store, error) {
	switch group.Type {
	case "fs":
		if err := makeDirs(group.FS.Path); err != nil {
			return nil, err
		}
		return avatar.NewLocalFS(group.FS.Path, group.RszLmt), nil
	case "mongo":
		mgServer, err := makeMongo(group.Mongo)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create mongo server")
		}
		conn := mongo.NewConnection(mgServer, group.Mongo.DB, "")
		return avatar.NewGridFS(conn, group.RszLmt), nil
	}
	return nil, errors.Errorf("unsupported avatar store type %s", group.Type)
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

func makeMongo(mopts MongoOpts) (result *mongo.Server, err error) {
	dial := &mgo.DialInfo{Addrs: mopts.Server, Database: mopts.DB} // default with addrs
	if mopts.URL != "" {                                           // use full mongo uri if defined
		var dialError error
		if dial, dialError = mgo.ParseURL(mopts.URL); dialError != nil {
			return result, errors.Wrapf(err, "failed to create mongo server with url %s", mopts.URL)
		}
		log.Printf("[DEBUG] dial %+v", dial)
	}
	dial.Timeout = 3 * time.Second
	return mongo.NewServer(*dial, mongo.ServerParams{
		Debug:      mopts.Dbg,
		SSL:        mopts.SSL,
		Credential: &mgo.Credential{Username: mopts.User, Password: mopts.Passwd, Source: "admin"},
	})
}

func makeAuthProviders(jwtService *auth.JWT, avatarProxy *proxy.Avatar, ds *service.DataStore, opts Opts) []auth.Provider {

	makeParams := func(cid, secret string) auth.Params {
		return auth.Params{
			JwtService:        jwtService,
			AvatarProxy:       avatarProxy,
			RemarkURL:         opts.RemarkURL,
			Cid:               cid,
			Csecret:           secret,
			SecretKey:         opts.SecretKey,
			PermissionChecker: ds,
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
	if opts.Auth.Dev {
		providers = append(providers, auth.NewDev(makeParams("", "")))
	}

	if len(providers) == 0 {
		log.Printf("[WARN] no auth providers defined")
	}
	return providers
}

// resetEnv clears all sensitive env vars
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
