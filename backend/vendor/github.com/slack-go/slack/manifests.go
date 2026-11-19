package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
)

// Manifest is an application manifest schema
type Manifest struct {
	Metadata        ManifestMetadata `json:"_metadata,omitempty" yaml:"_metadata,omitempty"`
	Display         Display          `json:"display_information" yaml:"display_information"`
	Settings        Settings         `json:"settings,omitempty" yaml:"settings,omitempty"`
	Features        Features         `json:"features,omitempty" yaml:"features,omitempty"`
	OAuthConfig     OAuthConfig      `json:"oauth_config,omitempty" yaml:"oauth_config,omitempty"`
	OutgoingDomains []string         `json:"outgoing_domains,omitempty" yaml:"outgoing_domains,omitempty"`
}

type createManifestParams struct {
	teamID string
}

// CreateManifestOption configures an apps.manifest.create request.
type CreateManifestOption func(*createManifestParams)

// CreateManifestOptionTeamID sets the workspace where an app is created when
// using an organization-level configuration token.
func CreateManifestOptionTeamID(teamID string) CreateManifestOption {
	return func(params *createManifestParams) {
		params.teamID = teamID
	}
}

// CreateManifest creates an app from an app manifest.
// For more details, see CreateManifestContext documentation.
func (api *Client) CreateManifest(manifest *Manifest, token string, options ...CreateManifestOption) (*ManifestResponse, error) {
	return api.CreateManifestContext(context.Background(), manifest, token, options...)
}

// CreateManifestContext creates an app from an app manifest with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.manifest.create
func (api *Client) CreateManifestContext(ctx context.Context, manifest *Manifest, token string, options ...CreateManifestOption) (*ManifestResponse, error) {
	jsonBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	return api.createManifest(ctx, jsonBytes, token, options...)
}

// CreateManifestRaw creates an app from a raw JSON app manifest. The manifest
// bytes are sent without re-marshaling.
// For more details, see CreateManifestRawContext documentation.
func (api *Client) CreateManifestRaw(manifest json.RawMessage, token string, options ...CreateManifestOption) (*ManifestResponse, error) {
	return api.CreateManifestRawContext(context.Background(), manifest, token, options...)
}

// CreateManifestRawContext creates an app from a raw JSON app manifest with a
// custom context. The manifest must be a JSON object. If token is empty, the
// configuration token set by OptionConfigToken is used.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.manifest.create
func (api *Client) CreateManifestRawContext(ctx context.Context, manifest json.RawMessage, token string, options ...CreateManifestOption) (*ManifestResponse, error) {
	if err := validateRawManifest(manifest); err != nil {
		return nil, err
	}

	return api.createManifest(ctx, manifest, token, options...)
}

func (api *Client) createManifest(ctx context.Context, manifest json.RawMessage, token string, options ...CreateManifestOption) (*ManifestResponse, error) {
	params := createManifestParams{}
	for _, option := range options {
		option(&params)
	}

	values := api.manifestValues(manifest, token)
	if params.teamID != "" {
		values.Set("team_id", params.teamID)
	}

	response := &ManifestResponse{}
	err := api.postMethod(ctx, "apps.manifest.create", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

func (api *Client) manifestValues(manifest json.RawMessage, token string) url.Values {
	if token == "" {
		token = api.configToken
	}

	return url.Values{
		"token":    {token},
		"manifest": {string(manifest)},
	}
}

func validateRawManifest(manifest json.RawMessage) error {
	if !json.Valid(manifest) {
		return errors.New("manifest must contain valid JSON")
	}

	if trimmed := bytes.TrimSpace(manifest); len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("manifest must be a JSON object")
	}

	return nil
}

// DeleteManifest permanently deletes an app created through app manifests.
// For more details, see DeleteManifestContext documentation.
func (api *Client) DeleteManifest(token string, appId string) (*SlackResponse, error) {
	return api.DeleteManifestContext(context.Background(), token, appId)
}

// DeleteManifestContext permanently deletes an app created through app manifests with a custom context.
// Slack API docs: https://api.slack.com/methods/apps.manifest.delete
func (api *Client) DeleteManifestContext(ctx context.Context, token string, appId string) (*SlackResponse, error) {
	if token == "" {
		token = api.configToken
	}

	values := url.Values{
		"token":  {token},
		"app_id": {appId},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "apps.manifest.delete", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// ExportManifest exports an app manifest from an existing app.
// For more details, see ExportManifestContext documentation.
func (api *Client) ExportManifest(token string, appId string) (*Manifest, error) {
	return api.ExportManifestContext(context.Background(), token, appId)
}

// ExportManifestContext exports an app manifest from an existing app with a custom context.
// Slack API docs: https://api.slack.com/methods/apps.manifest.export
func (api *Client) ExportManifestContext(ctx context.Context, token string, appId string) (*Manifest, error) {
	if token == "" {
		token = api.configToken
	}

	values := url.Values{
		"token":  {token},
		"app_id": {appId},
	}

	response := &ExportManifestResponse{}
	err := api.postMethod(ctx, "apps.manifest.export", values, response)
	if err != nil {
		return nil, err
	}

	return &response.Manifest, response.Err()
}

// ExportManifestRaw exports an app manifest without decoding it into Manifest.
// For more details, see ExportManifestRawContext documentation.
func (api *Client) ExportManifestRaw(token string, appID string) (json.RawMessage, error) {
	return api.ExportManifestRawContext(context.Background(), token, appID)
}

// ExportManifestRawContext exports an app manifest without decoding it into
// Manifest. The returned JSON retains unknown fields and its original
// representation. If token is empty, the configuration token set by
// OptionConfigToken is used.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.manifest.export
func (api *Client) ExportManifestRawContext(ctx context.Context, token string, appID string) (json.RawMessage, error) {
	if token == "" {
		token = api.configToken
	}

	values := url.Values{
		"token":  {token},
		"app_id": {appID},
	}

	response := &exportManifestRawResponse{}
	err := api.postMethod(ctx, "apps.manifest.export", values, response)
	if err != nil {
		return nil, err
	}

	return response.Manifest, response.Err()
}

// UpdateManifest updates an app from an app manifest.
// For more details, see UpdateManifestContext documentation.
func (api *Client) UpdateManifest(manifest *Manifest, token string, appId string) (*UpdateManifestResponse, error) {
	return api.UpdateManifestContext(context.Background(), manifest, token, appId)
}

// UpdateManifestContext updates an app from an app manifest with a custom context.
// Slack API docs: https://api.slack.com/methods/apps.manifest.update
func (api *Client) UpdateManifestContext(ctx context.Context, manifest *Manifest, token string, appId string) (*UpdateManifestResponse, error) {
	jsonBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	return api.updateManifest(ctx, jsonBytes, token, appId)
}

// UpdateManifestRaw updates an app from a raw JSON app manifest. The manifest
// bytes are sent without re-marshaling.
// For more details, see UpdateManifestRawContext documentation.
func (api *Client) UpdateManifestRaw(manifest json.RawMessage, token string, appID string) (*UpdateManifestResponse, error) {
	return api.UpdateManifestRawContext(context.Background(), manifest, token, appID)
}

// UpdateManifestRawContext updates an app from a raw JSON app manifest with a
// custom context. The manifest must be a JSON object. If token is empty, the
// configuration token set by OptionConfigToken is used.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.manifest.update
func (api *Client) UpdateManifestRawContext(ctx context.Context, manifest json.RawMessage, token string, appID string) (*UpdateManifestResponse, error) {
	if err := validateRawManifest(manifest); err != nil {
		return nil, err
	}

	return api.updateManifest(ctx, manifest, token, appID)
}

func (api *Client) updateManifest(ctx context.Context, manifest json.RawMessage, token string, appID string) (*UpdateManifestResponse, error) {
	values := api.manifestValues(manifest, token)
	values.Set("app_id", appID)

	response := &UpdateManifestResponse{}
	err := api.postMethod(ctx, "apps.manifest.update", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// ValidateManifest sends a request to apps.manifest.validate to validate your app manifest.
// For more details, see ValidateManifestContext documentation.
func (api *Client) ValidateManifest(manifest *Manifest, token string, appId string) (*ManifestResponse, error) {
	return api.ValidateManifestContext(context.Background(), manifest, token, appId)
}

// ValidateManifestContext sends a request to apps.manifest.validate to validate your app manifest with a custom context.
// appId is optional; pass an empty string to omit it.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.manifest.validate
func (api *Client) ValidateManifestContext(ctx context.Context, manifest *Manifest, token string, appId string) (*ManifestResponse, error) {
	jsonBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	return api.validateManifest(ctx, jsonBytes, token, appId)
}

// ValidateManifestRaw validates a raw JSON app manifest. The manifest bytes
// are sent without re-marshaling.
// For more details, see ValidateManifestRawContext documentation.
func (api *Client) ValidateManifestRaw(manifest json.RawMessage, token string, appID string) (*ManifestResponse, error) {
	return api.ValidateManifestRawContext(context.Background(), manifest, token, appID)
}

// ValidateManifestRawContext validates a raw JSON app manifest with a custom
// context. The manifest must be a JSON object. appID is optional; pass an empty
// string to omit it. If token is empty, the configuration token set by
// OptionConfigToken is used.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.manifest.validate
func (api *Client) ValidateManifestRawContext(ctx context.Context, manifest json.RawMessage, token string, appID string) (*ManifestResponse, error) {
	if err := validateRawManifest(manifest); err != nil {
		return nil, err
	}

	return api.validateManifest(ctx, manifest, token, appID)
}

func (api *Client) validateManifest(ctx context.Context, manifest json.RawMessage, token string, appID string) (*ManifestResponse, error) {
	values := api.manifestValues(manifest, token)

	if appID != "" {
		values.Set("app_id", appID)
	}

	response := &ManifestResponse{}
	err := api.postMethod(ctx, "apps.manifest.validate", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// ManifestMetadata is a group of settings that describe the manifest
type ManifestMetadata struct {
	MajorVersion int `json:"major_version,omitempty" yaml:"major_version,omitempty"`
	MinorVersion int `json:"minor_version,omitempty" yaml:"minor_version,omitempty"`
}

// Display is a group of settings that describe parts of an app's appearance within Slack
type Display struct {
	Name            string `json:"name" yaml:"name"`
	Description     string `json:"description,omitempty" yaml:"description,omitempty"`
	LongDescription string `json:"long_description,omitempty" yaml:"long_description,omitempty"`
	BackgroundColor string `json:"background_color,omitempty" yaml:"background_color,omitempty"`
}

// Settings is a group of settings corresponding to the Settings section of the app config pages.
type Settings struct {
	AllowedIPAddressRanges []string                  `json:"allowed_ip_address_ranges,omitempty" yaml:"allowed_ip_address_ranges,omitempty"`
	EventSubscriptions     *EventSubscriptions       `json:"event_subscriptions,omitempty" yaml:"event_subscriptions,omitempty"`
	FunctionRuntime        ManifestFunctionRuntime   `json:"function_runtime,omitempty" yaml:"function_runtime,omitempty"`
	IncomingWebhooks       *ManifestIncomingWebhooks `json:"incoming_webhooks,omitempty" yaml:"incoming_webhooks,omitempty"`
	Interactivity          *Interactivity            `json:"interactivity,omitempty" yaml:"interactivity,omitempty"`
	IsMCPEnabled           bool                      `json:"is_mcp_enabled,omitempty" yaml:"is_mcp_enabled,omitempty"`
	OrgDeployEnabled       bool                      `json:"org_deploy_enabled,omitempty" yaml:"org_deploy_enabled,omitempty"`
	SIWSLinks              *ManifestSIWSLinks        `json:"siws_links,omitempty" yaml:"siws_links,omitempty"`
	SocketModeEnabled      bool                      `json:"socket_mode_enabled,omitempty" yaml:"socket_mode_enabled,omitempty"`
	TokenRotationEnabled   bool                      `json:"token_rotation_enabled,omitempty" yaml:"token_rotation_enabled,omitempty"`
}

// EventSubscriptions is a group of settings that describe the Events API configuration
type EventSubscriptions struct {
	RequestUrl            string                         `json:"request_url,omitempty" yaml:"request_url,omitempty"`
	BotEvents             []string                       `json:"bot_events,omitempty" yaml:"bot_events,omitempty"`
	UserEvents            []string                       `json:"user_events,omitempty" yaml:"user_events,omitempty"`
	MetadataSubscriptions []ManifestMetadataSubscription `json:"metadata_subscriptions,omitempty" yaml:"metadata_subscriptions,omitempty"`
}

// ManifestMetadataSubscription is a message metadata event type that the app subscribes to
type ManifestMetadataSubscription struct {
	AppID     string `json:"app_id" yaml:"app_id"`
	EventType string `json:"event_type" yaml:"event_type"`
}

// Interactivity is a group of settings that describe the interactivity configuration
type Interactivity struct {
	IsEnabled             bool   `json:"is_enabled" yaml:"is_enabled"`
	RequestUrl            string `json:"request_url,omitempty" yaml:"request_url,omitempty"`
	MessageMenuOptionsUrl string `json:"message_menu_options_url,omitempty" yaml:"message_menu_options_url,omitempty"`
}

// ManifestIncomingWebhooks is a group of settings that describe the incoming webhooks configuration
type ManifestIncomingWebhooks struct {
	IncomingWebhooksEnabled bool `json:"incoming_webhooks_enabled" yaml:"incoming_webhooks_enabled"`
}

// ManifestSIWSLinks is a group of settings that describe Sign in with Slack (SIWS) links
type ManifestSIWSLinks struct {
	InitiateURI string `json:"initiate_uri,omitempty" yaml:"initiate_uri,omitempty"`
}

// ManifestFunctionRuntime is where the functions declared in the manifest run
type ManifestFunctionRuntime string

const (
	ManifestFunctionRuntimeRemote ManifestFunctionRuntime = "remote"
	ManifestFunctionRuntimeSlack  ManifestFunctionRuntime = "slack"
)

// Features is a group of settings corresponding to the Features section of the app config pages
type Features struct {
	AgentView       *AgentView               `json:"agent_view,omitempty" yaml:"agent_view,omitempty"`
	AppHome         AppHome                  `json:"app_home,omitempty" yaml:"app_home,omitempty"`
	AssistantView   *AssistantView           `json:"assistant_view,omitempty" yaml:"assistant_view,omitempty"`
	BotUser         BotUser                  `json:"bot_user,omitempty" yaml:"bot_user,omitempty"`
	CodeChannels    *ManifestCodeChannels    `json:"code_channels,omitempty" yaml:"code_channels,omitempty"`
	RichPreviews    *ManifestRichPreviews    `json:"rich_previews,omitempty" yaml:"rich_previews,omitempty"`
	Search          *ManifestSearch          `json:"search,omitempty" yaml:"search,omitempty"`
	Shortcuts       []Shortcut               `json:"shortcuts,omitempty" yaml:"shortcuts,omitempty"`
	SlashCommands   []ManifestSlashCommand   `json:"slash_commands,omitempty" yaml:"slash_commands,omitempty"`
	UnfurlDomains   []string                 `json:"unfurl_domains,omitempty" yaml:"unfurl_domains,omitempty"`
	UserConnections *ManifestUserConnections `json:"user_connections,omitempty" yaml:"user_connections,omitempty"`
	WorkflowSteps   []WorkflowStep           `json:"workflow_steps,omitempty" yaml:"workflow_steps,omitempty"`
}

// ManifestUserConnections configures standardized app authentication.
// AuthHosts lists permitted hosts without a scheme, including their subdomains.
// Hosts remain configured when IsEnabled is false.
type ManifestUserConnections struct {
	IsEnabled bool     `json:"is_enabled" yaml:"is_enabled"`
	AuthHosts []string `json:"auth_hosts" yaml:"auth_hosts"`
}

// ManifestCodeChannels configures code channels and their agent slash command URL.
type ManifestCodeChannels struct {
	Enabled         bool   `json:"enabled" yaml:"enabled"`
	SlashCommandURL string `json:"slash_command_url,omitempty" yaml:"slash_command_url,omitempty"`
}

// AgentView is a group of settings that describe the agent view configuration for apps using AI features
type AgentView struct {
	AgentDescription string                    `json:"agent_description" yaml:"agent_description"`
	SuggestedPrompts []ManifestSuggestedPrompt `json:"suggested_prompts,omitempty" yaml:"suggested_prompts,omitempty"`
	Actions          []ManifestAgentAction     `json:"actions,omitempty" yaml:"actions,omitempty"`
}

// AssistantView is a group of settings that describe the assistant view configuration for apps using AI features
type AssistantView struct {
	AssistantDescription string                    `json:"assistant_description" yaml:"assistant_description"`
	SuggestedPrompts     []ManifestSuggestedPrompt `json:"suggested_prompts,omitempty" yaml:"suggested_prompts,omitempty"`
}

// ManifestSuggestedPrompt is a hard-coded prompt shown in the agent or assistant container
type ManifestSuggestedPrompt struct {
	Title   string `json:"title" yaml:"title"`
	Message string `json:"message" yaml:"message"`
}

// ManifestAgentAction is an action available in the agent container
type ManifestAgentAction struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
}

// ManifestRichPreviews is a group of settings that describe rich previews
type ManifestRichPreviews struct {
	IsActive    bool     `json:"is_active" yaml:"is_active"`
	EntityTypes []string `json:"entity_types,omitempty" yaml:"entity_types,omitempty"`
}

// ManifestSearch is a group of settings that describe the enterprise search connector
type ManifestSearch struct {
	SearchFunctionCallbackID        string `json:"search_function_callback_id" yaml:"search_function_callback_id"`
	SearchFiltersFunctionCallbackID string `json:"search_filters_function_callback_id,omitempty" yaml:"search_filters_function_callback_id,omitempty"`
}

// AppHome is a group of settings that describe the App Home configuration
type AppHome struct {
	HomeTabEnabled             bool `json:"home_tab_enabled,omitempty" yaml:"home_tab_enabled,omitempty"`
	MessagesTabEnabled         bool `json:"messages_tab_enabled,omitempty" yaml:"messages_tab_enabled,omitempty"`
	MessagesTabReadOnlyEnabled bool `json:"messages_tab_read_only_enabled,omitempty" yaml:"messages_tab_read_only_enabled,omitempty"`
}

// BotUser is a group of settings that describe bot user configuration
type BotUser struct {
	DisplayName  string `json:"display_name" yaml:"display_name"`
	AlwaysOnline bool   `json:"always_online,omitempty" yaml:"always_online,omitempty"`
}

// Shortcut is a group of settings that describes shortcut configuration
type Shortcut struct {
	Name        string       `json:"name" yaml:"name"`
	CallbackID  string       `json:"callback_id" yaml:"callback_id"`
	Description string       `json:"description" yaml:"description"`
	Type        ShortcutType `json:"type" yaml:"type"`
}

// ShortcutType is a new string type for the available types of shortcuts
type ShortcutType string

const (
	MessageShortcut ShortcutType = "message"
	GlobalShortcut  ShortcutType = "global"
)

// ManifestSlashCommand is a group of settings that describes slash command configuration
type ManifestSlashCommand struct {
	Command      string `json:"command" yaml:"command"`
	Description  string `json:"description" yaml:"description"`
	ShouldEscape bool   `json:"should_escape,omitempty" yaml:"should_escape,omitempty"`
	Url          string `json:"url,omitempty" yaml:"url,omitempty"`
	UsageHint    string `json:"usage_hint,omitempty" yaml:"usage_hint,omitempty"`
}

// WorkflowStep is a group of settings that describes workflow steps configuration
type WorkflowStep struct {
	Name       string `json:"name" yaml:"name"`
	CallbackID string `json:"callback_id" yaml:"callback_id"`
}

// OAuthConfig is a group of settings that describe OAuth configuration for the app
type OAuthConfig struct {
	PKCEEnabled  bool        `json:"pkce_enabled,omitempty" yaml:"pkce_enabled,omitempty"`
	RedirectUrls []string    `json:"redirect_urls,omitempty" yaml:"redirect_urls,omitempty"`
	Scopes       OAuthScopes `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	// A pointer because Slack treats a missing key differently from false: it
	// accepts false only when settings.org_deploy_enabled is true.
	TokenManagementEnabled *bool `json:"token_management_enabled,omitempty" yaml:"token_management_enabled,omitempty"`
}

// OAuthScopes is a group of settings that describe permission scopes configuration
type OAuthScopes struct {
	Bot          []string `json:"bot,omitempty" yaml:"bot,omitempty"`
	User         []string `json:"user,omitempty" yaml:"user,omitempty"`
	BotOptional  []string `json:"bot_optional,omitempty" yaml:"bot_optional,omitempty"`
	UserOptional []string `json:"user_optional,omitempty" yaml:"user_optional,omitempty"`
}

// ManifestResponse is the response returned by the API for apps.manifest.x endpoints.
// AppId, Credentials and OAuthAuthorizeUrl are only populated by apps.manifest.create.
type ManifestResponse struct {
	AppId             string                    `json:"app_id,omitempty"`
	Credentials       *ManifestCredentials      `json:"credentials,omitempty"`
	OAuthAuthorizeUrl string                    `json:"oauth_authorize_url,omitempty"`
	Errors            []ManifestValidationError `json:"errors,omitempty"`
	SlackResponse
}

// ManifestValidationError is an error message returned for invalid manifests
type ManifestValidationError struct {
	Message string `json:"message"`
	Pointer string `json:"pointer"`
}

// ManifestCredentials are the app credentials returned by the apps.manifest.create endpoint.
// They are only returned on creation and cannot be retrieved afterwards through the API.
type ManifestCredentials struct {
	ClientId          string `json:"client_id,omitempty"`
	ClientSecret      string `json:"client_secret,omitempty"`
	VerificationToken string `json:"verification_token,omitempty"`
	SigningSecret     string `json:"signing_secret,omitempty"`
}

type ExportManifestResponse struct {
	Manifest Manifest `json:"manifest,omitempty"`
	SlackResponse
}

type exportManifestRawResponse struct {
	Manifest json.RawMessage `json:"manifest"`
	SlackResponse
}

type UpdateManifestResponse struct {
	AppId              string `json:"app_id,omitempty"`
	PermissionsUpdated bool   `json:"permissions_updated,omitempty"`
	ManifestResponse
}
