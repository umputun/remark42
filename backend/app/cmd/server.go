package cmd

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/bbolt"
	"github.com/go-pkgz/mongo"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/migrator"
	"github.com/umputun/remark/backend/app/notify"
	"github.com/umputun/remark/backend/app/rest/api"
	"github.com/umputun/remark/backend/app/rest/auth"
	"github.com/umputun/remark/backend/app/rest/cache"
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/avatar"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/backend/app/store/service"
)

// ServerCommand with command line flags and env
type ServerCommand struct {
	Store  StoreGroup  `group:"store" namespace:"store" env-namespace:"STORE"`
	Avatar AvatarGroup `group:"avatar" namespace:"avatar" env-namespace:"AVATAR"`
	Cache  CacheGroup  `group:"cache" namespace:"cache" env-namespace:"CACHE"`
	Mongo  MongoGroup  `group:"mongo" namespace:"mongo" env-namespace:"MONGO"`
	Admin  AdminGroup  `group:"admin" namespace:"admin" env-namespace:"ADMIN"`
	Notify NotifyGroup `group:"notify" namespace:"notify" env-namespace:"NOTIFY"`
	SSL    SSLGroup    `group:"ssl" namespace:"ssl" env-namespace:"SSL"`

	Sites          []string      `long:"site" env:"SITE" default:"remark" description:"site names" env-delim:","`
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

	CommonOpts
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
}

// AvatarGroup defines options group for avatar params
type AvatarGroup struct {
	Type string `long:"type" env:"TYPE" description:"type of avatar storage" choice:"fs" choice:"bolt" choice:"mongo" default:"fs"`
	FS   struct {
		Path string `long:"path" env:"PATH" default:"./var/avatars" description:"avatars location"`
	} `group:"fs" namespace:"fs" env-namespace:"FS"`
	Bolt struct {
		File string `long:"file" env:"FILE" default:"./var/avatars.db" description:"avatars bolt file location"`
	} `group:"bolt" namespace:"bolt" env-namespace:"bolt"`
	RszLmt int `long:"rsz-lmt" env:"RESIZE" default:"0" description:"max image size for resizing avatars on save"`
}

// CacheGroup defines options group for cache params
type CacheGroup struct {
	Type string `long:"type" env:"TYPE" description:"type of cache" choice:"mem" choice:"mongo" choice:"none" default:"mem"`
	Max  struct {
		Items int   `long:"items" env:"ITEMS" default:"1000" description:"max cached items"`
		Value int   `long:"value" env:"VALUE" default:"65536" description:"max size of cached value"`
		Size  int64 `long:"size" env:"SIZE" default:"50000000" description:"max size of total cache"`
	} `group:"max" namespace:"max" env-namespace:"MAX"`
}

// MongoGroup holds all mongo params, used by store, avatar and cache
type MongoGroup struct {
	URL string `long:"url" env:"URL" description:"mongo url"`
	DB  string `long:"db" env:"DB" default:"remark42" description:"mongo database"`
}

// AdminGroup defines options group for admin params
type AdminGroup struct {
	Type   string `long:"type" env:"TYPE" description:"type of admin store" choice:"shared" choice:"mongo" default:"shared"`
	Shared struct {
		Admins []string `long:"id" env:"ID" description:"admin(s) ids" env-delim:","`
		Email  string   `long:"email" env:"EMAIL" default:"" description:"admin email"`
	} `group:"shared" namespace:"shared" env-namespace:"SHARED"`
}

// NotifyGroup defines options for notification
type NotifyGroup struct {
	Type      string `long:"type" env:"TYPE" description:"type of notification" choice:"none" choice:"telegram" default:"none"`
	QueueSize int    `long:"queue" env:"QUEUE" description:"size of notification queue" default:"100"`
	Telegram  struct {
		Token   string        `long:"token" env:"TOKEN" description:"telegram token"`
		Channel string        `long:"chan" env:"CHAN" description:"telegram channel"`
		Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"5s" description:"telegram timeout"`
		API     string        `long:"api" env:"API" default:"https://api.telegram.org/bot" description:"telegram api prefix"`
	} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`
}

// SSLGroup defines options group for server ssl params
type SSLGroup struct {
	Type         string `long:"type" env:"TYPE" description:"ssl (auto)support" choice:"none" choice:"static" choice:"auto" default:"none"`
	Port         int    `long:"port" env:"PORT" description:"port number for https server" default:"8443"`
	Cert         string `long:"cert" env:"CERT" description:"path to cert.pem file"`
	Key          string `long:"key" env:"KEY" description:"path to key.pem file"`
	ACMELocation string `long:"acme-location" env:"ACME_LOCATION" description:"dir where certificates will be stored by autocert manager" default:"./var/acme"`
	ACMEEmail    string `long:"acme-email" env:"ACME_EMAIL" description:"admin email for certificate notifications"`
}

// serverApp holds all active objects
type serverApp struct {
	*ServerCommand
	restSrv       *api.Rest
	migratorSrv   *api.Migrator
	exporter      migrator.Exporter
	devAuth       *auth.DevAuthServer
	dataService   *service.DataStore
	avatarStore   avatar.Store
	notifyService *notify.Service
	terminated    chan struct{}
}

// Execute is the entry point for "server" command, called by flag parser
func (s *ServerCommand) Execute(args []string) error {
	log.Printf("[INFO] start server on port %d", s.Port)
	resetEnv("SECRET", "AUTH_GOOGLE_CSEC", "AUTH_GITHUB_CSEC", "AUTH_FACEBOOK_CSEC", "AUTH_YANDEX_CSEC")

	ctx, cancel := context.WithCancel(context.Background())
	go func() { // catch signal and invoke graceful termination
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Print("[WARN] interrupt signal")
		cancel()
	}()

	app, err := s.newServerApp()
	if err != nil {
		log.Fatalf("[ERROR] failed to setup application, %+v", err)
	}
	if err = app.run(ctx); err != nil {
		log.Printf("[WARN] remark terminated with error %+v", err)
		return err
	}
	log.Printf("[INFO] remark terminated")
	return nil
}

// newServerApp prepares application and return it with all active parts
// doesn't start anything
func (s *ServerCommand) newServerApp() (*serverApp, error) {

	if err := makeDirs(s.BackupLocation); err != nil {
		return nil, err
	}

	if !strings.HasPrefix(s.RemarkURL, "http://") && !strings.HasPrefix(s.RemarkURL, "https://") {
		return nil, errors.Errorf("invalid remark42 url %s", s.RemarkURL)
	}
	log.Printf("[INFO] root url=%s", s.RemarkURL)

	storeEngine, err := s.makeDataStore()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make data store engine")
	}

	adminStore, err := s.makeAdminStore()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make admin store")
	}

	dataService := &service.DataStore{
		Interface:      storeEngine,
		EditDuration:   s.EditDuration,
		AdminStore:     adminStore,
		MaxCommentSize: s.MaxCommentSize,
	}

	loadingCache, err := s.makeCache()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make cache")
	}

	// token TTL is 5 minutes, inactivity interval 7+ days by default
	jwtService := auth.NewJWT(adminStore, strings.HasPrefix(s.RemarkURL, "https://"), s.Auth.TTL.JWT, s.Auth.TTL.Cookie)

	avatarStore, err := s.makeAvatarStore()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make avatar store")
	}
	avatarProxy := &proxy.Avatar{
		Store:     avatarStore,
		RoutePath: "/api/v1/avatar",
		RemarkURL: strings.TrimSuffix(s.RemarkURL, "/"),
	}

	exporter := &migrator.Remark{DataStore: dataService}

	migr := &api.Migrator{
		Cache:             loadingCache,
		NativeImporter:    &migrator.Remark{DataStore: dataService},
		DisqusImporter:    &migrator.Disqus{DataStore: dataService},
		WordPressImporter: &migrator.WordPress{DataStore: dataService},
		NativeExported:    &migrator.Remark{DataStore: dataService},
		KeyStore:          adminStore,
	}

	notifyService, err := s.makeNotify(dataService)
	if err != nil {
		log.Printf("[WARN] failed to make notify service, %s", err)
		notifyService = notify.NopService // disable notifier
	}

	authProviders := s.makeAuthProviders(jwtService, avatarProxy, dataService)
	imgProxy := &proxy.Image{Enabled: s.ImageProxy, RoutePath: "/api/v1/img", RemarkURL: s.RemarkURL}
	commentFormatter := store.NewCommentFormatter(imgProxy)

	sslConfig, err := s.makeSSLConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make config of ssl server params")
	}

	srv := &api.Rest{
		Version:          s.Revision,
		DataService:      dataService,
		WebRoot:          s.WebRoot,
		RemarkURL:        s.RemarkURL,
		ImageProxy:       imgProxy,
		CommentFormatter: commentFormatter,
		AvatarProxy:      avatarProxy,
		Migrator:         migr,
		ReadOnlyAge:      s.ReadOnlyAge,
		SharedSecret:     s.SharedSecret,
		Authenticator: auth.Authenticator{
			JWTService:        jwtService,
			KeyStore:          adminStore,
			Providers:         authProviders,
			DevPasswd:         s.DevPasswd,
			PermissionChecker: dataService,
		},
		Cache:         loadingCache,
		NotifyService: notifyService,
		SSLConfig:     sslConfig,
	}

	srv.ScoreThresholds.Low, srv.ScoreThresholds.Critical = s.LowScore, s.CriticalScore

	var devAuth *auth.DevAuthServer
	if s.Auth.Dev {
		devAuth = &auth.DevAuthServer{Provider: authProviders[len(authProviders)-1]}
	}

	return &serverApp{
		ServerCommand: s,
		restSrv:       srv,
		migratorSrv:   migr,
		exporter:      exporter,
		devAuth:       devAuth,
		dataService:   dataService,
		avatarStore:   avatarStore,
		notifyService: notifyService,
		terminated:    make(chan struct{}),
	}, nil
}

// Run all application objects
func (a *serverApp) run(ctx context.Context) error {
	if a.DevPasswd != "" {
		log.Printf("[WARN] running in dev mode")
	}

	go func() {
		// shutdown on context cancellation
		<-ctx.Done()
		log.Print("[INFO] shutdown initiated")
		a.restSrv.Shutdown()
		if a.devAuth != nil {
			a.devAuth.Shutdown()
		}
		if e := a.dataService.Close(); e != nil {
			log.Printf("[WARN] failed to close data store, %s", e)
		}
		if e := a.avatarStore.Close(); e != nil {
			log.Printf("[WARN] failed to close avatar store, %s", e)
		}
		a.notifyService.Close()
		log.Print("[INFO] shutdown completed")
	}()
	a.activateBackup(ctx) // runs in goroutine for each site
	if a.Auth.Dev {
		go a.devAuth.Run() // dev oauth2 server on :8084
	}
	a.restSrv.Run(a.Port)
	close(a.terminated)
	return nil
}

// Wait for application completion (termination)
func (a *serverApp) Wait() {
	<-a.terminated
}

// activateBackup runs background backups for each site
func (a *serverApp) activateBackup(ctx context.Context) {
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
func (s *ServerCommand) makeDataStore() (result engine.Interface, err error) {
	log.Printf("[INFO] make data store, type=%s", s.Store.Type)

	switch s.Store.Type {
	case "bolt":
		if err = makeDirs(s.Store.Bolt.Path); err != nil {
			return nil, errors.Wrap(err, "failed to create bolt store")
		}
		sites := []engine.BoltSite{}
		for _, site := range s.Sites {
			sites = append(sites, engine.BoltSite{SiteID: site, FileName: fmt.Sprintf("%s/%s.db", s.Store.Bolt.Path, site)})
		}
		result, err = engine.NewBoltDB(bolt.Options{Timeout: s.Store.Bolt.Timeout}, sites...)
	case "mongo":
		mgServer, e := s.makeMongo()
		if e != nil {
			return result, errors.Wrap(e, "failed to create mongo server")
		}
		conn := mongo.NewConnection(mgServer, s.Mongo.DB, "")
		result, err = engine.NewMongo(conn, 500, 100*time.Millisecond)
	default:
		return nil, errors.Errorf("unsupported store type %s", s.Store.Type)
	}
	return result, errors.Wrap(err, "can't initialize data store")
}

func (s *ServerCommand) makeAvatarStore() (avatar.Store, error) {
	log.Printf("[INFO] make avatar store, type=%s", s.Avatar.Type)

	switch s.Avatar.Type {
	case "fs":
		if err := makeDirs(s.Avatar.FS.Path); err != nil {
			return nil, err
		}
		return avatar.NewLocalFS(s.Avatar.FS.Path, s.Avatar.RszLmt), nil
	case "mongo":
		mgServer, err := s.makeMongo()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create mongo server")
		}
		conn := mongo.NewConnection(mgServer, s.Mongo.DB, "")
		return avatar.NewGridFS(conn, s.Avatar.RszLmt), nil
	case "bolt":
		if err := makeDirs(path.Dir(s.Avatar.Bolt.File)); err != nil {
			return nil, err
		}
		return avatar.NewBoltDB(s.Avatar.Bolt.File, bolt.Options{}, s.Avatar.RszLmt)
	}
	return nil, errors.Errorf("unsupported avatar store type %s", s.Avatar.Type)
}

func (s *ServerCommand) makeAdminStore() (admin.Store, error) {
	log.Printf("[INFO] make admin store, type=%s", s.Admin.Type)

	switch s.Admin.Type {
	case "shared":
		if s.Admin.Shared.Email == "" { // no admin email, use admin@domain
			if u, err := url.Parse(s.RemarkURL); err == nil {
				s.Admin.Shared.Email = "admin@" + u.Host
			}
		}
		return admin.NewStaticStore(s.SharedSecret, s.Admin.Shared.Admins, s.Admin.Shared.Email), nil
	case "mongo":
		mgServer, e := s.makeMongo()
		if e != nil {
			return nil, errors.Wrap(e, "failed to create mongo server")
		}
		conn := mongo.NewConnection(mgServer, s.Mongo.DB, "admin")
		return admin.NewMongoStore(conn), nil
	default:
		return nil, errors.Errorf("unsupported admin store type %s", s.Admin.Type)
	}
}

func (s *ServerCommand) makeCache() (cache.LoadingCache, error) {
	log.Printf("[INFO] make cache, type=%s", s.Cache.Type)
	switch s.Cache.Type {
	case "mem":
		return cache.NewMemoryCache(cache.MaxCacheSize(s.Cache.Max.Size), cache.MaxValSize(s.Cache.Max.Value),
			cache.MaxKeys(s.Cache.Max.Items))
	case "mongo":
		mgServer, err := s.makeMongo()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create mongo server")
		}
		conn := mongo.NewConnection(mgServer, s.Mongo.DB, "cache")
		return cache.NewMongoCache(conn, cache.MaxCacheSize(s.Cache.Max.Size), cache.MaxValSize(s.Cache.Max.Value),
			cache.MaxKeys(s.Cache.Max.Items))
	case "none":
		return &cache.Nop{}, nil
	}
	return nil, errors.Errorf("unsupported cache type %s", s.Cache.Type)
}

func (s *ServerCommand) makeMongo() (result *mongo.Server, err error) {
	if s.Mongo.URL == "" {
		return nil, errors.New("no mongo URL provided")
	}
	return mongo.NewServerWithURL(s.Mongo.URL, 10*time.Second)
}

func (s *ServerCommand) makeAuthProviders(jwt *auth.JWT, ap *proxy.Avatar, ds *service.DataStore) []auth.Provider {

	makeParams := func(cid, secret string) auth.Params {
		return auth.Params{
			JwtService:        jwt,
			AvatarProxy:       ap,
			RemarkURL:         s.RemarkURL,
			Cid:               cid,
			Csecret:           secret,
			PermissionChecker: ds,
		}
	}

	providers := []auth.Provider{}
	if s.Auth.Google.CID != "" && s.Auth.Google.CSEC != "" {
		providers = append(providers, auth.NewGoogle(makeParams(s.Auth.Google.CID, s.Auth.Google.CSEC)))
	}
	if s.Auth.Github.CID != "" && s.Auth.Github.CSEC != "" {
		providers = append(providers, auth.NewGithub(makeParams(s.Auth.Github.CID, s.Auth.Github.CSEC)))
	}
	if s.Auth.Facebook.CID != "" && s.Auth.Facebook.CSEC != "" {
		providers = append(providers, auth.NewFacebook(makeParams(s.Auth.Facebook.CID, s.Auth.Facebook.CSEC)))
	}
	if s.Auth.Yandex.CID != "" && s.Auth.Yandex.CSEC != "" {
		providers = append(providers, auth.NewYandex(makeParams(s.Auth.Yandex.CID, s.Auth.Yandex.CSEC)))
	}
	if s.Auth.Dev {
		providers = append(providers, auth.NewDev(makeParams("", "")))
	}

	if len(providers) == 0 {
		log.Printf("[WARN] no auth providers defined")
	}
	return providers
}

func (s *ServerCommand) makeNotify(dataStore *service.DataStore) (*notify.Service, error) {
	log.Printf("[INFO] make notify, type=%s", s.Notify.Type)
	switch s.Notify.Type {
	case "telegram":
		tg, err := notify.NewTelegram(s.Notify.Telegram.Token, s.Notify.Telegram.Channel,
			s.Notify.Telegram.Timeout, s.Notify.Telegram.API)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create telegram notification destination")
		}
		return notify.NewService(dataStore, s.Notify.QueueSize, tg), nil
	case "none":
		return notify.NopService, nil
	}
	return nil, errors.Errorf("unsupported notification type %q", s.Notify.Type)
}

func (s *ServerCommand) makeSSLConfig() (config api.SSLConfig, err error) {
	switch s.SSL.Type {
	case "none":
		config.SSLMode = api.None
	case "static":
		if s.SSL.Cert == "" {
			return config, errors.New("path to cert.pem is required")
		}
		if s.SSL.Key == "" {
			return config, errors.New("path to key.pem is required")
		}
		config.SSLMode = api.Static
		config.Port = s.SSL.Port
		config.Cert = s.SSL.Cert
		config.Key = s.SSL.Key
	case "auto":
		config.SSLMode = api.Auto
		config.Port = s.SSL.Port
		config.ACMELocation = s.SSL.ACMELocation
		if s.SSL.ACMEEmail != "" {
			config.ACMEEmail = s.SSL.ACMEEmail
		} else if s.Admin.Type == "shared" && s.Admin.Shared.Email != "" {
			config.ACMEEmail = s.Admin.Shared.Email
		} else if u, e := url.Parse(s.RemarkURL); e == nil {
			config.ACMEEmail = "admin@" + u.Hostname()
		}
	}
	return config, err
}
