package slack

import (
	"context"
	"encoding/json"
	"net/url"
)

// View types documented for agents.conversations.setView.
const (
	AgentViewTypeHTML     = "html"
	AgentViewTypeDiff     = "diff"
	AgentViewTypeBlockKit = "block_kit"
	AgentViewTypeCanvas   = "canvas"
	// AgentViewTypePullRequest is documented by Slack, but has been observed to
	// return invalid_view_type. Use a diff view when the server rejects it.
	AgentViewTypePullRequest = "pull_request"
)

// AgentViewCSP declares HTTPS origins for a code channel's HTML view.
// Slack validates the origins. ConnectDomains is stored but is not yet honored
// when rendering the view.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.setView
type AgentViewCSP struct {
	ResourceDomains []string `json:"resource_domains,omitempty"`
	ConnectDomains  []string `json:"connect_domains,omitempty"`
}

type setAgentConversationViewParams struct {
	values url.Values
	blocks []Block
	csp    *AgentViewCSP
}

// SetAgentConversationViewOption configures a code channel view creation or update.
type SetAgentConversationViewOption func(*setAgentConversationViewParams)

// SetAgentConversationViewOptionType sets the view type. Slack defaults to html.
func SetAgentConversationViewOptionType(viewType string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("type", viewType)
	}
}

// SetAgentConversationViewOptionViewKey sets the stable identity used to create
// or update the same view. It is required for html, block_kit, and canvas views,
// and ignored for diff views, which are unique within a channel.
func SetAgentConversationViewOptionViewKey(viewKey string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("view_key", viewKey)
	}
}

// SetAgentConversationViewOptionContent sets the HTML document or unified diff.
// An empty string is sent explicitly. Slack limits content to 1,000,000 bytes.
func SetAgentConversationViewOptionContent(content string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("content", content)
	}
}

// SetAgentConversationViewOptionBlocks sets the blocks for a block_kit view.
// Calling it without blocks sends an empty array.
func SetAgentConversationViewOptionBlocks(blocks ...Block) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.blocks = append([]Block{}, blocks...)
	}
}

// SetAgentConversationViewOptionCanvasID sets the canvas to attach as a view.
func SetAgentConversationViewOptionCanvasID(canvasID string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("canvas_id", canvasID)
	}
}

// SetAgentConversationViewOptionAccessLevel sets the channel's access to a canvas
// view: read, write, or comment. Slack defaults to write.
func SetAgentConversationViewOptionAccessLevel(accessLevel string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("access_level", accessLevel)
	}
}

// SetAgentConversationViewOptionAgentContentHash records an opaque hash of the
// canvas-derived markdown last written by the agent, for detecting human edits.
func SetAgentConversationViewOptionAgentContentHash(hash string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("agent_content_hash", hash)
	}
}

// SetAgentConversationViewOptionPRURL sets the URL for a pull_request view.
func SetAgentConversationViewOptionPRURL(prURL string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("pr_url", prURL)
	}
}

// SetAgentConversationViewOptionBaseBranch sets the base branch label of a diff view.
func SetAgentConversationViewOptionBaseBranch(branch string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("base_branch", branch)
	}
}

// SetAgentConversationViewOptionHeadBranch sets the head branch label of a diff view.
func SetAgentConversationViewOptionHeadBranch(branch string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("head_branch", branch)
	}
}

// SetAgentConversationViewOptionName sets the view tab's display name.
func SetAgentConversationViewOptionName(name string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("name", name)
	}
}

// SetAgentConversationViewOptionLabel sets the legacy view tab label.
//
// Deprecated: Use SetAgentConversationViewOptionName instead.
func SetAgentConversationViewOptionLabel(label string) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.values.Set("label", label)
	}
}

// SetAgentConversationViewOptionCSP sets the HTML view's content security policy
// domain declarations.
func SetAgentConversationViewOptionCSP(csp AgentViewCSP) SetAgentConversationViewOption {
	return func(params *setAgentConversationViewParams) {
		params.csp = &csp
	}
}

// AgentConversationSetViewResponse is the response from agents.conversations.setView.
// ViewID identifies the channel tab; FileID and CanvasID identify backing content.
type AgentConversationSetViewResponse struct {
	SlackResponse
	ChannelID      string `json:"channel_id"`
	ViewID         string `json:"view_id"`
	FileID         string `json:"file_id"`
	CanvasID       string `json:"canvas_id"`
	ContentVersion int    `json:"content_version"`
	Type           string `json:"type"`
}

// SetAgentConversationView creates or updates a view in a code channel.
// HTML and diff views require content; block_kit views require blocks; canvas
// views require a canvas ID; pull_request views require a pull request URL.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.setView
func (api *Client) SetAgentConversationView(channelID string, options ...SetAgentConversationViewOption) (*AgentConversationSetViewResponse, error) {
	return api.SetAgentConversationViewContext(context.Background(), channelID, options...)
}

// SetAgentConversationViewContext creates or updates a code channel view with a
// custom context. Slack validates the view type and its required arguments.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.setView
func (api *Client) SetAgentConversationViewContext(ctx context.Context, channelID string, options ...SetAgentConversationViewOption) (*AgentConversationSetViewResponse, error) {
	params := setAgentConversationViewParams{
		values: url.Values{
			"token":      {api.token},
			"channel_id": {channelID},
		},
	}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}
	if params.blocks != nil {
		blocksJSON, err := json.Marshal(params.blocks)
		if err != nil {
			return nil, err
		}
		params.values.Set("blocks", string(blocksJSON))
	}
	if params.csp != nil {
		cspJSON, err := json.Marshal(params.csp)
		if err != nil {
			return nil, err
		}
		params.values.Set("csp", string(cspJSON))
	}

	response := &AgentConversationSetViewResponse{}
	if err := api.postMethod(ctx, "agents.conversations.setView", params.values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// AgentConversationView describes an entry in the legacy views array returned by
// agents.conversations.listViews.
// ViewKey may be absent for a diff view. DateAdded is a Unix timestamp in seconds.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.listViews
type AgentConversationView struct {
	ViewID         string `json:"view_id"`
	ViewKey        string `json:"view_key,omitempty"`
	FileID         string `json:"file_id"`
	Label          string `json:"label"`
	ContentVersion int    `json:"content_version"`
	DateAdded      int64  `json:"date_added"`
}

// AgentConversationSessionView describes an entry in the session_views array
// returned by agents.conversations.listViews. This array has been observed in
// live Slack responses alongside an empty legacy views array.
// ViewKey, Name, and OwnerBotUserID may be absent. A diff view's ViewID can be
// "code" rather than an encoded channel tab ID. DateAdded is a Unix timestamp.
type AgentConversationSessionView struct {
	ViewID         string `json:"view_id"`
	Type           string `json:"type"`
	ViewKey        string `json:"view_key,omitempty"`
	FileID         string `json:"file_id"`
	Name           string `json:"name,omitempty"`
	ContentVersion int    `json:"content_version"`
	DateAdded      int64  `json:"date_added"`
	OwnerBotUserID string `json:"owner_bot_user_id,omitempty"`
}

// AgentConversationListViewsResponse is the response from agents.conversations.listViews.
type AgentConversationListViewsResponse struct {
	SlackResponse
	// Views contains the legacy view list. An empty list does not imply that
	// the channel has no views; also check SessionViews.
	Views []AgentConversationView `json:"views"`
	// SessionViews contains the unified view list observed in live responses.
	// Older responses may omit this field.
	SessionViews []AgentConversationSessionView `json:"session_views"`
}

// ListAgentConversationViews lists all views attached to a code channel.
// Check both SessionViews and the legacy Views field. No pagination is required.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.listViews
func (api *Client) ListAgentConversationViews(channelID string) (*AgentConversationListViewsResponse, error) {
	return api.ListAgentConversationViewsContext(context.Background(), channelID)
}

// ListAgentConversationViewsContext lists all code channel views with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.listViews
func (api *Client) ListAgentConversationViewsContext(ctx context.Context, channelID string) (*AgentConversationListViewsResponse, error) {
	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}
	response := &AgentConversationListViewsResponse{}
	if err := api.postMethod(ctx, "agents.conversations.listViews", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

type removeAgentConversationViewParams struct {
	viewKey string
	viewID  string
}

// RemoveAgentConversationViewOption identifies the view to remove.
// Provide exactly one of ViewKey or ViewID.
type RemoveAgentConversationViewOption func(*removeAgentConversationViewParams)

// RemoveAgentConversationViewOptionViewKey selects a view by its agent-assigned key.
func RemoveAgentConversationViewOptionViewKey(viewKey string) RemoveAgentConversationViewOption {
	return func(params *removeAgentConversationViewParams) {
		params.viewKey = viewKey
	}
}

// RemoveAgentConversationViewOptionViewID selects a view by its encoded channel tab ID.
func RemoveAgentConversationViewOptionViewID(viewID string) RemoveAgentConversationViewOption {
	return func(params *removeAgentConversationViewParams) {
		params.viewID = viewID
	}
}

// AgentConversationRemoveViewResponse is the response from agents.conversations.removeView.
type AgentConversationRemoveViewResponse struct {
	SlackResponse
	ChannelID string `json:"channel_id"`
	ViewID    string `json:"view_id"`
}

// RemoveAgentConversationView removes a view's channel tab and channel properties.
// Slack also deletes the backing content file for html, diff, and block_kit views.
// Provide exactly one of the ViewKey or ViewID options to identify the view.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.removeView
func (api *Client) RemoveAgentConversationView(channelID string, options ...RemoveAgentConversationViewOption) (*AgentConversationRemoveViewResponse, error) {
	return api.RemoveAgentConversationViewContext(context.Background(), channelID, options...)
}

// RemoveAgentConversationViewContext removes a code channel view with a custom
// context. Provide exactly one of the ViewKey or ViewID options.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.removeView
func (api *Client) RemoveAgentConversationViewContext(ctx context.Context, channelID string, options ...RemoveAgentConversationViewOption) (*AgentConversationRemoveViewResponse, error) {
	params := removeAgentConversationViewParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}
	values := url.Values{
		"token":      {api.token},
		"channel_id": {channelID},
	}
	if params.viewKey != "" {
		values.Set("view_key", params.viewKey)
	}
	if params.viewID != "" {
		values.Set("view_id", params.viewID)
	}
	response := &AgentConversationRemoveViewResponse{}
	if err := api.postMethod(ctx, "agents.conversations.removeView", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}
