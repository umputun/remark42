package slack

import (
	"context"
	"net/url"
	"strconv"
)

type adminUsersListParams struct {
	teamID                           string
	cursor                           string
	limit                            int
	isActive                         *bool
	includeDeactivatedUserWorkspaces bool
	onlyGuests                       bool
	includeAdmins                    bool
	includeOwners                    bool
}

// AdminUsersListOption is an option for AdminUsersList.
type AdminUsersListOption func(*adminUsersListParams)

// AdminUsersListOptionTeamID filters results to the specified workspace.
func AdminUsersListOptionTeamID(teamID string) AdminUsersListOption {
	return func(params *adminUsersListParams) {
		params.teamID = teamID
	}
}

// AdminUsersListOptionCursor sets the cursor for pagination.
func AdminUsersListOptionCursor(cursor string) AdminUsersListOption {
	return func(params *adminUsersListParams) {
		params.cursor = cursor
	}
}

// AdminUsersListOptionLimit sets the maximum number of users to return.
func AdminUsersListOptionLimit(limit int) AdminUsersListOption {
	return func(params *adminUsersListParams) {
		params.limit = limit
	}
}

// AdminUsersListOptionIsActive returns only active users (true) or only deactivated
// users (false). Slack returns only active users when it is not set.
func AdminUsersListOptionIsActive(isActive bool) AdminUsersListOption {
	return func(params *adminUsersListParams) {
		params.isActive = &isActive
	}
}

// AdminUsersListOptionIncludeDeactivatedUserWorkspaces includes workspaces where a
// user is deactivated in each user's workspaces.
func AdminUsersListOptionIncludeDeactivatedUserWorkspaces(include bool) AdminUsersListOption {
	return func(params *adminUsersListParams) {
		params.includeDeactivatedUserWorkspaces = include
	}
}

// AdminUsersListOptionOnlyGuests returns only guests and their expiration dates.
func AdminUsersListOptionOnlyGuests(onlyGuests bool) AdminUsersListOption {
	return func(params *adminUsersListParams) {
		params.onlyGuests = onlyGuests
	}
}

// AdminUsersListOptionIncludeAdmins returns only admins, or admins and owners when
// combined with AdminUsersListOptionIncludeOwners.
func AdminUsersListOptionIncludeAdmins(includeAdmins bool) AdminUsersListOption {
	return func(params *adminUsersListParams) {
		params.includeAdmins = includeAdmins
	}
}

// AdminUsersListOptionIncludeOwners returns only owners.
func AdminUsersListOptionIncludeOwners(includeOwners bool) AdminUsersListOption {
	return func(params *adminUsersListParams) {
		params.includeOwners = includeOwners
	}
}

// AdminUser represents a user returned by admin.users.list.
type AdminUser struct {
	ID                   string   `json:"id"`
	Email                string   `json:"email"`
	IsAdmin              bool     `json:"is_admin"`
	IsOwner              bool     `json:"is_owner"`
	IsPrimaryOwner       bool     `json:"is_primary_owner"`
	IsRestricted         bool     `json:"is_restricted"`
	IsUltraRestricted    bool     `json:"is_ultra_restricted"`
	IsBot                bool     `json:"is_bot"`
	Username             string   `json:"username"`
	FullName             string   `json:"full_name"`
	IsActive             bool     `json:"is_active"`
	DateCreated          int64    `json:"date_created"`
	DeactivatedTimestamp int64    `json:"deactivated_ts"`
	ExpirationTimestamp  int64    `json:"expiration_ts"`
	LastActiveTimestamp  int64    `json:"last_active_ts"`
	Workspaces           []string `json:"workspaces"`
	Roles                []string `json:"roles"`
	Has2FA               bool     `json:"has_2fa"`
	HasSSO               bool     `json:"has_sso"`
}

// AdminUsersListResponse represents the response from admin.users.list.
type AdminUsersListResponse struct {
	SlackResponse
	Users []AdminUser `json:"users"`
}

// AdminUsersList lists users on a workspace or Enterprise organization.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.list
func (api *Client) AdminUsersList(ctx context.Context, options ...AdminUsersListOption) (*AdminUsersListResponse, error) {
	params := adminUsersListParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token": {api.token},
	}

	if params.teamID != "" {
		values.Add("team_id", params.teamID)
	}
	if params.cursor != "" {
		values.Add("cursor", params.cursor)
	}
	if params.limit > 0 {
		values.Add("limit", strconv.Itoa(params.limit))
	}
	if params.isActive != nil {
		values.Add("is_active", strconv.FormatBool(*params.isActive))
	}
	if params.includeDeactivatedUserWorkspaces {
		values.Add("include_deactivated_user_workspaces", "true")
	}
	if params.onlyGuests {
		values.Add("only_guests", "true")
	}
	if params.includeAdmins {
		values.Add("include_admins", "true")
	}
	if params.includeOwners {
		values.Add("include_owners", "true")
	}

	response := &AdminUsersListResponse{}
	err := api.postMethod(ctx, "admin.users.list", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}
