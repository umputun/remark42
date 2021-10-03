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

	"github.com/go-pkgz/jrpc"
	"github.com/go-pkgz/lcw/eventbus"
	log "github.com/go-pkgz/lgr"
	"github.com/golang-jwt/jwt"
	"github.com/kyokomi/emoji/v2"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"

	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/provider/sender"
	"github.com/go-pkgz/auth/token"
	cache "github.com/go-pkgz/lcw"

	"github.com/umputun/remark42/backend/app/migrator"
	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/rest/api"
	"github.com/umputun/remark42/backend/app/rest/proxy"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/image"
	"github.com/umputun/remark42/backend/app/store/service"
	"github.com/umputun/remark42/backend/app/templates"
)

// ServerCommand with command line flags and env
type ServerCommand struct {
	Store      StoreGroup      `group:"store" namespace:"store" env-namespace:"STORE"`
	Avatar     AvatarGroup     `group:"avatar" namespace:"avatar" env-namespace:"AVATAR"`
	Cache      CacheGroup      `group:"cache" namespace:"cache" env-namespace:"CACHE"`
	Admin      AdminGroup      `group:"admin" namespace:"admin" env-namespace:"ADMIN"`
	Notify     NotifyGroup     `group:"notify" namespace:"notify" env-namespace:"NOTIFY"`
	SMTP       SMTPGroup       `group:"smtp" namespace:"smtp" env-namespace:"SMTP"`
	Telegram   TelegramGroup   `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`
	Image      ImageGroup      `group:"image" namespace:"image" env-namespace:"IMAGE"`
	SSL        SSLGroup        `group:"ssl" namespace:"ssl" env-namespace:"SSL"`
	ImageProxy ImageProxyGroup `group:"image-proxy" namespace:"image-proxy" env-namespace:"IMAGE_PROXY"`

	Sites            []string      `long:"site" env:"SITE" default:"remark" description:"site names" env-delim:","`
	AnonymousVote    bool          `long:"anon-vote" env:"ANON_VOTE" description:"enable anonymous votes (works only with VOTES_IP enabled)"`
	AdminPasswd      string        `long:"admin-passwd" env:"ADMIN_PASSWD" default:"" description:"admin basic auth password"`
	BackupLocation   string        `long:"backup" env:"BACKUP_PATH" default:"./var/backup" description:"backups location"`
	MaxBackupFiles   int           `long:"max-back" env:"MAX_BACKUP_FILES" default:"10" description:"max backups to keep"`
	LegacyImageProxy bool          `long:"img-proxy" env:"IMG_PROXY" description:"[deprecated, use image-proxy.http2https] enable image proxy"`
	MaxCommentSize   int           `long:"max-comment" env:"MAX_COMMENT_SIZE" default:"2048" description:"max comment size"`
	MaxVotes         int           `long:"max-votes" env:"MAX_VOTES" default:"-1" description:"maximum number of votes per comment"`
	RestrictVoteIP   bool          `long:"votes-ip" env:"VOTES_IP" description:"restrict votes from the same ip"`
	DurationVoteIP   time.Duration `long:"votes-ip-time" env:"VOTES_IP_TIME" default:"5m" description:"same ip vote duration"`
	LowScore         int           `long:"low-score" env:"LOW_SCORE" default:"-5" description:"low score threshold"`
	CriticalScore    int           `long:"critical-score" env:"CRITICAL_SCORE" default:"-10" description:"critical score threshold"`
	PositiveScore    bool          `long:"positive-score" env:"POSITIVE_SCORE" description:"enable positive score only"`
	ReadOnlyAge      int           `long:"read-age" env:"READONLY_AGE" default:"0" description:"read-only age of comments, days"`
	EditDuration     time.Duration `long:"edit-time" env:"EDIT_TIME" default:"5m" description:"edit window"`
	AdminEdit        bool          `long:"admin-edit" env:"ADMIN_EDIT" description:"unlimited edit for admins"`
	Port             int           `long:"port" env:"REMARK_PORT" default:"8080" description:"port"`
	Address          string        `long:"address" env:"REMARK_ADDRESS" default:"" description:"listening address"`
	WebRoot          string        `long:"web-root" env:"REMARK_WEB_ROOT" default:"./web" description:"web root directory"`
	UpdateLimit      float64       `long:"update-limit" env:"UPDATE_LIMIT" default:"0.5" description:"updates/sec limit"`
	RestrictedWords  []string      `long:"restricted-words" env:"RESTRICTED_WORDS" description:"words prohibited to use in comments" env-delim:","`
	RestrictedNames  []string      `long:"restricted-names" env:"RESTRICTED_NAMES" description:"names prohibited to use by user" env-delim:","`
	EnableEmoji      bool          `long:"emoji" env:"EMOJI" description:"enable emoji"`
	SimpleView       bool          `long:"simpler-view" env:"SIMPLE_VIEW" description:"minimal comment editor mode"`
	ProxyCORS        bool          `long:"proxy-cors" env:"PROXY_CORS" description:"disable internal CORS and delegate it to proxy"`
	AllowedHosts     []string      `long:"allowed-hosts" env:"ALLOWED_HOSTS" description:"limit hosts/sources allowed to embed comments"`

	Auth struct {
		TTL struct {
			JWT    time.Duration `long:"jwt" env:"JWT" default:"5m" description:"jwt TTL"`
			Cookie time.Duration `long:"cookie" env:"COOKIE" default:"200h" description:"auth cookie TTL"`
		} `group:"ttl" namespace:"ttl" env-namespace:"TTL"`

		SendJWTHeader bool   `long:"send-jwt-header" env:"SEND_JWT_HEADER" description:"send JWT as a header instead of cookie"`
		SameSite      string `long:"same-site" env:"SAME_SITE" description:"set same site policy for cookies" choice:"default" choice:"none" choice:"lax" choice:"strict" default:"default"` // nolint

		Google    AuthGroup `group:"google" namespace:"google" env-namespace:"GOOGLE" description:"Google OAuth"`
		Github    AuthGroup `group:"github" namespace:"github" env-namespace:"GITHUB" description:"Github OAuth"`
		Facebook  AuthGroup `group:"facebook" namespace:"facebook" env-namespace:"FACEBOOK" description:"Facebook OAuth"`
		Microsoft AuthGroup `group:"microsoft" namespace:"microsoft" env-namespace:"MICROSOFT" description:"Microsoft OAuth"`
		Yandex    AuthGroup `group:"yandex" namespace:"yandex" env-namespace:"YANDEX" description:"Yandex OAuth"`
		Twitter   AuthGroup `group:"twitter" namespace:"twitter" env-namespace:"TWITTER" description:"Twitter OAuth"`
		Telegram  bool      `long:"telegram" env:"TELEGRAM" description:"Enable Telegram auth (using token from telegram.token)"`
		Dev       bool      `long:"dev" env:"DEV" description:"enable dev (local) oauth2"`
		Anonymous bool      `long:"anon" env:"ANON" description:"enable anonymous login"`
		Email     struct {
			Enable       bool          `long:"enable" env:"ENABLE" description:"enable auth via email"`
			From         string        `long:"from" env:"FROM" description:"from email address"`
			Subject      string        `long:"subj" env:"SUBJ" default:"remark42 confirmation" description:"email's subject"`
			ContentType  string        `long:"content-type" env:"CONTENT_TYPE" default:"text/html" description:"content type"`
			Host         string        `long:"host" env:"HOST" description:"[deprecated, use --smtp.host] SMTP host"`
			Port         int           `long:"port" env:"PORT" description:"[deprecated, use --smtp.port] SMTP password"`
			SMTPPassword string        `long:"passwd" env:"PASSWD" description:"[deprecated, use --smtp.password] SMTP port"`
			SMTPUserName string        `long:"user" env:"USER" description:"[deprecated, use --smtp.username] enable TLS"`
			TLS          bool          `long:"tls" env:"TLS" description:"[deprecated, use --smtp.tls] SMTP TCP connection timeout"`
			TimeOut      time.Duration `long:"timeout" env:"TIMEOUT" default:"10s" description:"[deprecated, use --smtp.timeout] SMTP TCP connection timeout"`
			MsgTemplate  string        `long:"template" env:"TEMPLATE" description:"[deprecated] message template file" default:"email_confirmation_login.html.tmpl"`
		} `group:"email" namespace:"email" env-namespace:"EMAIL"`
	} `group:"auth" namespace:"auth" env-namespace:"AUTH"`

	CommonOpts

	emailMsgTemplatePath          string // used only in tests
	emailVerificationTemplatePath string // used only in tests
}

// ImageProxyGroup defines options group for image proxy
type ImageProxyGroup struct {
	HTTP2HTTPS    bool `long:"http2https" env:"HTTP2HTTPS" description:"enable HTTP->HTTPS proxy"`
	CacheExternal bool `long:"cache-external" env:"CACHE_EXTERNAL" description:"enable caching for external images"`
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
	Type string `long:"type" env:"TYPE" description:"type of storage" choice:"fs" choice:"bolt" choice:"rpc" default:"fs"` // nolint
	FS   struct {
		Path       string `long:"path" env:"PATH" default:"./var/pictures" description:"images location"`
		Staging    string `long:"staging" env:"STAGING" default:"./var/pictures.staging" description:"staging location"`
		Partitions int    `long:"partitions" env:"PARTITIONS" default:"100" description:"partitions (subdirs)"`
	} `group:"fs" namespace:"fs" env-namespace:"FS"`
	Bolt struct {
		File string `long:"file" env:"FILE" default:"./var/pictures.db" description:"images bolt file location"`
	} `group:"bolt" namespace:"bolt" env-namespace:"bolt"`
	MaxSize      int      `long:"max-size" env:"MAX_SIZE" default:"5000000" description:"max size of image file"`
	ResizeWidth  int      `long:"resize-width" env:"RESIZE_WIDTH" default:"2400" description:"width of resized image"`
	ResizeHeight int      `long:"resize-height" env:"RESIZE_HEIGHT" default:"900" description:"height of resized image"`
	RPC          RPCGroup `group:"rpc" namespace:"rpc" env-namespace:"RPC"`
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
	Type      string `long:"type" env:"TYPE" description:"type of cache" choice:"redis_pub_sub" choice:"mem" choice:"none" default:"mem"` // nolint
	RedisAddr string `long:"redis_addr" env:"REDIS_ADDR" default:"127.0.0.1:6379" description:"address of redis cache, turn redis cache on for distributed cache"`
	Max       struct {
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
		Email  []string `long:"email" env:"EMAIL" description:"admin emails" env-delim:","`
	} `group:"shared" namespace:"shared" env-namespace:"SHARED"`
	RPC RPCGroup `group:"rpc" namespace:"rpc" env-namespace:"RPC"`
}

// TelegramGroup defines token for Telegram used in notify and auth modules
type TelegramGroup struct {
	Token   string        `long:"token" env:"TOKEN" description:"telegram token (used for auth and telegram notifications)"`
	Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"5s" description:"telegram timeout"`
}

// SMTPGroup defines options for SMTP server connection, used in auth and notify modules
type SMTPGroup struct {
	Host     string        `long:"host" env:"HOST" description:"SMTP host"`
	Port     int           `long:"port" env:"PORT" description:"SMTP port"`
	Username string        `long:"username" env:"USERNAME" description:"SMTP user name"`
	Password string        `long:"password" env:"PASSWORD" description:"SMTP password"`
	TLS      bool          `long:"tls" env:"TLS" description:"enable TLS"`
	TimeOut  time.Duration `long:"timeout" env:"TIMEOUT" default:"10s" description:"SMTP TCP connection timeout"`
}

// NotifyGroup defines options for notification
type NotifyGroup struct {
	Type      []string `long:"type" env:"TYPE" description:"[deprecated, use user and admin types instead] types of notifications" choice:"none" choice:"telegram" choice:"email" choice:"slack" default:"none" env-delim:","` //nolint
	Users     []string `long:"users" env:"USERS" description:"types of user notifications" choice:"none" choice:"email" choice:"telegram" default:"none" env-delim:","`                                                        //nolint
	Admins    []string `long:"admins" env:"ADMINS" description:"types of admin notifications" choice:"none" choice:"telegram" choice:"email" choice:"slack" choice:"webhook" default:"none" env-delim:","`                     //nolint
	QueueSize int      `long:"queue" env:"QUEUE" description:"size of notification queue" default:"100"`
	Telegram  struct {
		Channel string        `long:"chan" env:"CHAN" description:"telegram channel for admin notifications"`
		API     string        `long:"api" env:"API" default:"https://api.telegram.org/bot" description:"[deprecated, not used] telegram api prefix"`
		Token   string        `long:"token" env:"TOKEN" description:"[deprecated, use --telegram.token] telegram token"`
		Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"5s" description:"[deprecated, use --telegram.timeout] telegram timeout"`
	} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`
	Email struct {
		From                string `long:"from_address" env:"FROM" description:"from email address"`
		VerificationSubject string `long:"verification_subj" env:"VERIFICATION_SUBJ" description:"verification message subject"`
		AdminNotifications  bool   `long:"notify_admin" env:"ADMIN" description:"[deprecated, use --notify.admins=email] notify admin on new comments via ADMIN_SHARED_EMAIL"`
	} `group:"email" namespace:"email" env-namespace:"EMAIL"`
	Slack struct {
		Token   string `long:"token" env:"TOKEN" description:"slack token"`
		Channel string `long:"chan" env:"CHAN" description:"slack channel"`
	} `group:"slack" namespace:"slack" env-namespace:"SLACK"`
	Webhook struct {
		WebhookURL string        `long:"url" env:"URL" description:"webhook notification URL"`
		Template   string        `long:"template" env:"TEMPLATE" description:"webhook authentication template" default:"{\"text\": \"{{.Text}}\"}"`
		Headers    []string      `long:"headers" description:"webhook authentication headers in format --notify.webhook.headers=Header1:Value1,Value2,..."` // env NOTIFY_WEBHOOK_HEADERS split in code bellow to allow , inside ""
		Timeout    time.Duration `long:"timeout" env:"TIMEOUT" description:"webhook timeout" default:"5s"`
	} `group:"webhook" namespace:"webhook" env-namespace:"WEBHOOK"`
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
	Close() error
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
	authenticator *auth.Service
	terminated    chan struct{}

	authRefreshCache *authRefreshCache // stored only to close it properly on shutdown
}

// Execute is the entry point for "server" command, called by flag parser
func (s *ServerCommand) Execute(_ []string) error {
	log.Printf("[INFO] start server on port %s:%d", s.Address, s.Port)
	resetEnv(
		"SECRET",
		"AUTH_GOOGLE_CSEC",
		"AUTH_GITHUB_CSEC",
		"AUTH_FACEBOOK_CSEC",
		"AUTH_MICROSOFT_CSEC",
		"AUTH_TWITTER_CSEC",
		"AUTH_YANDEX_CSEC",
		"TELEGRAM_TOKEN",
		"SMTP_PASSWORD",
		"ADMIN_PASSWD",
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { // catch signal and invoke graceful termination
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Printf("[WARN] interrupt signal")
		cancel()
	}()

	app, err := s.newServerApp(ctx)
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

// HandleDeprecatedFlags sets new flags from deprecated returns their list
func (s *ServerCommand) HandleDeprecatedFlags() (result []DeprecatedFlag) {
	if s.Auth.Email.Host != "" && s.SMTP.Host == "" {
		s.SMTP.Host = s.Auth.Email.Host
		result = append(result, DeprecatedFlag{Old: "auth.email.host", New: "smtp.host", Version: "1.5"})
	}
	if s.Auth.Email.Port != 0 && s.SMTP.Port == 0 {
		s.SMTP.Port = s.Auth.Email.Port
		result = append(result, DeprecatedFlag{Old: "auth.email.port", New: "smtp.port", Version: "1.5"})
	}
	if s.Auth.Email.TLS && !s.SMTP.TLS {
		s.SMTP.TLS = s.Auth.Email.TLS
		result = append(result, DeprecatedFlag{Old: "auth.email.tls", New: "smtp.tls", Version: "1.5"})
	}
	if s.Auth.Email.SMTPUserName != "" && s.SMTP.Username == "" {
		s.SMTP.Username = s.Auth.Email.SMTPUserName
		result = append(result, DeprecatedFlag{Old: "auth.email.user", New: "smtp.username", Version: "1.5"})
	}
	if s.Auth.Email.SMTPPassword != "" && s.SMTP.Password == "" {
		s.SMTP.Password = s.Auth.Email.SMTPPassword
		result = append(result, DeprecatedFlag{Old: "auth.email.passwd", New: "smtp.password", Version: "1.5"})
	}
	if s.Auth.Email.TimeOut != 10*time.Second && s.SMTP.TimeOut == 10*time.Second {
		s.SMTP.TimeOut = s.Auth.Email.TimeOut
		result = append(result, DeprecatedFlag{Old: "auth.email.timeout", New: "smtp.timeout", Version: "1.5"})
	}
	if s.Auth.Email.MsgTemplate != "email_confirmation_login.html.tmpl" {
		result = append(result, DeprecatedFlag{Old: "auth.email.template", Version: "1.5"})
	}
	if s.LegacyImageProxy && !s.ImageProxy.HTTP2HTTPS {
		s.ImageProxy.HTTP2HTTPS = s.LegacyImageProxy
		result = append(result, DeprecatedFlag{Old: "img-proxy", New: "image-proxy.http2https", Version: "1.5"})
	}
	if len(s.Notify.Type) != 0 && (len(s.Notify.Users) != 0 || len(s.Notify.Admins) != 0) {
		s.handleDeprecatedNotifications()
		result = append(result, DeprecatedFlag{Old: "notify.type", New: "notify.(users|admins)", Version: "1.9"})
	}
	if s.Notify.Email.AdminNotifications && !contains("email", s.Notify.Admins) {
		s.Notify.Admins = append(s.Notify.Admins, "email")
		result = append(result, DeprecatedFlag{Old: "notify.email.notify_admin", New: "notify.admins=email", Version: "1.9"})
	}
	if s.Notify.Telegram.Token != "" && s.Telegram.Token == "" {
		s.Telegram.Token = s.Notify.Telegram.Token
		result = append(result, DeprecatedFlag{Old: "notify.telegram.token", New: "telegram.token", Version: "1.9"})
	}
	const telegramDefaultDuration = time.Second * 5
	if s.Notify.Telegram.Timeout != telegramDefaultDuration && s.Telegram.Timeout == telegramDefaultDuration {
		s.Telegram.Timeout = s.Notify.Telegram.Timeout
		result = append(result, DeprecatedFlag{Old: "notify.telegram.timeout", New: "telegram.timeout", Version: "1.9"})
	}
	if s.Notify.Telegram.API != "https://api.telegram.org/bot" {
		result = append(result, DeprecatedFlag{Old: "notify.telegram.api", Version: "1.9"})
	}
	return result
}

func (s *ServerCommand) handleDeprecatedNotifications() {
	for _, t := range s.Notify.Type {
		if t == "email" && !contains(t, s.Notify.Users) {
			s.Notify.Users = append(s.Notify.Users, t)
		}
		if (t == "telegram" || t == "slack") && !contains(t, s.Notify.Admins) {
			s.Notify.Admins = append(s.Notify.Admins, t)
		}
	}
}

func contains(s string, a []string) bool {
	for _, t := range a {
		if t == s {
			return true
		}
	}
	return false
}

// newServerApp prepares application and return it with all active parts
// doesn't start anything
func (s *ServerCommand) newServerApp(ctx context.Context) (*serverApp, error) {

	if err := makeDirs(s.BackupLocation); err != nil {
		return nil, errors.Wrap(err, "failed to create backup store")
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
	log.Printf("[DEBUG] image service for url=%s, EditDuration=%v", imageService.ImageAPI, imageService.EditDuration)

	dataService := &service.DataStore{
		Engine:                 storeEngine,
		EditDuration:           s.EditDuration,
		AdminEdits:             s.AdminEdit,
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
		_ = dataService.Close()
		return nil, errors.Wrap(err, "failed to make cache")
	}

	avatarStore, err := s.makeAvatarStore()
	if err != nil {
		_ = dataService.Close()
		return nil, errors.Wrap(err, "failed to make avatar store")
	}
	authRefreshCache := newAuthRefreshCache()
	authenticator, err := s.makeAuthenticator(ctx, dataService, avatarStore, adminStore, authRefreshCache)
	if err != nil {
		_ = dataService.Close()
		return nil, errors.Wrap(err, "failed to make authenticator")
	}

	exporter := &migrator.Native{DataStore: dataService}

	migr := &api.Migrator{
		Cache:             loadingCache,
		NativeImporter:    &migrator.Native{DataStore: dataService},
		DisqusImporter:    &migrator.Disqus{DataStore: dataService},
		WordPressImporter: &migrator.WordPress{DataStore: dataService},
		CommentoImporter:  &migrator.Commento{DataStore: dataService},
		NativeExporter:    &migrator.Native{DataStore: dataService},
		URLMapperMaker:    migrator.NewURLMapper,
		KeyStore:          adminStore,
	}

	var emailNotifications bool
	notifyService, telegramBotUsername, err := s.makeNotify(dataService, authenticator)

	if contains("email", s.Notify.Users) {
		emailNotifications = true
	}

	// we pass telegramBotUsername to Rest server only if user notifications are enabled
	if !contains("telegram", s.Notify.Users) {
		telegramBotUsername = ""
	}

	if err != nil {
		log.Printf("[WARN] failed to make notify service, %s", err)
		notifyService = notify.NopService // disable notifier
		emailNotifications = false        // email notifications are not available in this case
		telegramBotUsername = ""          // telegram notifications are not available in this case either
	}

	imgProxy := &proxy.Image{
		HTTP2HTTPS:    s.ImageProxy.HTTP2HTTPS,
		CacheExternal: s.ImageProxy.CacheExternal,
		RoutePath:     "/api/v1/img",
		RemarkURL:     s.RemarkURL,
		ImageService:  imageService,
	}
	emojiFmt := store.CommentConverterFunc(func(text string) string { return text })
	if s.EnableEmoji {
		emojiFmt = func(text string) string { return emoji.Sprint(text) }
	}
	commentFormatter := store.NewCommentFormatter(imgProxy, emojiFmt)

	sslConfig, err := s.makeSSLConfig()
	if err != nil {
		_ = dataService.Close()
		return nil, errors.Wrap(err, "failed to make config of ssl server params")
	}

	srv := &api.Rest{
		Version:             s.Revision,
		DataService:         dataService,
		WebRoot:             s.WebRoot,
		RemarkURL:           s.RemarkURL,
		ImageProxy:          imgProxy,
		CommentFormatter:    commentFormatter,
		Migrator:            migr,
		ReadOnlyAge:         s.ReadOnlyAge,
		SharedSecret:        s.SharedSecret,
		Authenticator:       authenticator,
		Cache:               loadingCache,
		NotifyService:       notifyService,
		SSLConfig:           sslConfig,
		UpdateLimiter:       s.UpdateLimit,
		ImageService:        imageService,
		EmailNotifications:  emailNotifications,
		TelegramBotUsername: telegramBotUsername,
		EmojiEnabled:        s.EnableEmoji,
		AnonVote:            s.AnonymousVote && s.RestrictVoteIP,
		SimpleView:          s.SimpleView,
		ProxyCORS:           s.ProxyCORS,
		AllowedAncestors:    s.AllowedHosts,
		SendJWTHeader:       s.Auth.SendJWTHeader,
	}

	srv.ScoreThresholds.Low, srv.ScoreThresholds.Critical = s.LowScore, s.CriticalScore

	var devAuth *provider.DevAuthServer
	if s.Auth.Dev {
		da, errDevAuth := authenticator.DevAuth()
		if errDevAuth != nil {
			_ = dataService.Close()
			return nil, errors.Wrap(errDevAuth, "can't make dev oauth2 server")
		}
		devAuth = da
	}

	return &serverApp{
		ServerCommand:    s,
		restSrv:          srv,
		migratorSrv:      migr,
		exporter:         exporter,
		devAuth:          devAuth,
		dataService:      dataService,
		avatarStore:      avatarStore,
		notifyService:    notifyService,
		imageService:     imageService,
		authenticator:    authenticator,
		terminated:       make(chan struct{}),
		authRefreshCache: authRefreshCache,
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
	}()

	a.activateBackup(ctx) // runs in goroutine for each site
	if a.Auth.Dev {
		go a.devAuth.Run(ctx) // dev oauth2 server on :8084
	}

	// staging images resubmit after restart of the app
	if e := a.dataService.ResubmitStagingImages(a.Sites); e != nil {
		log.Printf("[WARN] failed to resubmit comments with staging images, %s", e)
	}

	go a.imageService.Cleanup(ctx) // pictures cleanup for staging images

	a.restSrv.Run(a.Address, a.Port)

	// shutdown procedures after HTTP server is stopped
	if a.devAuth != nil {
		a.devAuth.Shutdown()
	}
	if e := a.dataService.Close(); e != nil {
		log.Printf("[WARN] failed to close data store, %s", e)
	}
	if e := a.avatarStore.Close(); e != nil {
		log.Printf("[WARN] failed to close avatar store, %s", e)
	}
	if e := a.restSrv.Cache.Close(); e != nil {
		log.Printf("[WARN] failed to close rest server cache, %s", e)
	}
	if e := a.authRefreshCache.Close(); e != nil {
		log.Printf("[WARN] failed to close auth authRefreshCache, %s", e)
	}
	a.notifyService.Close()
	// call potentially infinite loop with cancellation after a minute as a safeguard
	minuteCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	a.imageService.Close(minuteCtx)

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
			return nil, errors.Wrap(err, "failed to create avatar store")
		}
		return avatar.NewLocalFS(s.Avatar.FS.Path), nil
	case "bolt":
		if err := makeDirs(path.Dir(s.Avatar.Bolt.File)); err != nil {
			return nil, errors.Wrap(err, "failed to create avatar store")
		}
		return avatar.NewBoltDB(s.Avatar.Bolt.File, bolt.Options{})
	case "uri":
		return avatar.NewStore(s.Avatar.URI)
	}
	return nil, errors.Errorf("unsupported avatar store type %s", s.Avatar.Type)
}

func (s *ServerCommand) makePicturesStore() (*image.Service, error) {
	imageServiceParams := image.ServiceParams{
		ImageAPI:     s.RemarkURL + "/api/v1/picture/",
		ProxyAPI:     s.RemarkURL + "/api/v1/img",
		EditDuration: s.EditDuration,
		MaxSize:      s.Image.MaxSize,
		MaxHeight:    s.Image.ResizeHeight,
		MaxWidth:     s.Image.ResizeWidth,
	}
	switch s.Image.Type {
	case "bolt":
		boltImageStore, err := image.NewBoltStorage(s.Image.Bolt.File, bolt.Options{})
		if err != nil {
			return nil, err
		}
		return image.NewService(boltImageStore, imageServiceParams), nil
	case "fs":
		if err := makeDirs(s.Image.FS.Path); err != nil {
			return nil, errors.Wrap(err, "failed to create pictures store")
		}
		return image.NewService(&image.FileSystem{
			Location:   s.Image.FS.Path,
			Staging:    s.Image.FS.Staging,
			Partitions: s.Image.FS.Partitions,
		}, imageServiceParams), nil
	case "rpc":
		return image.NewService(&image.RPC{
			Client: jrpc.Client{
				API:        s.Image.RPC.API,
				Client:     http.Client{Timeout: s.Image.RPC.TimeOut},
				AuthUser:   s.Image.RPC.AuthUser,
				AuthPasswd: s.Image.RPC.AuthPassword,
			}}, imageServiceParams), nil
	}
	return nil, errors.Errorf("unsupported pictures store type %s", s.Image.Type)
}

func (s *ServerCommand) makeAdminStore() (admin.Store, error) {
	log.Printf("[INFO] make admin store, type=%s", s.Admin.Type)

	switch s.Admin.Type {
	case "shared":
		sharedAdminEmail := ""
		if len(s.Admin.Shared.Email) == 0 { // no admin email, use admin@domain
			if u, err := url.Parse(s.RemarkURL); err == nil {
				sharedAdminEmail = "admin@" + u.Host
			}
		} else {
			sharedAdminEmail = s.Admin.Shared.Email[0]
		}
		return admin.NewStaticStore(s.SharedSecret, s.Sites, s.Admin.Shared.Admins, sharedAdminEmail), nil
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
	case "redis_pub_sub":
		redisPubSub, err := eventbus.NewRedisPubSub(s.Cache.RedisAddr, "remark42-cache")
		if err != nil {
			return nil, errors.Wrap(err, "cache backend initialization, redis PubSub initialisation")
		}
		backend, err := cache.NewLruCache(cache.MaxCacheSize(s.Cache.Max.Size), cache.MaxValSize(s.Cache.Max.Value),
			cache.MaxKeys(s.Cache.Max.Items), cache.EventBus(redisPubSub))
		if err != nil {
			return nil, errors.Wrap(err, "cache backend initialization")
		}
		return cache.NewScache(backend), nil
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

func (s *ServerCommand) addAuthProviders(ctx context.Context, authenticator *auth.Service) error {

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
	if s.Auth.Microsoft.CID != "" && s.Auth.Microsoft.CSEC != "" {
		authenticator.AddProvider("microsoft", s.Auth.Microsoft.CID, s.Auth.Microsoft.CSEC)
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
	if s.Auth.Telegram {
		telegram := &provider.TelegramHandler{
			ProviderName: "telegram",
			ErrorMsg:     "❌ Invalid auth request. Please try clicking link again.",
			SuccessMsg:   "✅ You have successfully authenticated!",
			Telegram:     provider.NewTelegramAPI(s.Telegram.Token, &http.Client{Timeout: s.Telegram.Timeout}),
			L:            log.Default(),
			TokenService: authenticator.TokenService(),
			AvatarSaver:  authenticator.AvatarProxy(),
		}
		// Run Telegram provider in the background
		go func() {
			err := telegram.Run(ctx)
			if err != nil {
				log.Printf("[ERROR] telegram auth error %+v", err)
			}
		}()
		authenticator.AddCustomHandler(telegram)

		providers++
	}

	if s.Auth.Dev {
		log.Print("[INFO] dev access enabled")
		authenticator.AddProvider("dev", "", "")
		providers++
	}

	if s.Auth.Email.Enable {
		params := sender.EmailParams{
			Host:         s.SMTP.Host,
			Port:         s.SMTP.Port,
			SMTPUserName: s.SMTP.Username,
			SMTPPassword: s.SMTP.Password,
			TimeOut:      s.SMTP.TimeOut,
			TLS:          s.SMTP.TLS,
			From:         s.Auth.Email.From,
			Subject:      s.Auth.Email.Subject,
			ContentType:  s.Auth.Email.ContentType,
		}
		sndr := sender.NewEmailClient(params, log.Default())
		tmpl, err := s.loadEmailTemplate()
		if err != nil {
			return err
		}
		authenticator.AddVerifProvider("email", tmpl, sndr)
	}

	if s.Auth.Anonymous {
		log.Print("[INFO] anonymous access enabled")
		var isValidAnonName = regexp.MustCompile(`^[\p{L}\d_ ]+$`).MatchString
		authenticator.AddDirectProvider("anonymous", provider.CredCheckerFunc(func(user, _ string) (ok bool, err error) {

			// don't allow anon with space prefix or suffix
			if strings.HasPrefix(user, " ") || strings.HasSuffix(user, " ") {
				log.Printf("[WARN] name %q has space as a suffix or prefix", user)
				return false, nil
			}

			user = strings.TrimSpace(user)
			if len(user) < 3 {
				log.Printf("[WARN] name %q is too short, should be at least 3 characters", user)
				return false, nil
			}
			if len(user) > 64 {
				log.Printf("[WARN] name %q is too long, should be up to 64 characters", user)
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

	return nil
}

// loadEmailTemplate trying to get template from statik
func (s *ServerCommand) loadEmailTemplate() (string, error) {
	var file []byte
	var err error

	if s.Auth.Email.MsgTemplate == "email_confirmation_login.html.tmpl" {
		fs := templates.NewFS()
		file, err = fs.ReadFile(s.Auth.Email.MsgTemplate)
	} else {
		// deprecated loading from an external file, should be removed before v1.9.0
		file, err = ioutil.ReadFile(s.Auth.Email.MsgTemplate)
		log.Printf("[INFO] template %s will be read from disk", s.Auth.Email.MsgTemplate)
	}

	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", s.Auth.Email.MsgTemplate)
	}

	return string(file), nil
}

// aside from notify.Service and error, returns telegram bot name which will be passed to the frontend
func (s *ServerCommand) makeNotify(dataStore *service.DataStore, authenticator *auth.Service) (*notify.Service, string, error) {
	notifyService := notify.NopService
	var destinations []notify.Destination
	var telegramBotUsername string

	if contains("webhook", s.Notify.Admins) {
		client := &http.Client{Timeout: 5 * time.Second}
		webhookHeaders := s.Notify.Webhook.Headers
		if len(webhookHeaders) == 0 {
			webhookHeaders = splitAtCommas(os.Getenv("NOTIFY_WEBHOOK_HEADERS")) // env value may have comma inside "", parsed separately
		}

		whParams := notify.WebhookParams{
			WebhookURL: s.Notify.Webhook.WebhookURL,
			Template:   s.Notify.Webhook.Template,
			Headers:    webhookHeaders,
		}
		webhook, err := notify.NewWebhook(client, whParams)
		if err != nil {
			return nil, "", errors.Wrap(err, "failed to create webhook notification destination")
		}
		destinations = append(destinations, webhook)
	}

	if contains("slack", s.Notify.Admins) {
		slack, err := notify.NewSlack(s.Notify.Slack.Token, s.Notify.Slack.Channel)
		if err != nil {
			return nil, "", errors.Wrap(err, "failed to create slack notification destination")
		}
		destinations = append(destinations, slack)
	}

	if contains("telegram", s.Notify.Users) || contains("telegram", s.Notify.Admins) {
		if contains("telegram", s.Notify.Admins) && s.Notify.Telegram.Channel == "" {
			return nil, "", errors.New("--notify.telegram.channel must be set for admin notifications to work")
		}
		telegramParams := notify.TelegramParams{
			AdminChannelID:    s.Notify.Telegram.Channel,
			UserNotifications: contains("telegram", s.Notify.Users),
			Token:             s.Telegram.Token,
			Timeout:           s.Telegram.Timeout,
		}
		tg, err := notify.NewTelegram(telegramParams)
		if err != nil {
			return nil, "", errors.Wrap(err, "failed to create telegram notification destination")
		}
		destinations = append(destinations, tg)
		telegramBotUsername = tg.BotUsername
	}

	// with logic below admin notifications enable notifications for users on the backend even if they
	// are not enabled explicitly, however they won't be visible to the users in the frontend
	// because api.Rest.EmailNotifications would be set to false.
	if contains("email", s.Notify.Users) || contains("email", s.Notify.Admins) {
		emailParams := notify.EmailParams{
			MsgTemplatePath:          s.emailMsgTemplatePath,
			VerificationTemplatePath: s.emailVerificationTemplatePath, From: s.Notify.Email.From,
			VerificationSubject: s.Notify.Email.VerificationSubject,
			UnsubscribeURL:      s.RemarkURL + "/email/unsubscribe.html",
			// TODO: uncomment after #560 frontend part is ready and URL is known
			// SubscribeURL:        s.RemarkURL + "/subscribe.html?token=",
			TokenGenFn: func(userID, email, site string) (string, error) {
				claims := token.Claims{
					Handshake: &token.Handshake{ID: userID + "::" + email},
					StandardClaims: jwt.StandardClaims{
						Audience:  site,
						ExpiresAt: time.Now().Add(100 * 365 * 24 * time.Hour).Unix(),
						NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
						Issuer:    "remark42",
					},
				}
				tkn, err := authenticator.TokenService().Token(claims)
				if err != nil {
					return "", errors.Wrapf(err, "failed to make unsubscription token")
				}
				return tkn, nil
			},
		}
		if contains("email", s.Notify.Admins) {
			emailParams.AdminEmails = s.Admin.Shared.Email
		}
		smtpParams := notify.SMTPParams{
			Host:     s.SMTP.Host,
			Port:     s.SMTP.Port,
			TLS:      s.SMTP.TLS,
			Username: s.SMTP.Username,
			Password: s.SMTP.Password,
			TimeOut:  s.SMTP.TimeOut,
		}
		emailService, err := notify.NewEmail(emailParams, smtpParams)
		if err != nil {
			return nil, "", errors.Wrap(err, "failed to create email notification destination")
		}
		destinations = append(destinations, emailService)
	}

	if len(destinations) > 0 {
		log.Printf("[INFO] make notify, for users: %s, for admins: %s", s.Notify.Users, s.Notify.Admins)
		notifyService = notify.NewService(dataStore, s.Notify.QueueSize, destinations...)
	}
	return notifyService, telegramBotUsername, nil
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
		} else if s.Admin.Type == "shared" && len(s.Admin.Shared.Email) != 0 {
			config.ACMEEmail = s.Admin.Shared.Email[0]
		} else if u, e := url.Parse(s.RemarkURL); e == nil {
			config.ACMEEmail = "admin@" + u.Hostname()
		}
	}
	return config, err
}

func (s *ServerCommand) makeAuthenticator(ctx context.Context, ds *service.DataStore, avas avatar.Store, admns admin.Store, authRefreshCache *authRefreshCache) (*auth.Service, error) {
	authenticator := auth.NewService(auth.Opts{
		URL:            strings.TrimSuffix(s.RemarkURL, "/"),
		Issuer:         "remark42",
		TokenDuration:  s.Auth.TTL.JWT,
		CookieDuration: s.Auth.TTL.Cookie,
		SendJWTHeader:  s.Auth.SendJWTHeader,
		SameSiteCookie: s.parseSameSite(s.Auth.SameSite),
		SecureCookies:  strings.HasPrefix(s.RemarkURL, "https://"),
		SecretReader: token.SecretFunc(func(aud string) (string, error) { // get secret per site
			return admns.Key("")
		}),
		ClaimsUpd: token.ClaimsUpdFunc(func(c token.Claims) token.Claims { // set attributes, on new token or refresh
			if c.User == nil {
				return c
			}
			c.User.SetAdmin(ds.IsAdmin(c.Audience, c.User.ID))
			c.User.SetBoolAttr("blocked", ds.IsBlocked(c.Audience, c.User.ID))
			var err error
			c.User.Email, err = ds.GetUserEmail(c.Audience, c.User.ID)
			if err != nil {
				log.Printf("[WARN] can't read email for %s, %v", c.User.ID, err)
			}

			// don't allow anonymous and email with admins names
			// exclude admin from impersonation detection over email, it prevents a valid admin to login with RestrictedNames
			if strings.HasPrefix(c.User.ID, "anonymous_") || (strings.HasPrefix(c.User.ID, "email_") && !c.User.IsAdmin()) {
				for _, a := range s.RestrictedNames {
					if strings.EqualFold(strings.TrimSpace(c.User.Name), a) {
						c.User.SetBoolAttr("blocked", true)
						log.Printf("[INFO] blocked %+v, attempt to impersonate (restricted names)", c.User)
						break
					}
				}
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
		RefreshCache:      authRefreshCache,
		UseGravatar:       true,
	})

	if err := s.addAuthProviders(ctx, authenticator); err != nil {
		return nil, err
	}

	return authenticator, nil
}

func (s *ServerCommand) parseSameSite(ss string) http.SameSite {
	switch strings.ToLower(ss) {
	case "default":
		return http.SameSiteDefaultMode
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		return http.SameSiteLaxMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return http.SameSiteDefaultMode
	}
}

// splitAtCommas split s at commas, ignoring commas in strings.
// Eliminate leading and trailing dbl quotes in each element only if both presented
// based on https://stackoverflow.com/a/59318708
func splitAtCommas(s string) []string {

	cleanup := func(s string) string {
		if s == "" {
			return s
		}
		res := strings.TrimSpace(s)
		if res[0] == '"' && res[len(res)-1] == '"' {
			res = strings.TrimPrefix(res, `"`)
			res = strings.TrimSuffix(res, `"`)
		}
		return res
	}

	var res []string
	var beg int
	var inString bool

	for i := 0; i < len(s); i++ {
		if s[i] == ',' && !inString {
			res = append(res, cleanup(s[beg:i]))
			beg = i + 1
			continue
		}

		if s[i] == '"' {
			if !inString {
				inString = true
			} else if i > 0 && s[i-1] != '\\' { // also allow \"
				inString = false
			}
		}
	}
	res = append(res, cleanup(s[beg:]))
	if len(res) == 1 && res[0] == "" {
		return []string{}
	}
	return res
}

// authRefreshCache used by authenticator to minimize repeatable token refreshes
type authRefreshCache struct {
	cache.LoadingCache
}

func newAuthRefreshCache() *authRefreshCache {
	expirableCache, _ := cache.NewExpirableCache(cache.TTL(5 * time.Minute))
	return &authRefreshCache{LoadingCache: expirableCache}
}

// Get implements cache getter with key converted to string
func (c *authRefreshCache) Get(key interface{}) (interface{}, bool) {
	return c.LoadingCache.Peek(key.(string))
}

// Set implements cache setter with key converted to string
func (c *authRefreshCache) Set(key, value interface{}) {
	_, _ = c.LoadingCache.Get(key.(string), func() (interface{}, error) { return value, nil })
}
