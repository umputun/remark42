package cmd

import (
	"cmp"
	"context"
	"crypto/sha1" //nolint:gosec // used only for stable ID hashing, not for security
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/go-pkgz/jrpc"
	"github.com/go-pkgz/lcw/v2/eventbus"
	log "github.com/go-pkgz/lgr"
	ntf "github.com/go-pkgz/notify"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kyokomi/emoji/v2"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/oauth2"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/provider/sender"
	"github.com/go-pkgz/auth/v2/token"
	cache "github.com/go-pkgz/lcw/v2"

	"github.com/umputun/remark42/backend/app/migrator"
	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/providers"
	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/rest/api"
	"github.com/umputun/remark42/backend/app/rest/proxy"
	"github.com/umputun/remark42/backend/app/safehttp"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/image"
	"github.com/umputun/remark42/backend/app/store/service"
	"github.com/umputun/remark42/backend/app/templates"
)

//go:embed web
var webFS embed.FS

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

	Sites                      []string      `long:"site" env:"SITE" default:"remark" description:"site names" env-delim:","`
	AnonymousVote              bool          `long:"anon-vote" env:"ANON_VOTE" description:"enable anonymous votes (works only with VOTES_IP enabled)"`
	AdminPasswd                string        `long:"admin-passwd" env:"ADMIN_PASSWD" default:"" description:"admin basic auth password"`
	BackupLocation             string        `long:"backup" env:"BACKUP_PATH" default:"./var/backup" description:"backups location"`
	MaxBackupFiles             int           `long:"max-back" env:"MAX_BACKUP_FILES" default:"10" description:"max backups to keep"`
	LegacyImageProxy           bool          `long:"img-proxy" env:"IMG_PROXY" description:"[deprecated, use image-proxy.http2https] enable image proxy"`
	MinCommentSize             int           `long:"min-comment" env:"MIN_COMMENT_SIZE" default:"0" description:"min comment size"`
	MaxCommentSize             int           `long:"max-comment" env:"MAX_COMMENT_SIZE" default:"2048" description:"max comment size"`
	MaxVotes                   int           `long:"max-votes" env:"MAX_VOTES" default:"-1" description:"maximum number of votes per comment"`
	RestrictVoteIP             bool          `long:"votes-ip" env:"VOTES_IP" description:"restrict votes from the same ip"`
	DurationVoteIP             time.Duration `long:"votes-ip-time" env:"VOTES_IP_TIME" default:"5m" description:"same ip vote duration"`
	LowScore                   int           `long:"low-score" env:"LOW_SCORE" default:"-5" description:"low score threshold"`
	CriticalScore              int           `long:"critical-score" env:"CRITICAL_SCORE" default:"-10" description:"critical score threshold"`
	PositiveScore              bool          `long:"positive-score" env:"POSITIVE_SCORE" description:"enable positive score only"`
	ReadOnlyAge                int           `long:"read-age" env:"READONLY_AGE" default:"0" description:"read-only age of comments, days"`
	EditDuration               time.Duration `long:"edit-time" env:"EDIT_TIME" default:"5m" description:"edit window; 0 hides editing in the widget and disables staged image cleanup, but the API does not limit edit time"`
	AdminEdit                  bool          `long:"admin-edit" env:"ADMIN_EDIT" description:"unlimited edit for admins"`
	Port                       int           `long:"port" env:"REMARK_PORT" default:"8080" description:"port"`
	Address                    string        `long:"address" env:"REMARK_ADDRESS" default:"" description:"listening address"`
	WebRoot                    string        `long:"web-root" env:"REMARK_WEB_ROOT" default:"./web" description:"web root directory"`
	UpdateLimit                float64       `long:"update-limit" env:"UPDATE_LIMIT" default:"0.5" description:"updates/sec limit"`
	TrustedProxies             []string      `long:"trusted-proxy" env:"TRUSTED_PROXY" description:"reverse-proxy networks (CIDR or IP) trusted to set the client IP; if unset, trusted from any client (see docs)" env-delim:","`
	RestrictedWords            []string      `long:"restricted-words" env:"RESTRICTED_WORDS" description:"words prohibited to use in comments" env-delim:","`
	RestrictedWordsFile        string        `long:"restricted-words-file" env:"RESTRICTED_WORDS_FILE" description:"file with words prohibited to use in comments, one per line, re-read on change"`
	RestrictedNames            []string      `long:"restricted-names" env:"RESTRICTED_NAMES" description:"names prohibited to use by user" env-delim:","`
	EnableEmoji                bool          `long:"emoji" env:"EMOJI" description:"enable emoji"`
	SimpleView                 bool          `long:"simple-view" env:"SIMPLE_VIEW" description:"minimal comment editor mode"`
	ProxyCORS                  bool          `long:"proxy-cors" env:"PROXY_CORS" description:"disable internal CORS and delegate it to proxy"`
	AllowedHosts               []string      `long:"allowed-hosts" env:"ALLOWED_HOSTS" description:"limit hosts/sources allowed to embed comments via CSP 'frame-ancestors'" env-delim:","`
	SubscribersOnly            bool          `long:"subscribers-only" env:"SUBSCRIBERS_ONLY" description:"enable commenting only for Patreon subscribers"`
	DisableSignature           bool          `long:"disable-signature" env:"DISABLE_SIGNATURE" description:"disable server signature in headers"`
	DisableFancyTextFormatting bool          `long:"disable-fancy-text-formatting" env:"DISABLE_FANCY_TEXT_FORMATTING" description:"disable fancy comments text formatting (replacement of quotes, dashes, fractions, etc)"`

	Auth struct {
		TTL struct {
			JWT    time.Duration `long:"jwt" env:"JWT" default:"5m" description:"JWT TTL"`
			Cookie time.Duration `long:"cookie" env:"COOKIE" default:"200h" description:"auth cookie TTL"`
		} `group:"ttl" namespace:"ttl" env-namespace:"TTL"`

		SendJWTHeader bool   `long:"send-jwt-header" env:"SEND_JWT_HEADER" description:"also send JWT as a header, so the frontend can store it in a client-side cookie that survives third-party cookie blocking; server-set cookies are still sent (note: increases vulnerability to XSS attacks)"`
		SameSite      string `long:"same-site" env:"SAME_SITE" description:"set same site policy for cookies" choice:"default" choice:"none" choice:"lax" choice:"strict" default:"default"` // nolint

		Apple     AppleGroup         `group:"apple" namespace:"apple" env-namespace:"APPLE" description:"Apple OAuth"`
		Google    AuthGroup          `group:"google" namespace:"google" env-namespace:"GOOGLE" description:"Google OAuth"`
		Github    AuthGroup          `group:"github" namespace:"github" env-namespace:"GITHUB" description:"Github OAuth"`
		Facebook  AuthGroup          `group:"facebook" namespace:"facebook" env-namespace:"FACEBOOK" description:"Facebook OAuth"`
		Microsoft MicrosoftAuthGroup `group:"microsoft" namespace:"microsoft" env-namespace:"MICROSOFT" description:"Microsoft OAuth"`
		Yandex    AuthGroup          `group:"yandex" namespace:"yandex" env-namespace:"YANDEX" description:"Yandex OAuth"`
		Twitter   AuthGroup          `group:"twitter" namespace:"twitter" env-namespace:"TWITTER" description:"X (Twitter) OAuth"`
		Patreon   AuthGroup          `group:"patreon" namespace:"patreon" env-namespace:"PATREON" description:"Patreon OAuth"`
		Discord   AuthGroup          `group:"discord" namespace:"discord" env-namespace:"DISCORD" description:"Discord OAuth"`
		Custom    CustomAuthGroup    `group:"custom" namespace:"custom" env-namespace:"CUSTOM" description:"Custom OAuth2 provider"`
		Telegram  bool               `long:"telegram" env:"TELEGRAM" description:"Enable Telegram auth (using token from telegram.token)"`
		Dev       bool               `long:"dev" env:"DEV" description:"enable dev (local) oauth2"`
		Anonymous bool               `long:"anon" env:"ANON" description:"enable anonymous login"`
		Email     struct {
			Enable       bool          `long:"enable" env:"ENABLE" description:"enable auth via email"`
			From         string        `long:"from" env:"FROM" description:"from email address"`
			Subject      string        `long:"subj" env:"SUBJ" default:"remark42 confirmation" description:"email's subject"`
			ContentType  string        `long:"content-type" env:"CONTENT_TYPE" default:"text/html" description:"content type"`
			Host         string        `long:"host" env:"HOST" description:"[deprecated, use --smtp.host] SMTP host"`
			Port         int           `long:"port" env:"PORT" description:"[deprecated, use --smtp.port] SMTP port"`
			SMTPPassword string        `long:"passwd" env:"PASSWD" description:"[deprecated, use --smtp.password] SMTP password"`
			SMTPUserName string        `long:"user" env:"USER" description:"[deprecated, use --smtp.username] SMTP user name"`
			TLS          bool          `long:"tls" env:"TLS" description:"[deprecated, use --smtp.tls] enable TLS"`
			TimeOut      time.Duration `long:"timeout" env:"TIMEOUT" default:"10s" description:"[deprecated, use --smtp.timeout] SMTP TCP connection timeout"`
			MsgTemplate  string        `long:"template" env:"TEMPLATE" description:"[deprecated] message template file" default:"email_confirmation_login.html.tmpl"`
		} `group:"email" namespace:"email" env-namespace:"EMAIL"`
	} `group:"auth" namespace:"auth" env-namespace:"AUTH"`

	CommonOpts
}

// ImageProxyGroup defines options group for image proxy
type ImageProxyGroup struct {
	HTTP2HTTPS    bool `long:"http2https" env:"HTTP2HTTPS" description:"enable HTTP->HTTPS proxy, no effect when remark url is http"`
	CacheExternal bool `long:"cache-external" env:"CACHE_EXTERNAL" description:"enable caching for external images"`
}

// AppleGroup defines options for Apple auth params
type AppleGroup struct {
	CID                string `long:"cid" env:"CID" description:"Apple client ID (App ID or Services ID)"`
	TID                string `long:"tid" env:"TID" description:"Apple Team ID"`
	KID                string `long:"kid" env:"KID" description:"Private key ID"`
	PrivateKeyFilePath string `long:"private-key-filepath" env:"PRIVATE_KEY_FILEPATH" description:"Private key file location" default:"/srv/var/apple.p8"`
}

// AuthGroup defines options group for auth params
type AuthGroup struct {
	CID  string `long:"cid" env:"CID" description:"OAuth client ID"`
	CSEC string `long:"csec" env:"CSEC" description:"OAuth client secret"`
}

// MicrosoftAuthGroup defines options group for Microsoft auth params
type MicrosoftAuthGroup struct {
	CID    string `long:"cid" env:"CID" description:"OAuth client ID"`
	CSEC   string `long:"csec" env:"CSEC" description:"OAuth client secret"`
	Tenant string `long:"tenant" env:"TENANT" description:"Azure AD tenant ID, domain, or 'common' (default)" default:"common"`
}

// CustomAuthGroup defines options group for custom OAuth2 provider params
type CustomAuthGroup struct {
	Name         string   `long:"name" env:"NAME" description:"custom provider name used in auth route"`
	CID          string   `long:"cid" env:"CID" description:"OAuth client ID"`
	CSEC         string   `long:"csec" env:"CSEC" description:"OAuth client secret"`
	AuthURL      string   `long:"auth-url" env:"AUTH_URL" description:"OAuth authorization endpoint"`
	TokenURL     string   `long:"token-url" env:"TOKEN_URL" description:"OAuth token endpoint"`
	InfoURL      string   `long:"info-url" env:"INFO_URL" description:"OAuth user info endpoint"`
	Scopes       []string `long:"scopes" env:"SCOPES" env-delim:"," description:"OAuth scopes"`
	IDField      string   `long:"id-field" env:"ID_FIELD" default:"sub" description:"user info field used as unique id"`
	NameField    string   `long:"name-field" env:"NAME_FIELD" default:"name" description:"user info field used as display name"`
	PictureField string   `long:"picture-field" env:"PICTURE_FIELD" default:"picture" description:"user info field used as avatar url"`
	EmailField   string   `long:"email-field" env:"EMAIL_FIELD" default:"email" description:"user info field used as email"`
}

// StoreGroup defines options group for store params
type StoreGroup struct {
	Type string `long:"type" env:"TYPE" description:"type of storage" choice:"bolt" choice:"rpc" default:"bolt"` // nolint
	Bolt struct {
		Path    string        `long:"path" env:"PATH" default:"./var" description:"parent directory for the bolt files"`
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
	} `group:"bolt" namespace:"bolt" env-namespace:"BOLT"`
	MaxSize      int      `long:"max-size" env:"MAX_SIZE" default:"5000000" description:"max size of image file"`
	ResizeWidth  int      `long:"resize-width" env:"RESIZE_WIDTH" default:"2400" description:"width of a resized image"`
	ResizeHeight int      `long:"resize-height" env:"RESIZE_HEIGHT" default:"900" description:"height of a resized image"`
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
	} `group:"bolt" namespace:"bolt" env-namespace:"BOLT"`
	URI           string `long:"uri" env:"URI" default:"./var/avatars" description:"avatars store URI"`
	RszLmt        int    `long:"rsz-lmt" env:"RESIZE" default:"0" description:"max avatar width and height in pixels; larger avatars are scaled down on save, keeping the aspect ratio"`
	IdenticonSize int    `long:"identicon-size" env:"IDENTICON_SIZE" default:"48" description:"width and height in pixels of the avatar generated for users without a picture"`
}

// CacheGroup defines options group for cache params
type CacheGroup struct {
	Type      string `long:"type" env:"TYPE" description:"type of cache" choice:"redis_pub_sub" choice:"mem" choice:"none" default:"mem"` // nolint
	RedisAddr string `long:"redis_addr" env:"REDIS_ADDR" default:"127.0.0.1:6379" description:"address of Redis PubSub instance, turn redis_pub_sub cache on for distributed cache"`
	Max       struct {
		Items int   `long:"items" env:"ITEMS" default:"1000" description:"max cached items"`
		Value int   `long:"value" env:"VALUE" default:"0" description:"max size of the cached value, 0 - unlimited"`
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
	RPC AdminRPCGroup `group:"rpc" namespace:"rpc" env-namespace:"RPC"`
}

// TelegramGroup defines token for Telegram used in notify and auth modules
type TelegramGroup struct {
	Token   string        `long:"token" env:"TOKEN" description:"telegram token (used for auth and telegram notifications)"`
	Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"5s" description:"telegram timeout"`
	APIURL  string        `long:"api-url" env:"API_URL" description:"telegram bot api base url, for a proxy or a test double; the public api when unset"`
}

// SMTPGroup defines options for SMTP server connection, used in auth and notify modules
type SMTPGroup struct {
	Host               string        `long:"host" env:"HOST" description:"SMTP host"`
	Port               int           `long:"port" env:"PORT" description:"SMTP port"`
	HELOHost           string        `long:"helo_host" env:"HELO_HOST" description:"SMTP HELO/EHLO hostname"`
	Username           string        `long:"username" env:"USERNAME" description:"SMTP user name"`
	Password           string        `long:"password" env:"PASSWORD" description:"SMTP password"`
	TLS                bool          `long:"tls" env:"TLS" description:"enable TLS"`
	InsecureSkipVerify bool          `long:"insecure_skip_verify" env:"INSECURE_SKIP_VERIFY" description:"skip certificate verification"`
	LoginAuth          bool          `long:"login_auth" env:"LOGIN_AUTH" description:"enable LOGIN auth instead of PLAIN"`
	StartTLS           bool          `long:"starttls" env:"STARTTLS" description:"enable StartTLS"`
	TimeOut            time.Duration `long:"timeout" env:"TIMEOUT" default:"10s" description:"SMTP TCP connection timeout"`
}

// NotifyGroup defines options for notification
type NotifyGroup struct {
	Type      []string `long:"type" env:"TYPE" description:"[deprecated, use user and admin types instead] types of notifications" choice:"none" choice:"telegram" choice:"email" choice:"slack" default:"none" env-delim:","` //nolint
	Users     []string `long:"users" env:"USERS" description:"types of user notifications" choice:"none" choice:"email" choice:"telegram" default:"none" env-delim:","`                                                        //nolint
	Admins    []string `long:"admins" env:"ADMINS" description:"types of admin notifications" choice:"none" choice:"telegram" choice:"email" choice:"slack" choice:"webhook" default:"none" env-delim:","`                     //nolint
	QueueSize int      `long:"queue" env:"QUEUE" description:"size of notification queue" default:"100"`
	Telegram  struct {
		Channel string        `long:"chan" env:"CHAN" description:"the ID of telegram channel for admin notifications"`
		API     string        `long:"api" env:"API" default:"https://api.telegram.org/bot" description:"[deprecated, use --telegram.api-url] telegram api prefix"`
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
		Channel string `long:"chan" env:"CHAN" description:"slack channel for admin notifications"`
	} `group:"slack" namespace:"slack" env-namespace:"SLACK"`
	Webhook struct {
		URL      string        `long:"url" env:"URL" description:"webhook URL for admin notifications"`
		Template string        `long:"template" env:"TEMPLATE" description:"webhook payload template (Go text/template); falls back to {\"text\": {{.Text | escapeJSONString}}} when empty"`
		Headers  []string      `long:"headers" description:"webhook header in format Header:Value, repeat the flag for each header; the env var takes Header1:Value1,Header2:Value2 [$NOTIFY_WEBHOOK_HEADERS]"` // env NOTIFY_WEBHOOK_HEADERS split in code below to allow , inside ""
		Timeout  time.Duration `long:"timeout" env:"TIMEOUT" description:"webhook timeout" default:"5s"`
	} `group:"webhook" namespace:"webhook" env-namespace:"WEBHOOK"`
}

// SSLGroup defines options group for server ssl params
type SSLGroup struct {
	Type         string `long:"type" env:"TYPE" description:"ssl (auto) support" choice:"none" choice:"static" choice:"auto" default:"none"` //nolint
	Port         int    `long:"port" env:"PORT" description:"port number for https server" default:"8443"`
	Cert         string `long:"cert" env:"CERT" description:"path to the cert.pem file"`
	Key          string `long:"key" env:"KEY" description:"path to the key.pem file"`
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

// client makes the json-rpc client of a remote store
func (g RPCGroup) client() jrpc.Client {
	return jrpc.Client{API: g.API, Client: http.Client{Timeout: g.TimeOut}, AuthUser: g.AuthUser, AuthPasswd: g.AuthPassword}
}

// AdminRPCGroup defines options for remote admin store
type AdminRPCGroup struct {
	RPCGroup
	SecretPerSite bool `long:"secret_per_site" env:"SECRET_PER_SITE" description:"enable JWT secret retrieval per aud, which is site_id in this case"`
}

// serverApp holds all active objects
type serverApp struct {
	*ServerCommand
	restSrv       *api.Rest
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

// clearSecretEnv removes secrets from the environment once the flags are parsed. The webhook headers,
// which may carry an Authorization value, are read from the environment here, because the flag parser
// can't split them, and kept in the options
func (s *ServerCommand) clearSecretEnv() {
	if len(s.Notify.Webhook.Headers) == 0 {
		s.Notify.Webhook.Headers = splitAtCommas(os.Getenv("NOTIFY_WEBHOOK_HEADERS")) // env value may have comma inside ""
	}
	resetEnv(
		"SECRET",
		"AUTH_APPLE_KID",
		"AUTH_GOOGLE_CSEC",
		"AUTH_GITHUB_CSEC",
		"AUTH_FACEBOOK_CSEC",
		"AUTH_MICROSOFT_CSEC",
		"AUTH_TWITTER_CSEC",
		"AUTH_YANDEX_CSEC",
		"AUTH_PATREON_CSEC",
		"AUTH_DISCORD_CSEC",
		"AUTH_CUSTOM_CSEC",
		"AUTH_EMAIL_PASSWD",
		"TELEGRAM_TOKEN",
		"NOTIFY_TELEGRAM_TOKEN",
		"NOTIFY_SLACK_TOKEN",
		"NOTIFY_WEBHOOK_HEADERS",
		"SMTP_PASSWORD",
		"ADMIN_PASSWD",
		"STORE_RPC_AUTH_PASSWD",
		"IMAGE_RPC_AUTH_PASSWD",
		"ADMIN_RPC_AUTH_PASSWD",
	)
}

// Execute is the entry point for "server" command, called by flag parser
func (s *ServerCommand) Execute(_ []string) error {
	log.Printf("[INFO] start server on port %s:%d", s.Address, s.Port)
	s.clearSecretEnv()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) // graceful termination
	defer cancel()

	app, err := s.newServerApp(ctx)
	if err != nil {
		// returned to main, which logs it; a PANIC level here would dump the stack and exit before the return
		return fmt.Errorf("failed to setup application: %w", err)
	}
	if err = app.run(ctx); err != nil {
		log.Printf("[ERROR] remark terminated with error %+v", err)
		return err
	}
	log.Printf("[INFO] remark terminated")
	return nil
}

// HandleDeprecatedFlags sets new flags from deprecated ones and returns their list.
// An entry has Old and Version set and New when there is a substitute. A flag set both the old and
// the new way to different values comes back with Collision set instead: the new one is used.
func (s *ServerCommand) HandleDeprecatedFlags() (result []DeprecatedFlag) {
	result = moveDeprecated(result, s.Auth.Email.Host, &s.SMTP.Host, "", "auth.email.host", "smtp.host", "1.5")
	result = moveDeprecated(result, s.Auth.Email.Port, &s.SMTP.Port, 0, "auth.email.port", "smtp.port", "1.5")
	result = moveDeprecated(result, s.Auth.Email.TLS, &s.SMTP.TLS, false, "auth.email.tls", "smtp.tls", "1.5")
	result = moveDeprecated(result, s.Auth.Email.SMTPUserName, &s.SMTP.Username, "", "auth.email.user", "smtp.username", "1.5")
	result = moveDeprecated(result, s.Auth.Email.SMTPPassword, &s.SMTP.Password, "", "auth.email.passwd", "smtp.password", "1.5")
	result = moveDeprecated(result, s.Auth.Email.TimeOut, &s.SMTP.TimeOut, 10*time.Second, "auth.email.timeout", "smtp.timeout", "1.5")
	if s.Auth.Email.MsgTemplate != "email_confirmation_login.html.tmpl" {
		result = append(result, DeprecatedFlag{Old: "auth.email.template", Version: "1.5"})
	}
	result = moveDeprecated(result, s.LegacyImageProxy, &s.ImageProxy.HTTP2HTTPS, false, "img-proxy", "image-proxy.http2https", "1.5")
	if !contains("none", s.Notify.Type) {
		// if new notify param(s) are used, safe to ignore the old one
		collision := !contains("none", s.Notify.Users) || !contains("none", s.Notify.Admins)
		if !collision {
			s.handleDeprecatedNotifications()
		}
		result = append(result, deprecated("notify.type", "notify.(users|admins)", "1.9", collision))
	}
	if s.Notify.Email.AdminNotifications && !contains("email", s.Notify.Admins) {
		s.Notify.Admins = append(s.Notify.Admins, "email")
		result = append(result, DeprecatedFlag{Old: "notify.email.notify_admin", New: "notify.admins=email", Version: "1.9"})
	}
	result = moveDeprecated(result, s.Notify.Telegram.Token, &s.Telegram.Token, "", "notify.telegram.token", "telegram.token", "1.9")
	result = moveDeprecated(result, s.Notify.Telegram.Timeout, &s.Telegram.Timeout, 5*time.Second, "notify.telegram.timeout", "telegram.timeout", "1.9")
	if s.Notify.Telegram.API != "https://api.telegram.org/bot" {
		base, ok := telegramAPIBaseFromDeprecated(s.Notify.Telegram.API)
		switch {
		case !ok && s.Telegram.APIURL == "":
			result = append(result, DeprecatedFlag{Old: "notify.telegram.api", Version: "1.9"})
		case ok && s.Telegram.APIURL == "":
			s.Telegram.APIURL = base
			result = append(result, DeprecatedFlag{Old: "notify.telegram.api", New: "telegram.api-url", Version: "1.9"})
		case ok && base != "" && base != s.Telegram.APIURL:
			result = append(result, deprecated("notify.telegram.api", "telegram.api-url", "1.9", true))
		}
	}
	return result
}

// moveDeprecated carries the value of a deprecated flag over to its replacement when only the old one
// is set, and reports a collision when both are set to different values. unset is the flags' default.
func moveDeprecated[T comparable](res []DeprecatedFlag, old T, cur *T, unset T, oldName, newName, version string) []DeprecatedFlag {
	switch {
	case old == unset:
		return res
	case *cur == unset:
		*cur = old
		return append(res, deprecated(oldName, newName, version, false))
	case *cur != old:
		return append(res, deprecated(oldName, newName, version, true))
	}
	return res
}

// deprecated makes the entry for a deprecated flag; a collision carries no version
func deprecated(oldName, newName, version string, collision bool) DeprecatedFlag {
	if collision {
		return DeprecatedFlag{Old: oldName, New: newName, Collision: true}
	}
	return DeprecatedFlag{Old: oldName, New: newName, Version: version}
}

func telegramAPIBaseFromDeprecated(prefix string) (string, bool) {
	trimmed := strings.TrimSuffix(prefix, "/")
	if !strings.HasSuffix(trimmed, "/bot") {
		return "", false
	}
	return strings.TrimSuffix(trimmed, "/bot"), true
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
	return slices.Contains(a, s)
}

var reservedCustomProviderNames = []string{
	"email", "anonymous", "google", "github", "facebook", "yandex", "twitter",
	"microsoft", "patreon", "discord", "telegram", "dev", "apple",
}

var validCustomProviderName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func customProviderSourceID(data provider.UserData, cfg CustomAuthGroup) string {
	sourceID := cmp.Or(data.Value(cfg.IDField), data.Value(cfg.EmailField), data.Value(cfg.NameField), data.Value(cfg.PictureField))
	if sourceID == "" {
		log.Printf("[WARN] custom oauth provider returned no stable user identifier fields, falling back to hashed payload")
		payload, err := json.Marshal(data)
		if err != nil {
			log.Printf("[WARN] failed to serialize custom oauth user data for ID fallback: %v", err)
		} else {
			sourceID = string(payload)
		}
	}
	return sourceID
}

func (c CustomAuthGroup) isConfigured() bool {
	return c.Name != "" || c.CID != "" || c.CSEC != "" || c.AuthURL != "" || c.TokenURL != "" || c.InfoURL != "" ||
		len(c.Scopes) > 0 || c.IDField != "sub" || c.NameField != "name" || c.PictureField != "picture" || c.EmailField != "email"
}

func (c CustomAuthGroup) missingRequired() []string {
	missing := []string{}
	for _, f := range []struct{ value, env string }{
		{c.Name, "AUTH_CUSTOM_NAME"}, {c.CID, "AUTH_CUSTOM_CID"}, {c.CSEC, "AUTH_CUSTOM_CSEC"},
		{c.AuthURL, "AUTH_CUSTOM_AUTH_URL"}, {c.TokenURL, "AUTH_CUSTOM_TOKEN_URL"}, {c.InfoURL, "AUTH_CUSTOM_INFO_URL"},
	} {
		if f.value == "" {
			missing = append(missing, f.env)
		}
	}
	return missing
}

// newServerApp prepares application and return it with all active parts
// doesn't start anything
func (s *ServerCommand) newServerApp(ctx context.Context) (_ *serverApp, err error) {
	if err = makeDirs(s.BackupLocation); err != nil {
		return nil, fmt.Errorf("failed to create backup store: %w", err)
	}

	if !strings.HasPrefix(s.RemarkURL, "http://") && !strings.HasPrefix(s.RemarkURL, "https://") {
		return nil, fmt.Errorf("invalid remark42 url %s", s.RemarkURL)
	}
	log.Printf("[INFO] root url=%s", s.RemarkURL)

	// parse trusted proxies up front so a bad CIDR fails before any resource is allocated
	trustedProxies, err := api.ParseTrustedProxies(s.TrustedProxies)
	if err != nil {
		return nil, fmt.Errorf("invalid --trusted-proxy: %w", err)
	}
	switch {
	case len(trustedProxies) == 0:
		log.Printf("[WARN] --trusted-proxy not set: forwarding headers are trusted from any client and can be spoofed to bypass rate limiting / vote dedup; set it behind a reverse proxy (see docs)")
	case api.TrustsAnyPeer(trustedProxies):
		log.Printf("[WARN] --trusted-proxy has a catch-all (0.0.0.0/0 or ::/0): forwarding headers are trusted from any client, re-opening the spoofing bypass; scope it to your proxy network")
	}

	// read the restricted words file up front so a wrong path fails before any resource is allocated
	var restrictedWords service.RestrictedWordsLister = service.StaticRestrictedWordsLister{Words: s.RestrictedWords}
	if s.RestrictedWordsFile != "" {
		fileWords := &service.FileRestrictedWordsLister{Path: s.RestrictedWordsFile, Words: s.RestrictedWords}
		if err = fileWords.Load(); err != nil {
			return nil, fmt.Errorf("invalid --restricted-words-file: %w", err)
		}
		restrictedWords = fileWords
	}

	storeEngine, err := s.makeDataStore()
	if err != nil {
		return nil, fmt.Errorf("failed to make data store engine: %w", err)
	}

	adminStore, err := s.makeAdminStore()
	if err != nil {
		return nil, fmt.Errorf("failed to make admin store: %w", err)
	}

	imageService, err := s.makePicturesStore()
	if err != nil {
		return nil, fmt.Errorf("failed to make pictures store: %w", err)
	}
	log.Printf("[DEBUG] image service for url=%s, EditDuration=%v", imageService.ImageAPI, imageService.EditDuration)

	dataService := &service.DataStore{
		Engine:                 storeEngine,
		EditDuration:           s.EditDuration,
		AdminEdits:             s.AdminEdit,
		AdminStore:             adminStore,
		MinCommentSize:         s.MinCommentSize,
		MaxCommentSize:         s.MaxCommentSize,
		MaxVotes:               s.MaxVotes,
		PositiveScore:          s.PositiveScore,
		ImageService:           imageService,
		TitleExtractor:         service.NewTitleExtractor(http.Client{Timeout: time.Second * 5, Transport: safehttp.Transport()}, s.getAllowedDomains()),
		RestrictedWordsMatcher: service.NewRestrictedWordsMatcher(restrictedWords),
	}
	dataService.RestrictSameIPVotes.Enabled = s.RestrictVoteIP
	dataService.RestrictSameIPVotes.Duration = s.DurationVoteIP

	// from here on a failed setup closes what it has opened
	var refreshCache *authRefreshCache
	defer func() {
		if err == nil {
			return
		}
		_ = dataService.Close()
		if refreshCache != nil {
			_ = refreshCache.Close()
		}
	}()

	loadingCache, err := s.makeCache()
	if err != nil {
		return nil, fmt.Errorf("failed to make cache: %w", err)
	}

	avatarStore, err := s.makeAvatarStore()
	if err != nil {
		return nil, fmt.Errorf("failed to make avatar store: %w", err)
	}
	refreshCache = newAuthRefreshCache()
	authenticator := s.getAuthenticator(dataService, avatarStore, adminStore, refreshCache)

	// telegram auth requires the TelegramAPI listener started below
	telegramAuth, err := s.makeTelegramAuth(authenticator)
	if err != nil {
		return nil, err
	}
	telegramService, err := s.startTelegramAuthAndNotify(ctx, telegramAuth)
	if err != nil {
		return nil, err
	}

	providersCount, err := s.addAuthProviders(authenticator)
	if err != nil {
		return nil, fmt.Errorf("failed to make authenticator: %w", err)
	}
	if providersCount == 0 {
		log.Printf("[WARN] no auth providers defined")
	}

	exporter := &migrator.Native{DataStore: dataService}
	disqusImporter := &migrator.Disqus{DataStore: dataService}
	if avatarProxy := authenticator.AvatarProxy(); avatarProxy != nil {
		disqusImporter.Avatars = avatarProxy // imported avatars are stored and served the same way as the ones from login
	}

	migr := &api.Migrator{
		Cache:             loadingCache,
		NativeImporter:    &migrator.Native{DataStore: dataService},
		DisqusImporter:    disqusImporter,
		WordPressImporter: &migrator.WordPress{DataStore: dataService, DisableFancyTextFormatting: s.DisableFancyTextFormatting},
		CommentoImporter:  &migrator.Commento{DataStore: dataService, DisableFancyTextFormatting: s.DisableFancyTextFormatting},
		NativeExporter:    &migrator.Native{DataStore: dataService},
		URLMapperMaker:    migrator.NewURLMapper,
	}

	notifyDestinations, err := s.makeNotifyDestinations(authenticator)
	if err != nil {
		// each error leaves out one destination or the admin part of one; the rest are set up and work.
		// errors.Join puts each error on its own line, kept on this one so every part carries the level
		log.Printf("[ERROR] notifications are not fully set up, %s", strings.ReplaceAll(err.Error(), "\n", "; "))
	}

	notifyService := s.makeNotifyService(dataService, notifyDestinations, telegramService)

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
		return nil, fmt.Errorf("failed to make config of ssl server params: %w", err)
	}

	srv := &api.Rest{
		Version:                    s.Revision,
		DataService:                dataService,
		WebRoot:                    s.WebRoot,
		WebFS:                      webFS,
		RemarkURL:                  s.RemarkURL,
		DemoSiteID:                 s.demoSiteID(),
		ImageProxy:                 imgProxy,
		CommentFormatter:           commentFormatter,
		Migrator:                   migr,
		ReadOnlyAge:                s.ReadOnlyAge,
		SharedSecret:               s.SharedSecret,
		TrustedProxies:             trustedProxies,
		Authenticator:              authenticator,
		Cache:                      loadingCache,
		NotifyService:              notifyService,
		SSLConfig:                  sslConfig,
		UpdateLimiter:              s.UpdateLimit,
		ImageService:               imageService,
		EmailNotifications:         contains("email", s.Notify.Users),
		TelegramNotifications:      contains("telegram", s.Notify.Users) && telegramService != nil,
		EmojiEnabled:               s.EnableEmoji,
		AnonVote:                   s.AnonymousVote && s.RestrictVoteIP,
		SimpleView:                 s.SimpleView,
		ProxyCORS:                  s.ProxyCORS,
		AllowedAncestors:           s.AllowedHosts,
		SendJWTHeader:              s.Auth.SendJWTHeader,
		SubscribersOnly:            s.SubscribersOnly,
		DisableSignature:           s.DisableSignature,
		DisableFancyTextFormatting: s.DisableFancyTextFormatting,
		ExternalImageProxy:         s.ImageProxy.CacheExternal,
	}

	// assigned only when there is one: telegramService is an interface field, so a nil
	// *notify.Telegram put in it is a non-nil interface, and the handler's own `== nil` guard then
	// lets a dereference through as a 500. Telegram auth without telegram notifications is an
	// ordinary reachable configuration
	if telegramService != nil && contains("telegram", s.Notify.Users) {
		srv.TelegramService = telegramService
	}

	srv.ScoreThresholds.Low, srv.ScoreThresholds.Critical = s.LowScore, s.CriticalScore

	var devAuth *provider.DevAuthServer
	if s.Auth.Dev {
		da, errDevAuth := authenticator.DevAuth()
		if errDevAuth != nil {
			return nil, fmt.Errorf("can't make dev oauth2 server: %w", errDevAuth)
		}
		devAuth = da
	}

	return &serverApp{
		ServerCommand:    s,
		restSrv:          srv,
		exporter:         exporter,
		devAuth:          devAuth,
		dataService:      dataService,
		avatarStore:      avatarStore,
		notifyService:    notifyService,
		imageService:     imageService,
		authenticator:    authenticator,
		terminated:       make(chan struct{}),
		authRefreshCache: refreshCache,
	}, nil
}

// Extract domains from s.AllowedHosts and second level domain from s.RemarkURL.
// It can be and IP like http://127.0.0.1 in which case we need to use whole IP as domain
// Beware, if s.RemarkURL is in third-level domain like https://example.co.uk, co.uk will be returned.
func (s *ServerCommand) getAllowedDomains() []string {
	rawDomains := s.AllowedHosts
	rawDomains = append(rawDomains, s.RemarkURL)
	allowedDomains := []string{}
	for _, rawURL := range rawDomains {
		// case of 'self' AllowedHosts, which is not a valid rawURL name
		if rawURL == "self" || rawURL == "'self'" || rawURL == "\"self\"" {
			continue
		}
		// AllowedHosts usually don't have https:// prefix, so we're adding it just to make parsing below work the same way as for RemarkURL
		if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
			rawURL = "https://" + rawURL
		}
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			log.Printf("[WARN] failed to parse URL %s for TitleExtract whitelist: %v", rawURL, err)
			continue
		}
		domain := parsedURL.Hostname()

		if domain == "" || // don't add empty domain as it will allow everything to be extracted
			(len(strings.Split(domain, ".")) < 2 && // don't allow single-word domains like "com"
				domain != "localhost") { // localhost is an exceptional single-word domain which is allowed
			continue
		}

		// only for RemarkURL if domain is not IP and has more than two levels, extract second level domain.
		// for AllowedHosts we don't do this as they are exact list of domains which can host comments, but
		// remarkURL might be on a subdomain and we must allow parent domain to be used for TitleExtract.
		if rawURL == s.RemarkURL && net.ParseIP(domain) == nil && len(strings.Split(domain, ".")) > 2 {
			domain = strings.Join(strings.Split(domain, ".")[len(strings.Split(domain, "."))-2:], ".")
		}

		allowedDomains = append(allowedDomains, domain)
	}
	return allowedDomains
}

// getAllowedRedirectHosts normalises s.AllowedHosts into the form that
// go-pkgz/auth's redirect validator expects. Strips http(s) schemes and
// paths; preserves explicit ports (the validator matches both host-only
// and host:port, so an entry without a port accepts any port while an
// entry with a port restricts to that port). Skips CSP sentinels
// ('self' / "self") and wildcard entries (*, *.example.com) that are
// valid CSP source expressions but not valid hostnames.
func (s *ServerCommand) getAllowedRedirectHosts() []string {
	out := make([]string, 0, len(s.AllowedHosts))
	for _, raw := range s.AllowedHosts {
		raw = strings.TrimSpace(raw)
		if raw == "" || raw == "self" || raw == "'self'" || raw == `"self"` {
			continue
		}
		if strings.ContainsRune(raw, '*') { // CSP wildcard, not a host
			continue
		}
		// add scheme so url.Parse populates Hostname()/Host consistently for bare hosts
		toParse := raw
		if !strings.HasPrefix(toParse, "http://") && !strings.HasPrefix(toParse, "https://") {
			toParse = "https://" + toParse
		}
		u, err := url.Parse(toParse)
		if err != nil || u.Hostname() == "" {
			log.Printf("[WARN] skipping invalid AllowedHosts entry %q for redirect allowlist: %v", raw, err)
			continue
		}
		if u.Port() != "" {
			out = append(out, u.Host) // preserve explicit host:port so allowlist is port-specific
			continue
		}
		out = append(out, u.Hostname())
	}
	return out
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
			return nil, fmt.Errorf("failed to create bolt store: %w", err)
		}
		sites := []engine.BoltSite{}
		for _, site := range s.Sites {
			sites = append(sites, engine.BoltSite{SiteID: site, FileName: fmt.Sprintf("%s/%s.db", s.Store.Bolt.Path, site)})
		}
		result, err = engine.NewBoltDB(bolt.Options{Timeout: s.Store.Bolt.Timeout}, sites...)
	case "rpc":
		return &engine.RPC{Client: s.Store.RPC.client()}, nil
	default:
		return nil, fmt.Errorf("unsupported store type %s", s.Store.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("can't initialize data store: %w", err)
	}
	return result, nil
}

func (s *ServerCommand) makeAvatarStore() (avatar.Store, error) {
	log.Printf("[INFO] make avatar store, type=%s", s.Avatar.Type)
	return newAvatarStore(s.Avatar)
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
			return nil, fmt.Errorf("failed to create pictures store: %w", err)
		}
		return image.NewService(&image.FileSystem{
			Location:   s.Image.FS.Path,
			Staging:    s.Image.FS.Staging,
			Partitions: s.Image.FS.Partitions,
		}, imageServiceParams), nil
	case "rpc":
		return image.NewService(&image.RPC{Client: s.Image.RPC.client()}, imageServiceParams), nil
	}
	return nil, fmt.Errorf("unsupported pictures store type %s", s.Image.Type)
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
		return &admin.RPC{Client: s.Admin.RPC.client()}, nil
	default:
		return nil, fmt.Errorf("unsupported admin store type %s", s.Admin.Type)
	}
}

func (s *ServerCommand) makeCache() (api.LoadingCache, error) {
	log.Printf("[INFO] make cache, type=%s", s.Cache.Type)
	o := cache.NewOpts[[]byte]()
	switch s.Cache.Type {
	case "redis_pub_sub":
		redisPubSub, err := eventbus.NewRedisPubSub(s.Cache.RedisAddr, "remark42-cache")
		if err != nil {
			return nil, fmt.Errorf("cache backend initialization, redis PubSub initialisation: %w", err)
		}
		backend, err := cache.NewLruCache(o.MaxCacheSize(s.Cache.Max.Size), o.MaxValSize(s.Cache.Max.Value),
			o.MaxKeys(s.Cache.Max.Items), o.EventBus(redisPubSub))
		if err != nil {
			return nil, fmt.Errorf("cache backend initialization: %w", err)
		}
		return cache.NewScache[[]byte](backend), nil
	case "mem":
		backend, err := cache.NewLruCache(o.MaxCacheSize(s.Cache.Max.Size), o.MaxValSize(s.Cache.Max.Value),
			o.MaxKeys(s.Cache.Max.Items))
		if err != nil {
			return nil, fmt.Errorf("cache backend initialization: %w", err)
		}
		return cache.NewScache[[]byte](backend), nil
	case "none":
		return cache.NewScache[[]byte](&cache.Nop[[]byte]{}), nil
	}
	return nil, fmt.Errorf("unsupported cache type %s", s.Cache.Type)
}

// addAuthProviders registers the configured sign-in providers and returns how many the authenticator
// has in total, the Telegram one registered earlier by makeTelegramAuth included. The count is taken
// from the authenticator, so email and anonymous sign-in, which need no OAuth2 provider, count as well.
func (s *ServerCommand) addAuthProviders(authenticator *auth.Service) (int, error) {
	if s.Auth.Apple.CID != "" && s.Auth.Apple.TID != "" && s.Auth.Apple.KID != "" {
		err := authenticator.AddAppleProvider(
			provider.AppleConfig{
				ClientID: s.Auth.Apple.CID,
				TeamID:   s.Auth.Apple.TID,
				KeyID:    s.Auth.Apple.KID,
			},
			provider.LoadApplePrivateKeyFromFile(s.Auth.Apple.PrivateKeyFilePath),
		)
		if err != nil {
			return 0, err
		}
	}
	// registration order is the order of providers in /api/v1/config
	for _, p := range []struct{ name, cid, csec string }{
		{"google", s.Auth.Google.CID, s.Auth.Google.CSEC},
		{"github", s.Auth.Github.CID, s.Auth.Github.CSEC},
		{"facebook", s.Auth.Facebook.CID, s.Auth.Facebook.CSEC},
		{"microsoft", s.Auth.Microsoft.CID, s.Auth.Microsoft.CSEC},
		{"yandex", s.Auth.Yandex.CID, s.Auth.Yandex.CSEC},
		{"twitter", s.Auth.Twitter.CID, s.Auth.Twitter.CSEC},
		{"patreon", s.Auth.Patreon.CID, s.Auth.Patreon.CSEC},
		{"discord", s.Auth.Discord.CID, s.Auth.Discord.CSEC},
	} {
		if p.cid == "" || p.csec == "" {
			continue
		}
		if p.name == "microsoft" {
			authenticator.AddMicrosoftProvider(p.cid, p.csec, s.Auth.Microsoft.Tenant)
		} else {
			authenticator.AddProvider(p.name, p.cid, p.csec)
		}
	}

	if s.Auth.Custom.isConfigured() {
		missing := s.Auth.Custom.missingRequired()
		if len(missing) > 0 {
			return 0, fmt.Errorf("custom oauth provider configuration is incomplete, missing: %s", strings.Join(missing, ", "))
		}

		customName := strings.ToLower(strings.TrimSpace(s.Auth.Custom.Name))
		if !validCustomProviderName.MatchString(customName) {
			return 0, fmt.Errorf("custom oauth provider name %q is invalid, expected pattern %q", customName, validCustomProviderName.String())
		}
		if slices.Contains(reservedCustomProviderNames, customName) {
			return 0, fmt.Errorf("custom oauth provider name %q is reserved", customName)
		}

		authenticator.AddCustomProvider(customName, auth.Client{Cid: s.Auth.Custom.CID, Csecret: s.Auth.Custom.CSEC}, provider.CustomHandlerOpt{
			Endpoint: oauth2.Endpoint{
				AuthURL:  s.Auth.Custom.AuthURL,
				TokenURL: s.Auth.Custom.TokenURL,
			},
			InfoURL: s.Auth.Custom.InfoURL,
			Scopes:  s.Auth.Custom.Scopes,
			MapUserFn: func(data provider.UserData, _ []byte) token.User {
				sourceID := customProviderSourceID(data, s.Auth.Custom)
				hashID := token.HashID(sha1.New(), sourceID) //nolint:gosec // stable provider user id hash
				user := token.User{
					ID:      customName + "_" + hashID,
					Name:    data.Value(s.Auth.Custom.NameField),
					Picture: data.Value(s.Auth.Custom.PictureField),
					Email:   data.Value(s.Auth.Custom.EmailField),
				}
				if user.Name == "" {
					user.Name = "noname_" + hashID[:4]
				}
				return user
			},
		})
	}

	if s.Auth.Dev {
		log.Print("[INFO] dev access enabled")
		u, errURL := url.Parse(s.RemarkURL)
		if errURL != nil {
			return 0, fmt.Errorf("can't parse Remark42 URL: %w", errURL)
		}
		authenticator.AddDevProvider(u.Hostname(), 8084)
	}

	if s.Auth.Email.Enable {
		params := sender.EmailParams{
			Host:               s.SMTP.Host,
			Port:               s.SMTP.Port,
			HELOHost:           s.SMTP.HELOHost,
			SMTPUserName:       s.SMTP.Username,
			SMTPPassword:       s.SMTP.Password,
			TimeOut:            s.SMTP.TimeOut,
			StartTLS:           s.SMTP.StartTLS,
			LoginAuth:          s.SMTP.LoginAuth,
			TLS:                s.SMTP.TLS,
			InsecureSkipVerify: s.SMTP.InsecureSkipVerify,
			Charset:            "UTF-8",
			From:               s.Auth.Email.From,
			Subject:            s.Auth.Email.Subject,
			ContentType:        s.Auth.Email.ContentType,
		}
		sndr := newEmailSender(params, log.Default())
		tmpl, err := templates.Read(s.Auth.Email.MsgTemplate)
		if err != nil {
			return 0, err
		}
		authenticator.AddVerifProvider("email", string(tmpl), sndr)
	}

	if s.Auth.Anonymous {
		log.Print("[INFO] anonymous access enabled")
		authenticator.AddDirectProviderWithUserIDFunc("anonymous", provider.CredCheckerFunc(checkAnonymousName),
			anonymousUserID(s.SharedSecret))
	}

	return len(authenticator.Providers()), nil
}

// impersonationLogLine makes the log line for a user blocked for taking a restricted name. It names
// the user's ID and name only: the user in the claims also carries the notification email, which
// must stay out of the log.
func impersonationLogLine(u *token.User) string {
	return fmt.Sprintf("[INFO] blocked %s (%s), attempt to impersonate (restricted names)", u.ID, store.MaskForLog(u.Name))
}

// anonymousUserID makes the user ID of an anonymous login from the name and the client address, so
// users with the same name coming from different addresses get different IDs. The ID is public and so
// is the name, so the value is an HMAC keyed by the shared secret: a plain hash of the two would give
// the address away to anyone trying every IPv4 address. The separator is a character a name can't
// contain, so a name ending in digits and a shorter address can't add up to another user's input.
// The result is 40 hex characters, which the auth library uses as is.
func anonymousUserID(secret string) func(user string, r *http.Request) string {
	return func(user string, r *http.Request) string {
		return store.HashValue(user+"|"+r.RemoteAddr, secret)
	}
}

var isValidAnonName = regexp.MustCompile(`^[\p{L}\d_ ]+$`).MatchString

// checkAnonymousName accepts a name for anonymous login: 3 to 64 letters, digits, underscores and spaces,
// with no space at either end. A rejected name is logged through the mask, as an email address can be typed here
func checkAnonymousName(user, _ string) (ok bool, err error) {
	// don't allow anon with space prefix or suffix
	if strings.HasPrefix(user, " ") || strings.HasSuffix(user, " ") {
		log.Printf("[WARN] name %q has space as a suffix or prefix", store.MaskForLog(user))
		return false, nil
	}

	user = strings.TrimSpace(user)
	if len(user) < 3 {
		log.Printf("[WARN] name %q is too short, should be at least 3 characters", store.MaskForLog(user))
		return false, nil
	}
	if len(user) > 64 {
		log.Printf("[WARN] name %q is too long, should be up to 64 characters", store.MaskForLog(user))
		return false, nil
	}

	if !isValidAnonName(user) {
		log.Printf("[WARN] name %q should have letters, digits, underscores and spaces only", store.MaskForLog(user))
		return false, nil
	}
	return true, nil
}

// newEmailSender makes the sender of sign-in emails. Its debug line names the recipient, so the address is masked
func newEmailSender(params sender.EmailParams, l log.L) *sender.Email {
	return sender.NewEmailClient(params, rest.MaskedLog{L: l})
}

// creates and registers telegram auth, which we need separately from other auth providers
func (s *ServerCommand) makeTelegramAuth(authenticator *auth.Service) (*provider.TelegramHandler, error) {
	if !s.Auth.Telegram {
		return nil, nil
	}

	client := &http.Client{Timeout: s.Telegram.Timeout}

	// the public api unless an operator points it elsewhere. A bad base url stops the server because
	// every request carries the bot token in its path; a base that resolves somewhere unintended
	// ships the token there, and an operator who set the option meant it to be used
	tgAPI := provider.NewTelegramAPI(s.Telegram.Token, client)
	if s.Telegram.APIURL != "" {
		withBase, err := provider.NewTelegramAPIWithBaseURL(s.Telegram.Token, client, s.Telegram.APIURL)
		if err != nil {
			return nil, fmt.Errorf("failed to make telegram auth: %w", err)
		}
		tgAPI = withBase
	}

	telegram := &provider.TelegramHandler{
		ProviderName: "telegram",
		ErrorMsg:     "Authentication request was not found or expired.",
		SuccessMsg:   "✅ You have successfully authenticated, check the web!",
		Telegram:     tgAPI,
		L:            log.Default(),
		TokenService: authenticator.TokenService(),
		AvatarSaver:  authenticator.AvatarProxy(),
	}
	authenticator.AddCustomHandler(telegram)
	return telegram, nil
}

func (s *ServerCommand) makeNotifyService(dataStore *service.DataStore, destinations []notify.Destination, telegram *notify.Telegram) *notify.Service {
	// it's possible that telegram notification service was created for auth but should not be used for notifications
	if telegram != nil && (contains("telegram", s.Notify.Users) || contains("telegram", s.Notify.Admins)) {
		destinations = append(destinations, telegram)
	}

	if len(destinations) > 0 {
		log.Printf("[INFO] make notify, for users: %s, for admins: %s", s.Notify.Users, s.Notify.Admins)
		return notify.NewService(dataStore, s.Notify.QueueSize, destinations...)
	}
	return notify.NopService
}

// constructs list of notify destinations except for telegram. A destination that cannot be built is
// left out and its error returned, joined with the others, while the remaining destinations are still built
func (s *ServerCommand) makeNotifyDestinations(authenticator *auth.Service) ([]notify.Destination, error) {
	destinations := make([]notify.Destination, 0)
	var errs []error

	if contains("webhook", s.Notify.Admins) {
		whParams := notify.WebhookParams{
			URL:      s.Notify.Webhook.URL,
			Template: s.Notify.Webhook.Template,
			Headers:  s.Notify.Webhook.Headers,
			Timeout:  s.Notify.Webhook.Timeout,
		}
		if webhook, err := notify.NewWebhook(whParams); err != nil {
			errs = append(errs, fmt.Errorf("failed to create webhook notification destination: %w", err))
		} else {
			destinations = append(destinations, webhook)
		}
	}

	if contains("slack", s.Notify.Admins) {
		slack := notify.NewSlack(s.Notify.Slack.Token, s.Notify.Slack.Channel)
		destinations = append(destinations, slack)
	}

	// with logic below admin notifications enable notifications for users on the backend even if they
	// are not enabled explicitly, however they won't be visible to the users in the frontend
	// because api.Rest.EmailNotifications would be set to false.
	if contains("email", s.Notify.Users) || contains("email", s.Notify.Admins) {
		emailParams := notify.EmailParams{
			From:                s.Notify.Email.From,
			VerificationSubject: s.Notify.Email.VerificationSubject,
			UnsubscribeURL:      s.RemarkURL + "/email/unsubscribe.html",
			// TODO: uncomment after #560 frontend part is ready and URL is known
			// subscribeURL:        s.RemarkURL + "/subscribe.html?token=",
			TokenGenFn: func(userID, email, site string) (string, error) {
				claims := token.Claims{
					Handshake: &token.Handshake{ID: userID + "::" + email},
					Audience:  jwt.ClaimStrings{site},
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(100 * 365 * 24 * time.Hour)),
					NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
					Issuer:    "remark42",
				}
				tkn, err := authenticator.TokenService().Token(claims)
				if err != nil {
					return "", fmt.Errorf("failed to make unsubscription token: %w", err)
				}
				return tkn, nil
			},
		}
		if contains("email", s.Notify.Admins) {
			// the admin@ default of makeAdminStore is not used here: it is a guess, and notifications
			// carry comment text, so they go only to addresses the operator has set
			for _, email := range s.Admin.Shared.Email {
				if strings.TrimSpace(email) != "" {
					emailParams.AdminEmails = append(emailParams.AdminEmails, email)
				}
			}
			if len(emailParams.AdminEmails) == 0 {
				errs = append(errs, errors.New("admin email notifications are on, but --admin.shared.email (ADMIN_SHARED_EMAIL) "+
					"has no address, so no admin is notified by email"))
			}
		}
		smtpParams := ntf.SMTPParams{
			Host:               s.SMTP.Host,
			Port:               s.SMTP.Port,
			HELOHost:           s.SMTP.HELOHost,
			TLS:                s.SMTP.TLS,
			StartTLS:           s.SMTP.StartTLS,
			InsecureSkipVerify: s.SMTP.InsecureSkipVerify,
			LoginAuth:          s.SMTP.LoginAuth,
			Username:           s.SMTP.Username,
			Password:           s.SMTP.Password,
			TimeOut:            s.SMTP.TimeOut,
			ContentType:        "text/html",
			Charset:            "UTF-8",
		}
		if emailService, err := notify.NewEmail(emailParams, smtpParams); err != nil {
			errs = append(errs, fmt.Errorf("failed to create email notification destination: %w", err))
		} else {
			destinations = append(destinations, emailService)
		}
	}

	return destinations, errors.Join(errs...)
}

// constructs Telegram notify service
func (s *ServerCommand) makeTelegramNotify() (*notify.Telegram, error) {
	if contains("telegram", s.Notify.Admins) && s.Notify.Telegram.Channel == "" {
		return nil, errors.New("--notify.telegram.chan (NOTIFY_TELEGRAM_CHAN) must be set for admin notifications to work")
	}
	telegramParams := notify.TelegramParams{
		AdminChannelID:    s.Notify.Telegram.Channel,
		UserNotifications: contains("telegram", s.Notify.Users),
		Token:             s.Telegram.Token,
		Timeout:           s.Telegram.Timeout,
		// the same base as auth: the update poll runs through this service, so leaving it on the
		// public API would send the bot token there while everything else went where the operator
		// pointed it
		APIURL:     s.Telegram.APIURL,
		SuccessMsg: "✅ You have successfully subscribed for notifications, check the web!",
	}
	tg, err := notify.NewTelegram(telegramParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram notification destination: %w", err)
	}
	return tg, nil
}

func (s *ServerCommand) makeSSLConfig() (config api.SSLConfig, err error) {
	switch s.SSL.Type {
	case "none":
		config.SSLMode = api.None
	case "static":
		if s.SSL.Cert == "" {
			return config, fmt.Errorf("path to cert.pem is required")
		}
		if s.SSL.Key == "" {
			return config, fmt.Errorf("path to key.pem is required")
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

// getAuthenticator creates new authenticator service, which doesn't have any auth providers enabled
func (s *ServerCommand) getAuthenticator(ds *service.DataStore, avas avatar.Store, admns admin.Store, authRefreshCache *authRefreshCache) *auth.Service {
	return auth.NewService(auth.Opts{
		URL:            strings.TrimSuffix(s.RemarkURL, "/"),
		Issuer:         "remark42",
		TokenDuration:  s.Auth.TTL.JWT,
		CookieDuration: s.Auth.TTL.Cookie,
		SendJWTHeader:  s.Auth.SendJWTHeader,
		SameSiteCookie: s.parseSameSite(s.Auth.SameSite),
		SecureCookies:  strings.HasPrefix(s.RemarkURL, "https://"),
		// enable the `from` redirect allowlist in go-pkgz/auth v2.1.2+ — limits
		// post-auth redirects to RemarkURL's own host plus any configured
		// AllowedHosts. Prevents the OAuth open-redirect / phishing vector.
		AllowedRedirectHosts: token.AllowedHostsFunc(func() ([]string, error) {
			return s.getAllowedRedirectHosts(), nil
		}),
		SecretReader: token.SecretFunc(func(aud string) (string, error) { // get secret per site
			return admns.Key(aud)
		}),
		AudienceReader: s.tokenAudiences(),
		ClaimsUpd: token.ClaimsUpdFunc(func(c token.Claims) token.Claims { // set attributes, on new token or refresh
			if c.User == nil {
				return c
			}
			// audience is a slice but we set it to a single element, and situation when there is no audience or there are more than one is unexpected
			if len(c.Audience) != 1 {
				return c
			}
			audience := c.Audience[0]

			c.User.SetAdmin(ds.IsAdmin(audience, c.User.ID))
			c.User.SetBoolAttr("blocked", ds.IsBlocked(audience, c.User.ID))
			var err error
			c.User.Email, err = ds.GetUserEmail(audience, c.User.ID)
			if err != nil {
				log.Printf("[WARN] can't read email for %s, %v", c.User.ID, err)
			}

			// don't allow anonymous and email with admins names
			// exclude admin from impersonation detection over email, it prevents a valid admin to login with RestrictedNames
			if strings.HasPrefix(c.User.ID, "anonymous_") || (strings.HasPrefix(c.User.ID, "email_") && !c.User.IsAdmin()) {
				for _, a := range s.RestrictedNames {
					if strings.EqualFold(strings.TrimSpace(c.User.Name), a) {
						c.User.SetBoolAttr("blocked", true)
						log.Print(impersonationLogLine(c.User))
						break
					}
				}
			}

			return c
		}),
		AdminPasswd: s.AdminPasswd,
		Validator: token.ValidatorFunc(func(_ string, claims token.Claims) bool { // check on each auth call (in middleware)
			if claims.User == nil {
				return false
			}
			if claims.User.Audience == "" { // reject empty aud, made with old (pre 0.8.x) version of auth package
				return false
			}
			// a delete-me token names the user and is signed like a session, but it is handed to the
			// admin, so it must only ever work as the deletion request it is
			if claims.User.BoolAttr("delete_me") {
				return false
			}
			if !s.servesAudience(claims.Audience) {
				return false
			}
			return !claims.User.BoolAttr("blocked")
		}),
		JWTQuery:            "jwt", // change default from "token" as it used for deleteme
		AvatarStore:         avas,
		AvatarResizeLimit:   s.Avatar.RszLmt,
		AvatarIdenticonSize: s.Avatar.IdenticonSize,
		AvatarRoutePath:     "/api/v1/avatar",
		Logger:              rest.MaskedLog{L: log.Default()}, // its error lines carry the request url with the login address
		RefreshCache:        authRefreshCache,
		UseGravatar:         true,
		AudSecrets:          s.Admin.RPC.SecretPerSite,
	})
}

// tokenAudiences limits the audience of issued and accepted tokens to the configured sites when
// remark42 keeps the data itself. The audience is the site a user names at login, and it is echoed
// back in responses and error messages, so a value nothing serves must not be accepted at all. A
// remote store decides which sites exist on its side, so with one any audience is let through.
func (s *ServerCommand) tokenAudiences() token.Audience {
	if s.Store.Type != "bolt" {
		return nil
	}
	sites := slices.Clone(s.Sites)
	return token.AudienceFunc(func() ([]string, error) { return sites, nil })
}

// servesAudience reports whether every audience in a token is a configured site, compared exactly.
// tokenAudiences restricts the auth package's own check, which ignores case, while the bolt store
// keys sites by the exact name
func (s *ServerCommand) servesAudience(auds []string) bool {
	if s.Store.Type != "bolt" {
		return true
	}
	for _, aud := range auds {
		if !slices.Contains(s.Sites, aud) {
			return false
		}
	}
	return true
}

// demoSiteID is the site the demo pages under /web show: the first configured one, as the docker
// image picks it from SITE. It is returned exactly as configured, since the stores register it
// that way and look it up by exact match
func (s *ServerCommand) demoSiteID() string {
	for _, site := range s.Sites {
		if strings.TrimSpace(site) != "" {
			return site
		}
	}
	return ""
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

// startTelegramAuthAndNotify initializes telegram notify and the shared Telegram Bot update loop.
// Does nothing if telegram auth and notifications are disabled.
func (s *ServerCommand) startTelegramAuthAndNotify(
	ctx context.Context, telegramAuth *provider.TelegramHandler,
) (tg *notify.Telegram, err error) {
	notifyWanted := contains("telegram", s.Notify.Users) || contains("telegram", s.Notify.Admins)
	if !notifyWanted && telegramAuth == nil {
		return nil, nil
	}

	if tg, err = s.makeTelegramNotify(); err != nil {
		// Auth cannot serve a request without an update source. A caller-supplied base is also
		// security-sensitive because the bot token travels in its path, so either configuration
		// fails startup instead of leaving the requested feature silently disabled.
		if telegramAuth != nil || s.Telegram.APIURL != "" {
			return nil, fmt.Errorf("failed to make telegram update source: %w", err)
		}
		log.Printf("[WARN] failed to make telegram notify service, %s", err)
		return nil, nil
	}

	telegramReceivers := make([]providers.TGUpdatesReceiver, 0, 2)
	if notifyWanted {
		telegramReceivers = append(telegramReceivers, tg)
	}
	if telegramAuth != nil {
		// ProcessUpdate owns the auth handler's request state when the shared dispatcher is used.
		// Set it up before the HTTP server can accept a login: the dispatcher's first poll is
		// five seconds away, and a login before that otherwise fails as if no listener existed.
		if err := telegramAuth.ProcessUpdate(ctx, `{"ok":true,"result":[]}`); err != nil {
			return nil, fmt.Errorf("failed to set up telegram auth update receiver: %w", err)
		}
		telegramReceivers = append(telegramReceivers, telegramAuth)
	}
	// start bot messages receiver for both notify and auth services
	go providers.DispatchTelegramUpdates(ctx, tg, telegramReceivers, time.Second*5)

	return tg, nil
}

// splitAtCommas split s at commas, ignoring commas in strings.
// Eliminate leading and trailing dbl quotes in each element only if both presented
// based on https://stackoverflow.com/a/59318708
func splitAtCommas(s string) []string {
	cleanup := func(s string) string {
		res := strings.TrimSpace(s)
		if res == "" {
			return res
		}
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
	cache.LoadingCache[token.Claims]
}

func newAuthRefreshCache() *authRefreshCache {
	o := cache.NewOpts[token.Claims]()
	expirableCache, _ := cache.NewExpirableCache(o.TTL(5 * time.Minute))
	return &authRefreshCache{LoadingCache: expirableCache}
}

// Get implements cache getter with key converted to string
func (c *authRefreshCache) Get(key string) (token.Claims, bool) {
	return c.Peek(key)
}

// Set implements cache setter with key converted to string
func (c *authRefreshCache) Set(key string, value token.Claims) {
	_, _ = c.LoadingCache.Get(key, func() (token.Claims, error) { return value, nil })
}
