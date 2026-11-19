package slack

import (
	"context"
	"net/url"
	"strconv"
)

type adminAppsListParams struct {
	limit        int
	cursor       string
	teamID       string
	enterpriseID string
	certified    *bool
}

// AdminAppsListOption is an option for the admin.apps list methods.
type AdminAppsListOption func(*adminAppsListParams)

// AdminAppsListOptionLimit sets the maximum number of apps to return.
func AdminAppsListOptionLimit(limit int) AdminAppsListOption {
	return func(params *adminAppsListParams) {
		params.limit = limit
	}
}

// AdminAppsListOptionCursor sets the cursor for pagination.
func AdminAppsListOptionCursor(cursor string) AdminAppsListOption {
	return func(params *adminAppsListParams) {
		params.cursor = cursor
	}
}

// AdminAppsListOptionTeamID limits results to a workspace.
func AdminAppsListOptionTeamID(teamID string) AdminAppsListOption {
	return func(params *adminAppsListParams) {
		params.teamID = teamID
	}
}

// AdminAppsListOptionEnterpriseID limits results to an Enterprise organization.
func AdminAppsListOptionEnterpriseID(enterpriseID string) AdminAppsListOption {
	return func(params *adminAppsListParams) {
		params.enterpriseID = enterpriseID
	}
}

// AdminAppsListOptionCertified filters results by Slack Marketplace certification.
func AdminAppsListOptionCertified(certified bool) AdminAppsListOption {
	return func(params *adminAppsListParams) {
		params.certified = &certified
	}
}

// AdminAppIcons contains an app's icon URLs at each available size.
type AdminAppIcons struct {
	Image32       string `json:"image_32"`
	Image36       string `json:"image_36"`
	Image48       string `json:"image_48"`
	Image64       string `json:"image_64"`
	Image72       string `json:"image_72"`
	Image96       string `json:"image_96"`
	Image128      string `json:"image_128"`
	Image192      string `json:"image_192"`
	Image512      string `json:"image_512"`
	Image1024     string `json:"image_1024"`
	ImageOriginal string `json:"image_original"`
}

// AdminApp contains app metadata returned by the admin.apps list methods.
type AdminApp struct {
	ID                     string        `json:"id"`
	Name                   string        `json:"name"`
	Description            string        `json:"description"`
	HelpURL                string        `json:"help_url"`
	PrivacyPolicyURL       string        `json:"privacy_policy_url"`
	AppHomepageURL         string        `json:"app_homepage_url"`
	AppDirectoryURL        string        `json:"app_directory_url"`
	IsAppDirectoryApproved bool          `json:"is_app_directory_approved"`
	IsGranularBotApp       bool          `json:"is_granular_bot_app"`
	IsInternal             bool          `json:"is_internal"`
	DeveloperType          string        `json:"developer_type"`
	SocketModeEnabled      bool          `json:"socket_mode_enabled"`
	Icons                  AdminAppIcons `json:"icons"`
	AdditionalInfo         string        `json:"additional_info"`
}

// AdminAppScope describes one OAuth scope requested by an app.
type AdminAppScope struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsSensitive bool   `json:"is_sensitive"`
	TokenType   string `json:"token_type"`
}

// AdminAppResolutionActor identifies who last approved or restricted an app.
type AdminAppResolutionActor struct {
	ActorID   string `json:"actor_id"`
	ActorType string `json:"actor_type"`
}

// AdminAppListItem is an approved or restricted app entry.
type AdminAppListItem struct {
	App            AdminApp                `json:"app"`
	Scopes         []AdminAppScope         `json:"scopes"`
	DateUpdated    int64                   `json:"date_updated"`
	LastResolvedBy AdminAppResolutionActor `json:"last_resolved_by"`
}

// AdminAppsApprovedListResponse represents the response from admin.apps.approved.list.
type AdminAppsApprovedListResponse struct {
	SlackResponse
	ApprovedApps []AdminAppListItem `json:"approved_apps"`
}

// AdminAppsRestrictedListResponse represents the response from admin.apps.restricted.list.
type AdminAppsRestrictedListResponse struct {
	SlackResponse
	RestrictedApps []AdminAppListItem `json:"restricted_apps"`
}

func adminAppsListValues(token string, params adminAppsListParams) url.Values {
	values := url.Values{"token": {token}}
	if params.limit > 0 {
		values.Set("limit", strconv.Itoa(params.limit))
	}
	if params.cursor != "" {
		values.Set("cursor", params.cursor)
	}
	if params.teamID != "" {
		values.Set("team_id", params.teamID)
	}
	if params.enterpriseID != "" {
		values.Set("enterprise_id", params.enterpriseID)
	}
	if params.certified != nil {
		values.Set("certified", strconv.FormatBool(*params.certified))
	}
	return values
}

// AdminAppsApprovedList lists apps approved for installation on an Enterprise organization or workspace.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.apps.approved.list
func (api *Client) AdminAppsApprovedList(options ...AdminAppsListOption) (*AdminAppsApprovedListResponse, error) {
	return api.AdminAppsApprovedListContext(context.Background(), options...)
}

// AdminAppsApprovedListContext lists approved apps with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.apps.approved.list
func (api *Client) AdminAppsApprovedListContext(ctx context.Context, options ...AdminAppsListOption) (*AdminAppsApprovedListResponse, error) {
	params := adminAppsListParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	response := &AdminAppsApprovedListResponse{}
	if err := api.postMethod(ctx, "admin.apps.approved.list", adminAppsListValues(api.token, params), response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// AdminAppsRestrictedList lists apps restricted from installation on an Enterprise organization or workspace.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.apps.restricted.list
func (api *Client) AdminAppsRestrictedList(options ...AdminAppsListOption) (*AdminAppsRestrictedListResponse, error) {
	return api.AdminAppsRestrictedListContext(context.Background(), options...)
}

// AdminAppsRestrictedListContext lists restricted apps with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.apps.restricted.list
func (api *Client) AdminAppsRestrictedListContext(ctx context.Context, options ...AdminAppsListOption) (*AdminAppsRestrictedListResponse, error) {
	params := adminAppsListParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	response := &AdminAppsRestrictedListResponse{}
	if err := api.postMethod(ctx, "admin.apps.restricted.list", adminAppsListValues(api.token, params), response); err != nil {
		return nil, err
	}
	return response, response.Err()
}
