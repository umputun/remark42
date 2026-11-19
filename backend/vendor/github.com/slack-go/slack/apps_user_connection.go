package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// UserConnectionPrimaryProvider describes the provider shown in an auth modal.
// Name, Description, ConnectedIdentity, and ConnectedContext use plain_text objects.
// IsConnected is always sent, including when false.
//
// EXPERIMENTAL: this provider schema follows the official Node SDK and may change.
type UserConnectionPrimaryProvider struct {
	Value             string                `json:"value"`
	Name              *TextBlockObject      `json:"name"`
	IsConnected       bool                  `json:"is_connected"`
	Description       *TextBlockObject      `json:"description,omitempty"`
	IconURL           string                `json:"icon_url,omitempty"`
	AuthURL           string                `json:"auth_url,omitempty"`
	DisconnectURL     string                `json:"disconnect_url,omitempty"`
	ConnectedIdentity *TextBlockObject      `json:"connected_identity,omitempty"`
	ConnectedContext  *TextBlockObject      `json:"connected_context,omitempty"`
	Accessory         *OverflowBlockElement `json:"accessory,omitempty"`
}

type requestUserConnectionAuthParams struct{ triggerID, userID string }

// RequestUserConnectionAuthOption configures the recipient of an auth modal.
type RequestUserConnectionAuthOption func(*requestUserConnectionAuthParams)

// RequestUserConnectionAuthOptionTriggerID opens a modal using an interaction trigger.
// Slack ignores UserID when TriggerID is present.
func RequestUserConnectionAuthOptionTriggerID(triggerID string) RequestUserConnectionAuthOption {
	return func(params *requestUserConnectionAuthParams) { params.triggerID = triggerID }
}

// RequestUserConnectionAuthOptionUserID updates the specified user's auth modal.
func RequestUserConnectionAuthOptionUserID(userID string) RequestUserConnectionAuthOption {
	return func(params *requestUserConnectionAuthParams) { params.userID = userID }
}

// RequestUserConnectionAuth opens or updates a standardized authentication modal.
// Provide a trigger ID to open a modal or a user ID to update one.
//
// EXPERIMENTAL: this API follows the official Node SDK and may change.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.user.connection.requestAuth
func (api *Client) RequestUserConnectionAuth(provider UserConnectionPrimaryProvider, options ...RequestUserConnectionAuthOption) error {
	return api.RequestUserConnectionAuthContext(context.Background(), provider, options...)
}

// RequestUserConnectionAuthContext opens or updates an auth modal with a custom context.
//
// EXPERIMENTAL: this API follows the official Node SDK and may change.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.user.connection.requestAuth
func (api *Client) RequestUserConnectionAuthContext(ctx context.Context, provider UserConnectionPrimaryProvider, options ...RequestUserConnectionAuthOption) error {
	params := requestUserConnectionAuthParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}
	if params.triggerID == "" && params.userID == "" {
		return fmt.Errorf("one of trigger_id or user_id is required")
	}
	encoded, err := json.Marshal(provider)
	if err != nil {
		return err
	}
	values := url.Values{"token": {api.token}, "primary_provider": {string(encoded)}}
	if params.triggerID != "" {
		values.Set("trigger_id", params.triggerID)
	}
	if params.userID != "" {
		values.Set("user_id", params.userID)
	}
	response := &SlackResponse{}
	if err := api.postMethod(ctx, "apps.user.connection.requestAuth", values, response); err != nil {
		return err
	}
	return response.Err()
}

// UserConnectionFailure reports a user whose connection status could not be updated.
type UserConnectionFailure struct {
	UserID string `json:"user_id"`
	Error  string `json:"error"`
}

// UserConnectionBulkUpdateResponse includes changed users and per-user failures.
// Check Failures even when the request succeeds.
type UserConnectionBulkUpdateResponse struct {
	SlackResponse
	ChangedUserIDs []string                `json:"changed_user_ids"`
	Failures       []UserConnectionFailure `json:"failures"`
}

// BulkUpdateUserConnections backfills connection status for up to 1000 users.
// At least one of the connected or disconnected user lists must be nonempty.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.user.connection.bulkUpdate
func (api *Client) BulkUpdateUserConnections(connectedUserIDs, disconnectedUserIDs []string) (*UserConnectionBulkUpdateResponse, error) {
	return api.BulkUpdateUserConnectionsContext(context.Background(), connectedUserIDs, disconnectedUserIDs)
}

// BulkUpdateUserConnectionsContext backfills user connection status with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.user.connection.bulkUpdate
func (api *Client) BulkUpdateUserConnectionsContext(ctx context.Context, connectedUserIDs, disconnectedUserIDs []string) (*UserConnectionBulkUpdateResponse, error) {
	count := len(connectedUserIDs) + len(disconnectedUserIDs)
	if count == 0 || count > 1000 {
		return nil, fmt.Errorf("between 1 and 1000 user IDs are required")
	}
	values := url.Values{"token": {api.token}}
	if len(connectedUserIDs) > 0 {
		values.Set("connected_user_ids", strings.Join(connectedUserIDs, ","))
	}
	if len(disconnectedUserIDs) > 0 {
		values.Set("disconnected_user_ids", strings.Join(disconnectedUserIDs, ","))
	}
	response := &UserConnectionBulkUpdateResponse{}
	if err := api.postMethod(ctx, "apps.user.connection.bulkUpdate", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}
