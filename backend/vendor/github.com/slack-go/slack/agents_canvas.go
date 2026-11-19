package slack

import (
	"context"
	"net/url"
	"strconv"
)

// AgentCanvasCommentReply is a reply to a comment thread on an agent canvas.
type AgentCanvasCommentReply struct {
	Timestamp string `json:"ts"`
	UserID    string `json:"user_id"`
	Text      string `json:"text"`
}

// AgentCanvasComment is a comment thread anchored to quoted canvas text.
// HasMoreReplies indicates that older replies were omitted from the response.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.getCanvas
type AgentCanvasComment struct {
	ID             string                    `json:"id"`
	Timestamp      string                    `json:"ts"`
	UserID         string                    `json:"user_id"`
	Text           string                    `json:"text"`
	QuotedText     string                    `json:"quoted_text"`
	IsResolved     bool                      `json:"is_resolved"`
	Replies        []AgentCanvasCommentReply `json:"replies"`
	HasMoreReplies bool                      `json:"has_more_replies"`
}

// AgentConversationGetCanvasResponse is the response from agents.conversations.getCanvas.
// HasMoreComments indicates that older comment threads were omitted.
type AgentConversationGetCanvasResponse struct {
	SlackResponse
	CanvasID        string               `json:"canvas_id"`
	Title           string               `json:"title"`
	Content         string               `json:"content"`
	Comments        []AgentCanvasComment `json:"comments"`
	HasMoreComments bool                 `json:"has_more_comments"`
}

type getAgentConversationCanvasParams struct {
	contentFormat   string
	includeResolved *bool
}

// GetAgentConversationCanvasOption configures canvas content and comment retrieval.
type GetAgentConversationCanvasOption func(*getAgentConversationCanvasParams)

// GetAgentConversationCanvasOptionContentFormat sets the canvas rendering format.
// Slack currently accepts markdown, which is also the default.
func GetAgentConversationCanvasOptionContentFormat(format string) GetAgentConversationCanvasOption {
	return func(params *getAgentConversationCanvasParams) {
		params.contentFormat = format
	}
}

// GetAgentConversationCanvasOptionIncludeResolved controls whether resolved
// comment threads are included. Slack defaults to false when this is omitted.
func GetAgentConversationCanvasOptionIncludeResolved(includeResolved bool) GetAgentConversationCanvasOption {
	return func(params *getAgentConversationCanvasParams) {
		params.includeResolved = &includeResolved
	}
}

// GetAgentConversationCanvas fetches an attached canvas's content and comment threads.
// Slack returns up to 100 recent threads and 100 recent replies per thread,
// ordered oldest first, and always includes each thread's top-level comment.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.getCanvas
func (api *Client) GetAgentConversationCanvas(channelID, canvasID string, options ...GetAgentConversationCanvasOption) (*AgentConversationGetCanvasResponse, error) {
	return api.GetAgentConversationCanvasContext(context.Background(), channelID, canvasID, options...)
}

// GetAgentConversationCanvasContext fetches an attached canvas's content and
// comment threads with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.getCanvas
func (api *Client) GetAgentConversationCanvasContext(ctx context.Context, channelID, canvasID string, options ...GetAgentConversationCanvasOption) (*AgentConversationGetCanvasResponse, error) {
	params := getAgentConversationCanvasParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}
	values := url.Values{
		"token":     {api.token},
		"channel":   {channelID},
		"canvas_id": {canvasID},
	}
	if params.contentFormat != "" {
		values.Set("content_format", params.contentFormat)
	}
	if params.includeResolved != nil {
		values.Set("include_resolved", strconv.FormatBool(*params.includeResolved))
	}
	response := &AgentConversationGetCanvasResponse{}
	if err := api.postMethod(ctx, "agents.conversations.getCanvas", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// AgentConversationSetCanvasContentResponse is the response from
// agents.conversations.setCanvasContent. SectionsChangedCount is zero for a no-op.
type AgentConversationSetCanvasContentResponse struct {
	SlackResponse
	CanvasID             string `json:"canvas_id"`
	SectionsChangedCount int    `json:"sections_changed_count"`
}

// SetAgentConversationCanvasContent sends the full new markdown for an attached
// canvas. Slack applies section-level edits, preserving section IDs and comment
// anchors on sections that did not change.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.setCanvasContent
func (api *Client) SetAgentConversationCanvasContent(channelID, canvasID, content string) (*AgentConversationSetCanvasContentResponse, error) {
	return api.SetAgentConversationCanvasContentContext(context.Background(), channelID, canvasID, content)
}

// SetAgentConversationCanvasContentContext sends the full new canvas markdown
// with a custom context. Content is sent even when it is an empty string.
//
// Slack API docs: https://docs.slack.dev/reference/methods/agents.conversations.setCanvasContent
func (api *Client) SetAgentConversationCanvasContentContext(ctx context.Context, channelID, canvasID, content string) (*AgentConversationSetCanvasContentResponse, error) {
	values := url.Values{
		"token":     {api.token},
		"channel":   {channelID},
		"canvas_id": {canvasID},
		"content":   {content},
	}
	response := &AgentConversationSetCanvasContentResponse{}
	if err := api.postMethod(ctx, "agents.conversations.setCanvasContent", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}
