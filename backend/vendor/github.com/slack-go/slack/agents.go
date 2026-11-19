package slack

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// AgentConversationCreateResponse is the response from agents.conversations.create.
type AgentConversationCreateResponse struct {
	SlackResponse
	ChannelID string `json:"channel_id"`
}

type createAgentConversationParams struct {
	teamID                 string
	sessionID              string
	isPrivate              *bool
	originChannelID        string
	originMessageTimestamp string
}

// CreateAgentConversationOption configures a code channel creation request.
type CreateAgentConversationOption func(*createAgentConversationParams)

// CreateAgentConversationOptionTeamID sets the workspace for the code channel.
func CreateAgentConversationOptionTeamID(teamID string) CreateAgentConversationOption {
	return func(params *createAgentConversationParams) {
		params.teamID = teamID
	}
}

// CreateAgentConversationOptionSessionID sets the session ID used to make creation idempotent.
func CreateAgentConversationOptionSessionID(sessionID string) CreateAgentConversationOption {
	return func(params *createAgentConversationParams) {
		params.sessionID = sessionID
	}
}

// CreateAgentConversationOptionIsPrivate overrides the privacy inherited from the origin channel.
func CreateAgentConversationOptionIsPrivate(isPrivate bool) CreateAgentConversationOption {
	return func(params *createAgentConversationParams) {
		params.isPrivate = &isPrivate
	}
}

// CreateAgentConversationOptionOriginChannelID sets the channel containing the origin message.
func CreateAgentConversationOptionOriginChannelID(channelID string) CreateAgentConversationOption {
	return func(params *createAgentConversationParams) {
		params.originChannelID = channelID
	}
}

// CreateAgentConversationOptionOriginMessageTimestamp sets the origin message timestamp.
func CreateAgentConversationOptionOriginMessageTimestamp(timestamp string) CreateAgentConversationOption {
	return func(params *createAgentConversationParams) {
		params.originMessageTimestamp = timestamp
	}
}

// CreateAgentConversation creates a code channel for this agent.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.create
//
// Pass an empty name to let Slack name the channel from the origin message.
// OriginChannelID and OriginMessageTimestamp must be set together.
func (api *Client) CreateAgentConversation(name string, options ...CreateAgentConversationOption) (*AgentConversationCreateResponse, error) {
	return api.CreateAgentConversationContext(context.Background(), name, options...)
}

// CreateAgentConversationContext creates a code channel for this agent with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.create
//
// Pass an empty name to let Slack name the channel from the origin message.
// OriginChannelID and OriginMessageTimestamp must be set together.
func (api *Client) CreateAgentConversationContext(ctx context.Context, name string, options ...CreateAgentConversationOption) (*AgentConversationCreateResponse, error) {
	params := createAgentConversationParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{"token": {api.token}}
	if name != "" {
		values.Add("name", name)
	}
	if params.teamID != "" {
		values.Add("team_id", params.teamID)
	}
	if params.sessionID != "" {
		values.Add("session_id", params.sessionID)
	}
	if params.isPrivate != nil {
		values.Add("is_private", strconv.FormatBool(*params.isPrivate))
	}
	if params.originChannelID != "" {
		values.Add("origin_channel_id", params.originChannelID)
	}
	if params.originMessageTimestamp != "" {
		values.Add("origin_message_ts", params.originMessageTimestamp)
	}

	response := &AgentConversationCreateResponse{}
	if err := api.postMethod(ctx, "agents.conversations.create", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

type archiveAgentConversationParams struct {
	summaryMessageTimestamp string
}

// ArchiveAgentConversationOption configures a code channel archival request.
type ArchiveAgentConversationOption func(*archiveAgentConversationParams)

// ArchiveAgentConversationOptionSummaryMessageTimestamp sets the summary to share with the origin thread.
func ArchiveAgentConversationOptionSummaryMessageTimestamp(timestamp string) ArchiveAgentConversationOption {
	return func(params *archiveAgentConversationParams) {
		params.summaryMessageTimestamp = timestamp
	}
}

// ArchiveAgentConversation archives a code channel.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.archive
func (api *Client) ArchiveAgentConversation(channelID string, options ...ArchiveAgentConversationOption) error {
	return api.ArchiveAgentConversationContext(context.Background(), channelID, options...)
}

// ArchiveAgentConversationContext archives a code channel with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.archive
func (api *Client) ArchiveAgentConversationContext(ctx context.Context, channelID string, options ...ArchiveAgentConversationOption) error {
	params := archiveAgentConversationParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}
	if params.summaryMessageTimestamp != "" {
		values.Add("summary_message_ts", params.summaryMessageTimestamp)
	}
	response := SlackResponse{}
	if err := api.postMethod(ctx, "agents.conversations.archive", values, &response); err != nil {
		return err
	}
	return response.Err()
}

// AgentCommand is an agent slash command registered in a code channel.
// Name does not include a leading slash.
type AgentCommand struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ArgumentHint string `json:"argument_hint,omitempty"`
	ShouldEscape bool   `json:"should_escape,omitempty"`
}

// AgentConversationSetCommandsResponse is the response from agents.conversations.setCommands.
type AgentConversationSetCommandsResponse struct {
	SlackResponse
	ChannelID    string `json:"channel_id"`
	CommandCount int    `json:"command_count"`
}

// SetAgentConversationCommands replaces this agent's slash commands in a code channel.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.setCommands
//
// A nil or empty commands slice clears this agent's commands.
func (api *Client) SetAgentConversationCommands(channelID string, commands []AgentCommand) (*AgentConversationSetCommandsResponse, error) {
	return api.SetAgentConversationCommandsContext(context.Background(), channelID, commands)
}

// SetAgentConversationCommandsContext replaces this agent's slash commands with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.setCommands
//
// A nil or empty commands slice clears this agent's commands.
func (api *Client) SetAgentConversationCommandsContext(ctx context.Context, channelID string, commands []AgentCommand) (*AgentConversationSetCommandsResponse, error) {
	if commands == nil {
		commands = []AgentCommand{}
	}
	commandsJSON, err := json.Marshal(commands)
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
		"commands":   {string(commandsJSON)},
	}
	response := &AgentConversationSetCommandsResponse{}
	if err := api.postMethod(ctx, "agents.conversations.setCommands", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// AgentContextBarItem is a link, action, or informational item in a code channel's
// context bar. Key and Label are required. ItemType may be "info" (the default)
// or "action". Action items should not also specify a URL.
type AgentContextBarItem struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Icon     string `json:"icon,omitempty"`
	URL      string `json:"url,omitempty"`
	ItemType string `json:"item_type,omitempty"`
}

// AgentSummaryMessage identifies the message representing a session's summary.
// ThreadTS is the parent timestamp when the summary is a thread reply.
type AgentSummaryMessage struct {
	MessageTS string `json:"message_ts"`
	ThreadTS  string `json:"thread_ts,omitempty"`
}

// AgentCodeChannelProperties contains the writable code channel properties.
type AgentCodeChannelProperties struct {
	// ContextBarItems replaces the entire context bar (at most five items).
	// A nil slice leaves it unchanged; a non-nil empty slice clears it.
	ContextBarItems []AgentContextBarItem `json:"context_bar_items,omitzero"`
	SummaryMessage  *AgentSummaryMessage  `json:"summary_message,omitempty"`
}

// AgentResourceProperties describes the external resource for an agent session.
// Nil fields are omitted; a pointer to an empty string sends an explicit empty value.
type AgentResourceProperties struct {
	URL          *string `json:"url,omitempty"`
	ResourceType *string `json:"resource_type,omitempty"`
	Title        *string `json:"title,omitempty"`
	Provider     *string `json:"provider,omitempty"`
}

type setAgentConversationPropertiesParams struct {
	codeChannel   *AgentCodeChannelProperties
	agentResource *AgentResourceProperties
}

// SetAgentConversationPropertiesOption configures a code channel property update.
type SetAgentConversationPropertiesOption func(*setAgentConversationPropertiesParams)

// SetAgentConversationPropertiesOptionCodeChannel sets the context bar or summary
// message properties to update. Only provided fields are changed.
func SetAgentConversationPropertiesOptionCodeChannel(properties AgentCodeChannelProperties) SetAgentConversationPropertiesOption {
	return func(params *setAgentConversationPropertiesParams) {
		params.codeChannel = &properties
	}
}

// SetAgentConversationPropertiesOptionAgentResource sets the external resource
// properties to update. Only provided fields are changed.
func SetAgentConversationPropertiesOptionAgentResource(properties AgentResourceProperties) SetAgentConversationPropertiesOption {
	return func(params *setAgentConversationPropertiesParams) {
		params.agentResource = &properties
	}
}

// AgentConversationSetPropertiesResponse is the response from
// agents.conversations.setProperties. Check warnings for truncated context bars
// or action items that Slack converted to informational links.
type AgentConversationSetPropertiesResponse struct {
	SlackResponse
	ChannelID string `json:"channel_id"`
}

// SetAgentConversationProperties updates the context bar, summary message, or
// external resource of a code channel. At least one property must be provided.
// Use SetAgentSessionStatus and RenameAgentSession to change its status and title.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.setProperties
func (api *Client) SetAgentConversationProperties(channelID string, options ...SetAgentConversationPropertiesOption) (*AgentConversationSetPropertiesResponse, error) {
	return api.SetAgentConversationPropertiesContext(context.Background(), channelID, options...)
}

// SetAgentConversationPropertiesContext updates code channel properties with a
// custom context. At least one property must be provided.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.setProperties
func (api *Client) SetAgentConversationPropertiesContext(ctx context.Context, channelID string, options ...SetAgentConversationPropertiesOption) (*AgentConversationSetPropertiesResponse, error) {
	params := setAgentConversationPropertiesParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}
	if params.codeChannel != nil {
		codeChannelJSON, err := json.Marshal(params.codeChannel)
		if err != nil {
			return nil, err
		}
		values.Add("code_channel", string(codeChannelJSON))
	}
	if params.agentResource != nil {
		agentResourceJSON, err := json.Marshal(params.agentResource)
		if err != nil {
			return nil, err
		}
		values.Add("agent_resource", string(agentResourceJSON))
	}

	response := &AgentConversationSetPropertiesResponse{}
	if err := api.postMethod(ctx, "agents.conversations.setProperties", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// Lifecycle statuses accepted by agents.sessions.setStatus.
const (
	AgentSessionStatusActive     = "active"
	AgentSessionStatusProcessing = "processing"
	AgentSessionStatusSuspended  = "suspended"
	AgentSessionStatusClosed     = "closed"
)

// AgentSessionSetStatusParameters are the parameters for SetAgentSessionStatus.
// Title and InitiatorUserID only apply when the call creates the session.
//
// IconEmoji, IconURL and Username form one set of overrides that persists across
// calls. A call that leaves all three empty keeps the current overrides. A call
// that sets any of them replaces the whole set, so a field left empty is cleared.
type AgentSessionSetStatusParameters struct {
	Status          string `json:"status"`
	ChannelID       string `json:"channel_id,omitempty"`
	ThreadTS        string `json:"thread_ts,omitempty"`
	Title           string `json:"title,omitempty"`
	InitiatorUserID string `json:"initiator_user_id,omitempty"`
	IconEmoji       string `json:"icon_emoji,omitempty"`
	IconURL         string `json:"icon_url,omitempty"`
	Username        string `json:"username,omitempty"`
}

// AgentSessionSetStatusResponse is the response from agents.sessions.setStatus.
// Status covers every agent in the session; AgentStatus is the caller's own.
type AgentSessionSetStatusResponse struct {
	SlackResponse
	Status      string `json:"status"`
	AgentStatus string `json:"agent_status"`
	Title       string `json:"title,omitempty"`
}

// AgentSessionRenameParameters are the parameters for RenameAgentSession.
type AgentSessionRenameParameters struct {
	Title     string `json:"title"`
	ChannelID string `json:"channel_id,omitempty"`
	ThreadTS  string `json:"thread_ts,omitempty"`
}

// AgentSessionRenameResponse is the response from agents.sessions.rename.
type AgentSessionRenameResponse struct {
	SlackResponse
	Title string `json:"title"`
}

// SetAgentSessionStatus sets an agent session's lifecycle status, creating the session if needed.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.sessions.setStatus
func (api *Client) SetAgentSessionStatus(params AgentSessionSetStatusParameters) (*AgentSessionSetStatusResponse, error) {
	return api.SetAgentSessionStatusContext(context.Background(), params)
}

// SetAgentSessionStatusContext sets an agent session's lifecycle status with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.sessions.setStatus
func (api *Client) SetAgentSessionStatusContext(ctx context.Context, params AgentSessionSetStatusParameters) (*AgentSessionSetStatusResponse, error) {
	values := url.Values{
		"token":  {api.token},
		"status": {params.Status},
	}

	if params.ChannelID != "" {
		values.Add("channel_id", params.ChannelID)
	}

	if params.ThreadTS != "" {
		values.Add("thread_ts", params.ThreadTS)
	}

	if params.Title != "" {
		values.Add("title", params.Title)
	}

	if params.InitiatorUserID != "" {
		values.Add("initiator_user_id", params.InitiatorUserID)
	}

	if params.IconEmoji != "" {
		values.Add("icon_emoji", params.IconEmoji)
	}

	if params.IconURL != "" {
		values.Add("icon_url", params.IconURL)
	}

	if params.Username != "" {
		values.Add("username", params.Username)
	}

	response := &AgentSessionSetStatusResponse{}

	err := api.postMethod(ctx, "agents.sessions.setStatus", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// RenameAgentSession sets the title of an agent session.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.sessions.rename
func (api *Client) RenameAgentSession(params AgentSessionRenameParameters) (*AgentSessionRenameResponse, error) {
	return api.RenameAgentSessionContext(context.Background(), params)
}

// RenameAgentSessionContext sets the title of an agent session with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.sessions.rename
func (api *Client) RenameAgentSessionContext(ctx context.Context, params AgentSessionRenameParameters) (*AgentSessionRenameResponse, error) {
	values := url.Values{
		"token": {api.token},
		"title": {params.Title},
	}

	if params.ChannelID != "" {
		values.Add("channel_id", params.ChannelID)
	}

	if params.ThreadTS != "" {
		values.Add("thread_ts", params.ThreadTS)
	}

	response := &AgentSessionRenameResponse{}

	err := api.postMethod(ctx, "agents.sessions.rename", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}
