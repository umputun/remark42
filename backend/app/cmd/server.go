package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"regexp"
	"strings"
	"syscall"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/go-pkgz/jrpc"
	log "github.com/go-pkgz/lgr"
	"github.com/kyokomi/emoji"
	authcache "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"

	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/provider/sender"
	"github.com/go-pkgz/auth/token"
	cache "github.com/go-pkgz/lcw"

	"github.com/umputun/remark/backend/app/migrator"
	"github.com/umputun/remark/backend/app/notify"
	"github.com/umputun/remark/backend/app/rest/api"
	"github.com/umputun/remark/backend/app/rest/proxy"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/backend/app/store/image"
	"github.com/umputun/remark/backend/app/store/service"
)

// ServerCommand with command line flags and env
type ServerCommand struct {
	Store  StoreGroup  `group:"store" namespace:"store" env-namespace:"STORE"`
	Avatar AvatarGroup `group:"avatar" namespace:"avatar" env-namespace:"AVATAR"`
	Cache  CacheGroup  `group:"cache" namespace:"cache" env-namespace:"CACHE"`
	Admin  AdminGroup  `group:"admin" namespace:"admin" env-namespace:"ADMIN"`
	Notify NotifyGroup `group:"notify" namespace:"notify" env-namespace:"NOTIFY"`
	Image  ImageGroup  `group:"image" namespace:"image" env-namespace:"IMAGE"`
	SSL    SSLGroup    `group:"ssl" namespace:"ssl" env-namespace:"SSL"`
	Stream StreamGroup `group:"stream" namespace:"stream" env-namespace:"STREAM"`

	Sites           []string      `long:"site" env:"SITE" default:"remark" description:"site names" env-delim:","`
	AdminPasswd     string        `long:"admin-passwd" env:"ADMIN_PASSWD" default:"" description:"admin basic auth password"`
	BackupLocation  string        `long:"backup" env:"BACKUP_PATH" default:"./var/backup" description:"backups location"`
	MaxBackupFiles  int           `long:"max-back" env:"MAX_BACKUP_FILES" default:"10" description:"max backups to keep"`
	ImageProxy      bool          `long:"img-proxy" env:"IMG_PROXY" description:"enable image proxy"`
	MaxCommentSize  int           `long:"max-comment" env:"MAX_COMMENT_SIZE" default:"2048" description:"max comment size"`
	MaxVotes        int           `long:"max-votes" env:"MAX_VOTES" default:"-1" description:"maximum number of votes per comment"`
	RestrictVoteIP  bool          `long:"votes-ip" env:"VOTES_IP" description:"restrict votes from the same ip"`
	DurationVoteIP  time.Duration `long:"votes-ip-time" env:"VOTES_IP_TIME" default:"5m" description:"same ip vote duration"`
	LowScore        int           `long:"low-score" env:"LOW_SCORE" default:"-5" description:"low score threshold"`
	CriticalScore   int           `long:"critical-score" env:"CRITICAL_SCORE" default:"-10" description:"critical score threshold"`
	PositiveScore   bool          `long:"positive-score" env:"POSITIVE_SCORE" description:"enable positive score only"`
	ReadOnlyAge     int           `long:"read-age" env:"READONLY_AGE" default:"0" description:"read-only age of comments, days"`
	EditDuration    time.Duration `long:"edit-time" env:"EDIT_TIME" default:"5m" description:"edit window"`
	Port            int           `long:"port" env:"REMARK_PORT" default:"8080" description:"port"`
	WebRoot         string        `long:"web-root" env:"REMARK_WEB_ROOT" default:"./web" description:"web root directory"`
	UpdateLimit     float64       `long:"update-limit" env:"UPDATE_LIMIT" default:"0.5" description:"updates/sec limit"`
	RestrictedWords []string      `long:"restricted-words" env:"RESTRICTED_WORDS" description:"words prohibited to use in comments" env-delim:","`
	EnableEmoji     bool          `long:"emoji" env:"EMOJI" description:"enable emoji"`
	SimpleView      bool          `long:"simpler-view" env:"SIMPLE_VIEW" description:"minimal comment editor mode"`

	Auth struct {
		TTL struct {
			JWT    time.Duration `long:"jwt" env:"JWT" default:"5m" description:"jwt TTL"`
			Cookie time.Duration `long:"cookie" env:"COOKIE" default:"200h" description:"auth cookie TTL"`
		} `group:"ttl" namespace:"ttl" env-namespace:"TTL"`
		Google    AuthGroup `group:"google" namespace:"google" env-namespace:"GOOGLE" description:"Google OAuth"`
		Github    AuthGroup `group:"github" namespace:"github" env-namespace:"GITHUB" description:"Github OAuth"`
		Facebook  AuthGroup `group:"facebook" namespace:"facebook" env-namespace:"FACEBOOK" description:"Facebook OAuth"`
		Yandex    AuthGroup `group:"yandex" namespace:"yandex" env-namespace:"YANDEX" description:"Yandex OAuth"`
		Twitter   AuthGroup `group:"twitter" namespace:"twitter" env-namespace:"TWITTER" description:"Twitter OAuth"`
		Dev       bool      `long:"dev" env:"DEV" description:"enable dev (local) oauth2"`
		Anonymous bool      `long:"anon" env:"ANON" description:"enable anonymous login"`
		Email     struct {
			Enable       bool          `long:"enable" env:"ENABLE" description:"enable auth via email"`
			Host         string        `long:"host" env:"HOST" description:"smtp host"`
			Port         int           `long:"port" env:"PORT" description:"smtp port"`
			From         string        `long:"from" env:"FROM" description:"email's from"`
			Subject      string        `long:"subj" env:"SUBJ" default:"remark42 confirmation" description:"email's subject"`
			ContentType  string        `long:"content-type" env:"CONTENT_TYPE" default:"text/html" description:"content type"`
			TLS          bool          `long:"tls" env:"TLS" description:"enable TLS"`
			SMTPUserName string        `long:"user" env:"USER" description:"smtp user name"`
			SMTPPassword string        `long:"passwd" env:"PASSWD" description:"smtp password"`
			TimeOut      time.Duration `long:"timeout" env:"TIMEOUT" default:"10s" description:"smtp timeout"`
			MsgTemplate  string        `long:"template" env:"TEMPLATE" description:"message template file"`
		} `group:"email" namespace:"email" env-namespace:"EMAIL"`
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
	Type string `long:"type" env:"TYPE" description:"type of storage" choice:"bolt" choice:"rpc" default:"bolt"` // nolint
	Bolt struct {
		Path    string        `long:"path" env:"PATH" default:"./var" description:"parent dir for bolt files"`
		Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"30s" description:"bolt timeout"`
	} `group:"bolt" namespace:"bolt" env-namespace:"BOLT"`
	RPC RPCGroup `group:"rpc" namespace:"rpc" env-namespace:"RPC"`
}

// ImageGroup defines options group for store pictures
type ImageGroup struct {
	Type string `long:"type" env:"TYPE" description:"type of storage" choice:"fs" choice:"bolt" default:"fs"` // nolint
	FS   struct {
		Path       string `long:"path" env:"PATH" default:"./var/pictures" description:"images location"`
		Staging    string `long:"staging" env:"STAGING" default:"./var/pictures.staging" description:"staging location"`
		Partitions int    `long:"partitions" env:"PARTITIONS" default:"100" description:"partitions (subdirs)"`
	} `group:"fs" namespace:"fs" env-namespace:"FS"`
	Bolt struct {
		File string `long:"file" env:"FILE" default:"./var/pictures.db" description:"images bolt file location"`
	} `group:"bolt" namespace:"bolt" env-namespace:"bolt"`
	MaxSize      int `long:"max-size" env:"MAX_SIZE" default:"5000000" description:"max size of image file"`
	ResizeWidth  int `long:"resize-width" env:"RESIZE_WIDTH" default:"800" description:"width of resized image"`
	ResizeHeight int `long:"resize-height" env:"RESIZE_HEIGHT" default:"300" description:"height of resized image"`
}

// AvatarGroup defines options group for avatar params
type AvatarGroup struct {
	Type string `long:"type" env:"TYPE" description:"type of avatar storage" choice:"fs" choice:"bolt" choice:"uri" default:"fs"` //nolint
	FS   struct {
		Path string `long:"path" env:"PATH" default:"./var/avatars" description:"avatars location"`
	} `group:"fs" namespace:"fs" env-namespace:"FS"`
	Bolt struct {
		File string `long:"file" env:"FILE" default:"./var/avatars.db" description:"avatars bolt file location"`
	} `group:"bolt" namespace:"bolt" env-namespace:"bolt"`
	URI    string `long:"uri" env:"URI" default:"./var/avatars" description:"avatar's store URI"`
	RszLmt int    `long:"rsz-lmt" env:"RESIZE" default:"0" description:"max image size for resizing avatars on save"`
}

// CacheGroup defines options group for cache params
type CacheGroup struct {
	Type string `long:"type" env:"TYPE" description:"type of cache" choice:"mem" choice:"none" default:"mem"` // nolint
	Max  struct {
		Items int   `long:"items" env:"ITEMS" default:"1000" description:"max cached items"`
		Value int   `long:"value" env:"VALUE" default:"65536" description:"max size of cached value"`
		Size  int64 `long:"size" env:"SIZE" default:"50000000" description:"max size of total cache"`
	} `group:"max" namespace:"max" env-namespace:"MAX"`
}

// AdminGroup defines options group for admin params
type AdminGroup struct {
	Type   string `long:"type" env:"TYPE" description:"type of admin store" choice:"shared" choice:"rpc" default:"shared"` //nolint
	Shared struct {
		Admins []string `long:"id" env:"ID" description:"admin(s) ids" env-delim:","`
		Email  string   `long:"email" env:"EMAIL" default:"" description:"admin email"`
	} `group:"shared" namespace:"shared" env-namespace:"SHARED"`
	RPC RPCGroup `group:"rpc" namespace:"rpc" env-namespace:"RPC"`
}

// NotifyGroup defines options for notification
type NotifyGroup struct {
	Type      []string `long:"type" env:"TYPE" description:"type of notification" choice:"none" choice:"telegram" choice:"email" default:"none" env-delim:","` //nolint
	QueueSize int      `long:"queue" env:"QUEUE" description:"size of notification queue" default:"100"`
	Telegram  struct {
		Token   string        `long:"token" env:"TOKEN" description:"telegram token"`
		Channel string        `long:"chan" env:"CHAN" description:"telegram channel"`
		Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"5s" description:"telegram timeout"`
		API     string        `long:"api" env:"API" default:"https://api.telegram.org/bot" description:"telegram api prefix"`
	} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`
	Email struct {
		Host                string        `long:"host" env:"HOST" description:"SMTP host"`
		Port                int           `long:"port" env:"PORT" default:"587" description:"SMTP port"`
		TLS                 bool          `long:"tls" env:"TLS" description:"enable TLS for SMTP"`
		From                string        `long:"fromAddress" env:"FROM" description:"from email address"`
		Username            string        `long:"username" env:"USERNAME" description:"SMTP user name"`
		Password            string        `long:"password" env:"PASSWORD" description:"SMTP password"`
		TimeOut             time.Duration `long:"timeout" env:"TIMEOUT" default:"10s" description:"SMTP TCP connection timeout"`
		VerificationSubject string        `long:"verification_subj" env:"VERIFICATION_SUBJ" description:"verification message subject"`
	} `group:"email" namespace:"email" env-namespace:"EMAIL"`
}

// SSLGroup defines options group for server ssl params
type SSLGroup struct {
	Type         string `long:"type" env:"TYPE" description:"ssl (auto) support" choice:"none" choice:"static" choice:"auto" default:"none"` //nolint
	Port         int    `long:"port" env:"PORT" description:"port number for https server" default:"8443"`
	Cert         string `long:"cert" env:"CERT" description:"path to cert.pem file"`
	Key          string `long:"key" env:"KEY" description:"path to key.pem file"`
	ACMELocation string `long:"acme-location" env:"ACME_LOCATION" description:"dir where certificates will be stored by autocert manager" default:"./var/acme"`
	ACMEEmail    string `long:"acme-email" env:"ACME_EMAIL" description:"admin email for certificate notifications"`
}

// StreamGroup define options for streaming apis
type StreamGroup struct {
	RefreshInterval time.Duration `long:"refresh" env:"REFRESH" default:"5s" description:"refresh interval for streams"`
	TimeOut         time.Duration `long:"timeout" env:"TIMEOUT" default:"15m" description:"timeout to close streams on inactivity"`
	MaxActive       int           `long:"max" env:"MAX" default:"500" description:"max number of parallel streams"`
}

// RPCGroup defines options for remote modules (plugins)
type RPCGroup struct {
	API          string        `long:"api" env:"API" description:"rpc extension api url"`
	TimeOut      time.Duration `long:"timeout" env:"TIMEOUT" default:"5s" description:"http timeout"`
	AuthUser     string        `long:"auth_user" env:"AUTH_USER" description:"basic auth user name"`
	AuthPassword string        `long:"auth_passwd" env:"AUTH_PASSWD" description:"basic auth user password"`
}

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key cache.Key, fn func() ([]byte, error)) (data []byte, err error) // load from cache if found or put to cache and return
	Flush(req cache.FlusherRequest)                                        // evict matched records
}

// serverApp holds all active objects
type serverApp struct {
	*ServerCommand
	restSrv       *api.Rest
	migratorSrv   *api.Migrator
	exporter      migrator.Exporter
	devAuth       *provider.DevAuthServer
	dataService   *service.DataStore
	avatarStore   avatar.Store
	notifyService *notify.Service
	imageService  *image.Service
	terminated    chan struct{}
}

// Execute is the entry point for "server" command, called by flag parser
func (s *ServerCommand) Execute(args []string) error {
	log.Printf("[INFO] start server on port %d", s.Port)
	resetEnv("SECRET", "AUTH_GOOGLE_CSEC", "AUTH_GITHUB_CSEC", "AUTH_FACEBOOK_CSEC", "AUTH_YANDEX_CSEC", "ADMIN_PASSWD")

	ctx, cancel := context.WithCancel(context.Background())
	go func() { // catch signal and invoke graceful termination
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Printf("[WARN] interrupt signal")
		cancel()
	}()

	app, err := s.newServerApp()
	if err != nil {
		log.Printf("[PANIC] failed to setup application, %+v", err)
		return err
	}
	if err = app.run(ctx); err != nil {
		log.Printf("[ERROR] remark terminated with error %+v", err)
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

	imageService, err := s.makePicturesStore()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make pictures store")
	}
	log.Printf("[DEBUG] image service for url=%s, ttl=%v", imageService.ImageAPI, imageService.TTL)

	dataService := &service.DataStore{
		Engine:                 storeEngine,
		EditDuration:           s.EditDuration,
		AdminStore:             adminStore,
		MaxCommentSize:         s.MaxCommentSize,
		MaxVotes:               s.MaxVotes,
		PositiveScore:          s.PositiveScore,
		ImageService:           imageService,
		TitleExtractor:         service.NewTitleExtractor(http.Client{Timeout: time.Second * 5}),
		RestrictedWordsMatcher: service.NewRestrictedWordsMatcher(service.StaticRestrictedWordsLister{Words: s.RestrictedWords}),
	}
	dataService.RestrictSameIPVotes.Enabled = s.RestrictVoteIP
	dataService.RestrictSameIPVotes.Duration = s.DurationVoteIP

	loadingCache, err := s.makeCache()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make cache")
	}

	avatarStore, err := s.makeAvatarStore()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make avatar store")
	}
	authenticator := s.makeAuthenticator(dataService, avatarStore, adminStore)

	exporter := &migrator.Native{DataStore: dataService}

	migr := &api.Migrator{
		Cache:             loadingCache,
		NativeImporter:    &migrator.Native{DataStore: dataService},
		DisqusImporter:    &migrator.Disqus{DataStore: dataService},
		WordPressImporter: &migrator.WordPress{DataStore: dataService},
		NativeExporter:    &migrator.Native{DataStore: dataService},
		UrlMapperMaker:    migrator.NewUrlMapper,
		KeyStore:          adminStore,
	}

	notifyService, err := s.makeNotify(dataService)
	if err != nil {
		log.Printf("[WARN] failed to make notify service, %s", err)
		notifyService = notify.NopService // disable notifier
	}

	imgProxy := &proxy.Image{Enabled: s.ImageProxy, RoutePath: "/api/v1/img", RemarkURL: s.RemarkURL}
	emojiFmt := store.CommentConverterFunc(func(text string) string { return text })
	if s.EnableEmoji {
		emojiFmt = func(text string) string { return emoji.Sprint(text) }
	}
	commentFormatter := store.NewCommentFormatter(imgProxy, emojiFmt)

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
		Migrator:         migr,
		ReadOnlyAge:      s.ReadOnlyAge,
		SharedSecret:     s.SharedSecret,
		Authenticator:    authenticator,
		Cache:            loadingCache,
		NotifyService:    notifyService,
		SSLConfig:        sslConfig,
		UpdateLimiter:    s.UpdateLimit,
		ImageService:     imageService,
		Streamer: &api.Streamer{
			TimeOut:   s.Stream.TimeOut,
			Refresh:   s.Stream.RefreshInterval,
			MaxActive: int32(s.Stream.MaxActive),
		},
		EmojiEnabled: s.EnableEmoji,
		SimpleView:   s.SimpleView,
	}

	srv.ScoreThresholds.Low, srv.ScoreThresholds.Critical = s.LowScore, s.CriticalScore

	var devAuth *provider.DevAuthServer
	if s.Auth.Dev {
		da, errDevAuth := authenticator.DevAuth()
		if errDevAuth != nil {
			return nil, errors.Wrap(errDevAuth, "can't make dev oauth2 server")
		}
		devAuth = da
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
		imageService:  imageService,
		terminated:    make(chan struct{}),
	}, nil
}

// Run all application objects
func (a *serverApp) run(ctx context.Context) error {
	if a.AdminPasswd != "" {
		log.Printf("[WARN] admin basic auth enabled")
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
		a.imageService.Close()
		log.Print("[INFO] shutdown completed")
	}()

	a.activateBackup(ctx) // runs in goroutine for each site
	if a.Auth.Dev {
		go a.devAuth.Run(context.Background()) // dev oauth2 server on :8084
	}

	go a.imageService.Cleanup(ctx) // pictures cleanup for staging images

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
	case "rpc":
		r := &engine.RPC{Client: jrpc.Client{
			API:        s.Store.RPC.API,
			Client:     http.Client{Timeout: s.Store.RPC.TimeOut},
			AuthUser:   s.Store.RPC.AuthUser,
			AuthPasswd: s.Store.RPC.AuthPassword,
		}}
		return r, nil
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
		return avatar.NewLocalFS(s.Avatar.FS.Path), nil
	case "bolt":
		if err := makeDirs(path.Dir(s.Avatar.Bolt.File)); err != nil {
			return nil, err
		}
		return avatar.NewBoltDB(s.Avatar.Bolt.File, bolt.Options{})
	case "uri":
		return avatar.NewStore(s.Avatar.URI)
	}
	return nil, errors.Errorf("unsupported avatar store type %s", s.Avatar.Type)
}

func (s *ServerCommand) makePicturesStore() (*image.Service, error) {
	switch s.Image.Type {
	case "bolt":
		boltImageStore, err := image.NewBoltStorage(
			s.Image.Bolt.File,
			s.Image.MaxSize,
			s.Image.ResizeHeight,
			s.Image.ResizeWidth,
			bolt.Options{},
		)
		if err != nil {
			return nil, err
		}
		return &image.Service{
			Store:    boltImageStore,
			ImageAPI: s.RemarkURL + "/api/v1/picture/",
			TTL:      5 * s.EditDuration, // add extra time to image TTL for staging
		}, nil
	case "fs":
		if err := makeDirs(s.Image.FS.Path); err != nil {
			return nil, err
		}
		return &image.Service{
			Store: &image.FileSystem{
				Location:   s.Image.FS.Path,
				Staging:    s.Image.FS.Staging,
				Partitions: s.Image.FS.Partitions,
				MaxSize:    s.Image.MaxSize,
				MaxHeight:  s.Image.ResizeHeight,
				MaxWidth:   s.Image.ResizeWidth,
			},
			ImageAPI: s.RemarkURL + "/api/v1/picture/",
			TTL:      5 * s.EditDuration, // add extra time to image TTL for staging
		}, nil
	}
	return nil, errors.Errorf("unsupported pictures store type %s", s.Image.Type)
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
		return admin.NewStaticStore(s.SharedSecret, s.Sites, s.Admin.Shared.Admins, s.Admin.Shared.Email), nil
	case "rpc":
		r := &admin.RPC{Client: jrpc.Client{
			API:        s.Admin.RPC.API,
			Client:     http.Client{Timeout: s.Admin.RPC.TimeOut},
			AuthUser:   s.Admin.RPC.AuthUser,
			AuthPasswd: s.Admin.RPC.AuthPassword,
		}}
		return r, nil
	default:
		return nil, errors.Errorf("unsupported admin store type %s", s.Admin.Type)
	}
}

func (s *ServerCommand) makeCache() (LoadingCache, error) {
	log.Printf("[INFO] make cache, type=%s", s.Cache.Type)
	switch s.Cache.Type {
	case "mem":
		backend, err := cache.NewLruCache(cache.MaxCacheSize(s.Cache.Max.Size), cache.MaxValSize(s.Cache.Max.Value),
			cache.MaxKeys(s.Cache.Max.Items))
		if err != nil {
			return nil, errors.Wrap(err, "cache backend initialization")
		}
		return cache.NewScache(backend), nil
	case "none":
		return cache.NewScache(&cache.Nop{}), nil
	}
	return nil, errors.Errorf("unsupported cache type %s", s.Cache.Type)
}

var msgTemplate = `
<!DOCTYPE html>
<html>
<head>
	<meta name="viewport" content="width=device-width" />
	<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
</head>
<body>
<div style="text-align: center; font-family: Arial, sans-serif; font-size: 18px;">
	<h1 style="position: relative; color: #4fbbd6; margin-top: 0.2em;">Remark42</h1>
	<p style="position: relative; max-width: 20em; margin: 0 auto 1em auto; line-height: 1.4em;">Confirmation for <b>{{.User}}</b> on site <b>{{.Site}}</b></p>
	<div style="background-color: #eee; max-width: 20em; margin: 0 auto; border-radius: 0.4em; padding: 0.5em;">
		<p style="position: relative; margin: 0 0 0.5em 0;">TOKEN</p>
		<p style="position: relative; font-size: 0.7em; opacity: 0.8;"><i>Copy and paste this text into “token” field on comments page</i></p>
		<p style="position: relative; font-family: monospace; background-color: #fff; margin: 0; padding: 0.5em; word-break: break-all; text-align: left; border-radius: 0.2em; -webkit-user-select: all; user-select: all;">{{.Token}}</p>
	</div>
	<p style="position: relative; margin-top: 2em; font-size: 0.8em; opacity: 0.8;"><i>Sent to {{.Address}}</i></p>
</div>
</body>
</html>
`

func (s *ServerCommand) addAuthProviders(authenticator *auth.Service) {

	providers := 0
	if s.Auth.Google.CID != "" && s.Auth.Google.CSEC != "" {
		authenticator.AddProvider("google", s.Auth.Google.CID, s.Auth.Google.CSEC)
		providers++
	}
	if s.Auth.Github.CID != "" && s.Auth.Github.CSEC != "" {
		authenticator.AddProvider("github", s.Auth.Github.CID, s.Auth.Github.CSEC)
		providers++
	}
	if s.Auth.Facebook.CID != "" && s.Auth.Facebook.CSEC != "" {
		authenticator.AddProvider("facebook", s.Auth.Facebook.CID, s.Auth.Facebook.CSEC)
		providers++
	}
	if s.Auth.Yandex.CID != "" && s.Auth.Yandex.CSEC != "" {
		authenticator.AddProvider("yandex", s.Auth.Yandex.CID, s.Auth.Yandex.CSEC)
		providers++
	}
	if s.Auth.Twitter.CID != "" && s.Auth.Twitter.CSEC != "" {
		authenticator.AddProvider("twitter", s.Auth.Twitter.CID, s.Auth.Twitter.CSEC)
		providers++
	}

	if s.Auth.Dev {
		log.Print("[INFO] dev access enabled")
		authenticator.AddProvider("dev", "", "")
		providers++
	}

	if s.Auth.Email.Enable {
		params := sender.EmailParams{
			Host:         s.Auth.Email.Host,
			Port:         s.Auth.Email.Port,
			From:         s.Auth.Email.From,
			Subject:      s.Auth.Email.Subject,
			ContentType:  s.Auth.Email.ContentType,
			TLS:          s.Auth.Email.TLS,
			SMTPUserName: s.Auth.Email.SMTPUserName,
			SMTPPassword: s.Auth.Email.SMTPPassword,
			TimeOut:      s.Auth.Email.TimeOut,
		}
		sndr := sender.NewEmailClient(params, log.Default())
		authenticator.AddVerifProvider("email", s.loadEmailTemplate(), sndr)
	}

	if s.Auth.Anonymous {
		log.Print("[INFO] anonymous access enabled")
		var isValidAnonName = regexp.MustCompile(`^[a-zA-Z][\w ]+$`).MatchString
		authenticator.AddDirectProvider("anonymous", provider.CredCheckerFunc(func(user, _ string) (ok bool, err error) {
			user = strings.TrimSpace(user)
			if len(user) < 3 {
				log.Printf("[WARN] name %q is too short, should be at least 3 characters", user)
				return false, nil
			}

			if !isValidAnonName(user) {
				log.Printf("[WARN] name %q should have letters, digits, underscores and spaces only", user)
				return false, nil
			}
			return true, nil
		}))
	}

	if providers == 0 {
		log.Printf("[WARN] no auth providers defined")
	}
}

// loadEmailTemplate trying to get template from opts MsgTemplate and default to embedded
// if not defined or failed to load
func (s *ServerCommand) loadEmailTemplate() string {
	tmpl := msgTemplate
	if s.Auth.Email.MsgTemplate != "" {
		log.Printf("[DEBUG] load email template from %s", s.Auth.Email.MsgTemplate)
		b, err := ioutil.ReadFile(s.Auth.Email.MsgTemplate)
		if err == nil {
			tmpl = string(b)
		} else {
			log.Printf("[WARN] failed to load email template from %s, %v", s.Auth.Email.MsgTemplate, err)
		}
	}
	return tmpl
}

func (s *ServerCommand) makeNotify(dataStore *service.DataStore) (*notify.Service, error) {
	var notifyService *notify.Service
	var destinations []notify.Destination
	for _, t := range s.Notify.Type {
		switch t {
		case "telegram":
			tg, err := notify.NewTelegram(s.Notify.Telegram.Token, s.Notify.Telegram.Channel,
				s.Notify.Telegram.Timeout, s.Notify.Telegram.API)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create telegram notification destination")
			}
			destinations = append(destinations, tg)
		case "email":
			emailParams := notify.EmailParams{
				From:                s.Notify.Email.From,
				VerificationSubject: s.Notify.Email.VerificationSubject,
			}
			smtpParams := notify.SmtpParams{
				Host:     s.Notify.Email.Host,
				Port:     s.Notify.Email.Port,
				TLS:      s.Notify.Email.TLS,
				Username: s.Notify.Email.Username,
				Password: s.Notify.Email.Password,
				TimeOut:  s.Notify.Email.TimeOut,
			}
			emailService, err := notify.NewEmail(emailParams, smtpParams)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create email notification destination")
			}
			destinations = append(destinations, emailService)
		case "none":
			notifyService = notify.NopService
		default:
			return nil, errors.Errorf("unsupported notification type %q", s.Notify.Type)
		}
	}

	if len(destinations) != 0 {
		log.Printf("[INFO] make notify, types=%s", s.Notify.Type)
		notifyService = notify.NewService(dataStore, s.Notify.QueueSize, destinations...)
	}
	return notifyService, nil
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

func (s *ServerCommand) makeAuthenticator(ds *service.DataStore, avas avatar.Store, admns admin.Store) *auth.Service {
	authenticator := auth.NewService(auth.Opts{
		URL:            strings.TrimSuffix(s.RemarkURL, "/"),
		Issuer:         "remark42",
		TokenDuration:  s.Auth.TTL.JWT,
		CookieDuration: s.Auth.TTL.Cookie,
		SecureCookies:  strings.HasPrefix(s.RemarkURL, "https://"),
		SecretReader: token.SecretFunc(func() (string, error) { // get secret per site
			return admns.Key()
		}),
		ClaimsUpd: token.ClaimsUpdFunc(func(c token.Claims) token.Claims { // set attributes, on new token or refresh
			if c.User == nil {
				return c
			}
			c.User.SetAdmin(ds.IsAdmin(c.Audience, c.User.ID))
			c.User.SetBoolAttr("blocked", ds.IsBlocked(c.Audience, c.User.ID))
			var err error
			c.User.Email, err = ds.GetUserEmail(store.Locator{SiteID: c.Audience}, c.User.ID)
			if err != nil {
				log.Printf("[WARN] can't read email for %s, %v", c.User.ID, err)
			}
			return c
		}),
		AdminPasswd: s.AdminPasswd,
		Validator: token.ValidatorFunc(func(token string, claims token.Claims) bool { // check on each auth call (in middleware)
			if claims.User == nil {
				return false
			}
			if claims.User.Audience == "" { // reject empty aud, made with old (pre 0.8.x) version of auth package
				return false
			}
			return !claims.User.BoolAttr("blocked")
		}),
		JWTQuery:          "jwt", // change default from "token" as it used for deleteme
		AvatarStore:       avas,
		AvatarResizeLimit: s.Avatar.RszLmt,
		AvatarRoutePath:   "/api/v1/avatar",
		Logger:            log.Default(),
		RefreshCache:      newAuthRefreshCache(),
		UseGravatar:       true,
	})
	s.addAuthProviders(authenticator)
	return authenticator
}

// authRefreshCache used by authenticator to minimize repeatable token refreshes
type authRefreshCache struct {
	*authcache.Cache
}

func newAuthRefreshCache() *authRefreshCache {
	return &authRefreshCache{Cache: authcache.New(5*time.Minute, 10*time.Minute)}
}

// Get implements cache getter with key converted to string
func (c *authRefreshCache) Get(key interface{}) (interface{}, bool) {
	return c.Cache.Get(key.(string))
}

// Set implements cache setter with key converted to string
func (c *authRefreshCache) Set(key, value interface{}) {
	c.Cache.Set(key.(string), value, authcache.DefaultExpiration)
}
