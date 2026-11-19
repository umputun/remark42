package slack

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// AdminUsersSessionClearSettings clears the session duration and
// desktop_app_browser_quit settings of users.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.session.clearSettings
func (api *Client) AdminUsersSessionClearSettings(userIDs []string) error {
	return api.AdminUsersSessionClearSettingsContext(context.Background(), userIDs)
}

// AdminUsersSessionClearSettingsContext clears session settings with a custom context.
func (api *Client) AdminUsersSessionClearSettingsContext(ctx context.Context, userIDs []string) error {
	values := url.Values{
		"token":    {api.token},
		"user_ids": {strings.Join(userIDs, ",")},
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "admin.users.session.clearSettings", values, response); err != nil {
		return err
	}
	return response.Err()
}

// AdminUserSessionSettings is a user's session settings. Duration is in seconds.
type AdminUserSessionSettings struct {
	UserID                string `json:"user_id"`
	DesktopAppBrowserQuit bool   `json:"desktop_app_browser_quit"`
	Duration              int    `json:"duration"`
}

// AdminUsersSessionGetSettingsResponse represents the response from
// admin.users.session.getSettings.
type AdminUsersSessionGetSettingsResponse struct {
	SlackResponse
	SessionSettings   []AdminUserSessionSettings `json:"session_settings"`
	NoSettingsApplied []string                   `json:"no_settings_applied"`
}

// AdminUsersSessionGetSettings gets the session settings of users. Slack omits
// users without session settings from SessionSettings.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.session.getSettings
func (api *Client) AdminUsersSessionGetSettings(userIDs []string) (*AdminUsersSessionGetSettingsResponse, error) {
	return api.AdminUsersSessionGetSettingsContext(context.Background(), userIDs)
}

// AdminUsersSessionGetSettingsContext gets session settings with a custom context.
func (api *Client) AdminUsersSessionGetSettingsContext(ctx context.Context, userIDs []string) (*AdminUsersSessionGetSettingsResponse, error) {
	values := url.Values{
		"token":    {api.token},
		"user_ids": {strings.Join(userIDs, ",")},
	}

	response := &AdminUsersSessionGetSettingsResponse{}
	if err := api.postMethod(ctx, "admin.users.session.getSettings", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// AdminUserSessionDevice describes the client and network of a session.
type AdminUserSessionDevice struct {
	DeviceHardware     string `json:"device_hardware"`
	OS                 string `json:"os"`
	OSVersion          string `json:"os_version"`
	SlackClientVersion string `json:"slack_client_version"`
	IP                 string `json:"ip"`
}

// AdminUserSession is an active session returned by admin.users.session.list.
// Recent is nil when the session hasn't changed since it was created.
type AdminUserSession struct {
	UserID    string                  `json:"user_id"`
	TeamID    string                  `json:"team_id"`
	SessionID int64                   `json:"session_id"`
	Created   AdminUserSessionDevice  `json:"created"`
	Recent    *AdminUserSessionDevice `json:"recent,omitempty"`
}

// AdminUsersSessionListResponse represents the response from admin.users.session.list.
type AdminUsersSessionListResponse struct {
	SlackResponse
	ActiveSessions []AdminUserSession `json:"active_sessions"`
}

type adminUsersSessionListParams struct {
	teamID string
	userID string
	cursor string
	limit  int
}

// AdminUsersSessionListOption is an option for AdminUsersSessionList.
type AdminUsersSessionListOption func(*adminUsersSessionListParams)

// AdminUsersSessionListOptionUser lists only a user's sessions on a workspace.
// Slack requires the workspace and user IDs together.
func AdminUsersSessionListOptionUser(teamID, userID string) AdminUsersSessionListOption {
	return func(params *adminUsersSessionListParams) {
		params.teamID = teamID
		params.userID = userID
	}
}

// AdminUsersSessionListOptionCursor sets the cursor for pagination.
func AdminUsersSessionListOptionCursor(cursor string) AdminUsersSessionListOption {
	return func(params *adminUsersSessionListParams) {
		params.cursor = cursor
	}
}

// AdminUsersSessionListOptionLimit sets the maximum number of sessions to return.
func AdminUsersSessionListOptionLimit(limit int) AdminUsersSessionListOption {
	return func(params *adminUsersSessionListParams) {
		params.limit = limit
	}
}

// AdminUsersSessionList lists active sessions in the org.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.session.list
func (api *Client) AdminUsersSessionList(options ...AdminUsersSessionListOption) (*AdminUsersSessionListResponse, error) {
	return api.AdminUsersSessionListContext(context.Background(), options...)
}

// AdminUsersSessionListContext lists active sessions with a custom context.
func (api *Client) AdminUsersSessionListContext(ctx context.Context, options ...AdminUsersSessionListOption) (*AdminUsersSessionListResponse, error) {
	params := adminUsersSessionListParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}

	values := url.Values{
		"token": {api.token},
	}
	if params.teamID != "" {
		values.Add("team_id", params.teamID)
	}
	if params.userID != "" {
		values.Add("user_id", params.userID)
	}
	if params.cursor != "" {
		values.Add("cursor", params.cursor)
	}
	if params.limit > 0 {
		values.Add("limit", strconv.Itoa(params.limit))
	}

	response := &AdminUsersSessionListResponse{}
	if err := api.postMethod(ctx, "admin.users.session.list", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

type adminUsersSessionResetParams struct {
	mobileOnly bool
	webOnly    bool
}

func (params adminUsersSessionResetParams) addTo(values url.Values) {
	if params.mobileOnly {
		values.Add("mobile_only", "true")
	}
	if params.webOnly {
		values.Add("web_only", "true")
	}
}

// AdminUsersSessionResetOption is an option for AdminUsersSessionReset.
type AdminUsersSessionResetOption func(*adminUsersSessionResetParams)

// AdminUsersSessionResetOptionMobileOnly resets only mobile sessions.
func AdminUsersSessionResetOptionMobileOnly(mobileOnly bool) AdminUsersSessionResetOption {
	return func(params *adminUsersSessionResetParams) {
		params.mobileOnly = mobileOnly
	}
}

// AdminUsersSessionResetOptionWebOnly resets only web sessions.
func AdminUsersSessionResetOptionWebOnly(webOnly bool) AdminUsersSessionResetOption {
	return func(params *adminUsersSessionResetParams) {
		params.webOnly = webOnly
	}
}

// AdminUsersSessionReset signs a user out of their sessions.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.session.reset
func (api *Client) AdminUsersSessionReset(userID string, options ...AdminUsersSessionResetOption) error {
	return api.AdminUsersSessionResetContext(context.Background(), userID, options...)
}

// AdminUsersSessionResetContext signs a user out with a custom context.
func (api *Client) AdminUsersSessionResetContext(ctx context.Context, userID string, options ...AdminUsersSessionResetOption) error {
	params := adminUsersSessionResetParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}

	values := url.Values{
		"token":   {api.token},
		"user_id": {userID},
	}
	params.addTo(values)

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "admin.users.session.reset", values, response); err != nil {
		return err
	}
	return response.Err()
}

// AdminUsersSessionResetBulkOption is an option for AdminUsersSessionResetBulk.
type AdminUsersSessionResetBulkOption func(*adminUsersSessionResetParams)

// AdminUsersSessionResetBulkOptionMobileOnly resets only mobile sessions.
func AdminUsersSessionResetBulkOptionMobileOnly(mobileOnly bool) AdminUsersSessionResetBulkOption {
	return func(params *adminUsersSessionResetParams) {
		params.mobileOnly = mobileOnly
	}
}

// AdminUsersSessionResetBulkOptionWebOnly resets only web sessions.
func AdminUsersSessionResetBulkOptionWebOnly(webOnly bool) AdminUsersSessionResetBulkOption {
	return func(params *adminUsersSessionResetParams) {
		params.webOnly = webOnly
	}
}

// AdminUsersSessionResetBulk queues a job that signs up to 1,000 users out of
// their sessions. A nil error means the job was queued, not that it finished.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.session.resetBulk
func (api *Client) AdminUsersSessionResetBulk(userIDs []string, options ...AdminUsersSessionResetBulkOption) error {
	return api.AdminUsersSessionResetBulkContext(context.Background(), userIDs, options...)
}

// AdminUsersSessionResetBulkContext queues the session reset with a custom context.
func (api *Client) AdminUsersSessionResetBulkContext(ctx context.Context, userIDs []string, options ...AdminUsersSessionResetBulkOption) error {
	params := adminUsersSessionResetParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}

	values := url.Values{
		"token":    {api.token},
		"user_ids": {strings.Join(userIDs, ",")},
	}
	params.addTo(values)

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "admin.users.session.resetBulk", values, response); err != nil {
		return err
	}
	return response.Err()
}

type adminUsersSessionSetSettingsParams struct {
	duration              *int
	desktopAppBrowserQuit *bool
}

// AdminUsersSessionSetSettingsOption is an option for AdminUsersSessionSetSettings.
type AdminUsersSessionSetSettingsOption func(*adminUsersSessionSetSettingsParams)

// AdminUsersSessionSetSettingsOptionDuration sets how long a session lasts, in
// seconds. Slack accepts 8 hours to 10 years.
func AdminUsersSessionSetSettingsOptionDuration(seconds int) AdminUsersSessionSetSettingsOption {
	return func(params *adminUsersSessionSetSettingsParams) {
		params.duration = &seconds
	}
}

// AdminUsersSessionSetSettingsOptionDesktopAppBrowserQuit sets whether closing the
// desktop app or browser window ends the session.
func AdminUsersSessionSetSettingsOptionDesktopAppBrowserQuit(quit bool) AdminUsersSessionSetSettingsOption {
	return func(params *adminUsersSessionSetSettingsParams) {
		params.desktopAppBrowserQuit = &quit
	}
}

// AdminUsersSessionSetSettings sets session settings for up to 1,000 users.
// Slack requires at least one setting.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.session.setSettings
func (api *Client) AdminUsersSessionSetSettings(userIDs []string, options ...AdminUsersSessionSetSettingsOption) error {
	return api.AdminUsersSessionSetSettingsContext(context.Background(), userIDs, options...)
}

// AdminUsersSessionSetSettingsContext sets session settings with a custom context.
func (api *Client) AdminUsersSessionSetSettingsContext(ctx context.Context, userIDs []string, options ...AdminUsersSessionSetSettingsOption) error {
	params := adminUsersSessionSetSettingsParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}

	values := url.Values{
		"token":    {api.token},
		"user_ids": {strings.Join(userIDs, ",")},
	}
	if params.duration != nil {
		values.Add("duration", strconv.Itoa(*params.duration))
	}
	if params.desktopAppBrowserQuit != nil {
		values.Add("desktop_app_browser_quit", strconv.FormatBool(*params.desktopAppBrowserQuit))
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "admin.users.session.setSettings", values, response); err != nil {
		return err
	}
	return response.Err()
}
