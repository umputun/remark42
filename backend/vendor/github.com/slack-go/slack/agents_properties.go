package slack

// AgentSessionOriginLink identifies the message that started an agent session.
type AgentSessionOriginLink struct {
	ChannelID string `json:"channel_id"`
	Timestamp string `json:"ts"`
}

// AgentSessionProperties contains session metadata returned in channel.properties.
//
// Slack API docs: https://docs.slack.dev/ai/slack-code/code-channel-properties/
type AgentSessionProperties struct {
	Status                 string                  `json:"status,omitempty"`
	Title                  string                  `json:"title,omitempty"`
	AgentBotUserIDs        []string                `json:"agent_bot_user_ids,omitempty"`
	EncodedAgentBotUserIDs []string                `json:"encoded_agent_bot_user_ids,omitempty"`
	OriginLink             *AgentSessionOriginLink `json:"origin_link,omitempty"`
	// EncodedInitiatorUserID has been observed in live conversations.info
	// responses and is not currently described in the reference or Node types.
	EncodedInitiatorUserID string `json:"encoded_initiator_user_id,omitempty"`
}

// AgentContextBarItemInfo is a received context-bar item with the owning bot's
// user ID. BotUserID is populated by Slack and is not a writable item property.
type AgentContextBarItemInfo struct {
	AgentContextBarItem
	BotUserID string `json:"bot_user_id,omitempty"`
}

// AgentCodeChannelView is a view entry in channel.properties.code_channel.
// CanvasID and DateUpdated are additional fields observed in live channel
// properties. Canvas views may provide CanvasID instead of FileID.
type AgentCodeChannelView struct {
	AgentConversationSessionView
	CanvasID    string `json:"canvas_id,omitempty"`
	DateUpdated int64  `json:"date_updated,omitempty"`
}

// AgentCodeChannelInfo contains code-channel metadata returned in
// channel.properties. AgentSessionViews is the unified list maintained by Slack.
//
// Slack API docs: https://docs.slack.dev/ai/slack-code/code-channel-properties/
type AgentCodeChannelInfo struct {
	ContextBarItems   []AgentContextBarItemInfo `json:"context_bar_items,omitempty"`
	SummaryMessage    *AgentSummaryMessage      `json:"summary_message,omitempty"`
	AgentSessionViews []AgentCodeChannelView    `json:"agent_session_views,omitempty"`
}
