package slack

import (
	"context"
	"net/url"
	"strconv"
	"strings"
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

// AdminUsersSetAdmin makes a regular user or owner an admin. teamID may be a
// workspace or org ID. With an org ID, Slack also makes the user a Workspace
// Owner on every workspace in the org they belong to.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.setAdmin
func (api *Client) AdminUsersSetAdmin(teamID, userID string) error {
	return api.AdminUsersSetAdminContext(context.Background(), teamID, userID)
}

// AdminUsersSetAdminContext makes a user an admin with a custom context.
func (api *Client) AdminUsersSetAdminContext(ctx context.Context, teamID, userID string) error {
	return api.adminUsersSetAccountType(ctx, "admin.users.setAdmin", teamID, userID)
}

// AdminUserExpiration is a guest returned by admin.users.getExpiration.
type AdminUserExpiration struct {
	ID                  string `json:"id"`
	Email               string `json:"email"`
	IsRestricted        bool   `json:"is_restricted"`
	IsUltraRestricted   bool   `json:"is_ultra_restricted"`
	ExpirationTimestamp int64  `json:"expiration_ts"`
}

// AdminUsersGetExpirationResponse represents the response from admin.users.getExpiration.
type AdminUsersGetExpirationResponse struct {
	SlackResponse
	User AdminUserExpiration `json:"user"`
}

type adminUsersGetExpirationParams struct {
	targetTeam string
}

// AdminUsersGetExpirationOption is an option for AdminUsersGetExpiration.
type AdminUsersGetExpirationOption func(*adminUsersGetExpirationParams)

// AdminUsersGetExpirationOptionTargetTeam returns the guest's expiration on a
// workspace in the org. It requires an org-level token.
func AdminUsersGetExpirationOptionTargetTeam(teamID string) AdminUsersGetExpirationOption {
	return func(params *adminUsersGetExpirationParams) {
		params.targetTeam = teamID
	}
}

// AdminUsersGetExpiration gets when a guest account is disabled, as Unix seconds.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.getExpiration
func (api *Client) AdminUsersGetExpiration(userID string, options ...AdminUsersGetExpirationOption) (*AdminUsersGetExpirationResponse, error) {
	return api.AdminUsersGetExpirationContext(context.Background(), userID, options...)
}

// AdminUsersGetExpirationContext gets a guest's expiration with a custom context.
func (api *Client) AdminUsersGetExpirationContext(ctx context.Context, userID string, options ...AdminUsersGetExpirationOption) (*AdminUsersGetExpirationResponse, error) {
	params := adminUsersGetExpirationParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}

	values := url.Values{
		"token":   {api.token},
		"user_id": {userID},
	}
	if params.targetTeam != "" {
		values.Add("target_team", params.targetTeam)
	}

	response := &AdminUsersGetExpirationResponse{}
	if err := api.postMethod(ctx, "admin.users.getExpiration", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

type adminUsersSetExpirationParams struct {
	teamID string
}

// AdminUsersSetExpirationOption is an option for AdminUsersSetExpiration.
type AdminUsersSetExpirationOption func(*adminUsersSetExpirationParams)

// AdminUsersSetExpirationOptionTeamID sets the guest's workspace. It is required
// with an org-level token; without it, Slack returns
// team_id_required_for_enterprise.
func AdminUsersSetExpirationOptionTeamID(teamID string) AdminUsersSetExpirationOption {
	return func(params *adminUsersSetExpirationParams) {
		params.teamID = teamID
	}
}

// AdminUsersSetExpiration sets when a guest account is disabled, as Unix seconds.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.setExpiration
func (api *Client) AdminUsersSetExpiration(userID string, expirationTS int64, options ...AdminUsersSetExpirationOption) error {
	return api.AdminUsersSetExpirationContext(context.Background(), userID, expirationTS, options...)
}

// AdminUsersSetExpirationContext sets a guest's expiration with a custom context.
func (api *Client) AdminUsersSetExpirationContext(ctx context.Context, userID string, expirationTS int64, options ...AdminUsersSetExpirationOption) error {
	params := adminUsersSetExpirationParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}

	values := url.Values{
		"token":         {api.token},
		"user_id":       {userID},
		"expiration_ts": {strconv.FormatInt(expirationTS, 10)},
	}
	if params.teamID != "" {
		values.Add("team_id", params.teamID)
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "admin.users.setExpiration", values, response); err != nil {
		return err
	}
	return response.Err()
}

// AdminUsersSetOwner makes a regular user or admin an owner. teamID may be a
// workspace or org ID. With an org ID, Slack also makes the user a Workspace
// Owner on every workspace in the org they belong to.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.setOwner
func (api *Client) AdminUsersSetOwner(teamID, userID string) error {
	return api.AdminUsersSetOwnerContext(context.Background(), teamID, userID)
}

// AdminUsersSetOwnerContext makes a user an owner with a custom context.
func (api *Client) AdminUsersSetOwnerContext(ctx context.Context, teamID, userID string) error {
	return api.adminUsersSetAccountType(ctx, "admin.users.setOwner", teamID, userID)
}

// AdminUsersSetRegular makes a guest, admin, or owner a regular user. teamID may
// be a workspace or org ID. With an org ID, Slack also removes every Workspace
// Owner role the user holds in the org; restore any with AdminUsersSetOwner and a
// workspace ID. On failed_to_set_regular_on_workspaces, the org-level change has
// been applied and the call can be retried.
//
// Unlike SetRegular, this calls the documented Admin API method.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.setRegular
func (api *Client) AdminUsersSetRegular(teamID, userID string) error {
	return api.AdminUsersSetRegularContext(context.Background(), teamID, userID)
}

// AdminUsersSetRegularContext makes a user a regular user with a custom context.
func (api *Client) AdminUsersSetRegularContext(ctx context.Context, teamID, userID string) error {
	return api.adminUsersSetAccountType(ctx, "admin.users.setRegular", teamID, userID)
}

func (api *Client) adminUsersSetAccountType(ctx context.Context, method, teamID, userID string) error {
	values := url.Values{
		"token":   {api.token},
		"team_id": {teamID},
		"user_id": {userID},
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, method, values, response); err != nil {
		return err
	}
	return response.Err()
}

type adminUsersAssignParams struct {
	channelIDs        []string
	isRestricted      *bool
	isUltraRestricted *bool
}

// AdminUsersAssignOption is an option for AdminUsersAssign.
type AdminUsersAssignOption func(*adminUsersAssignParams)

// AdminUsersAssignOptionChannelIDs adds the user to channels in the workspace.
func AdminUsersAssignOptionChannelIDs(channelIDs ...string) AdminUsersAssignOption {
	return func(params *adminUsersAssignParams) {
		params.channelIDs = append([]string{}, channelIDs...)
	}
}

// AdminUsersAssignOptionIsRestricted sets whether the user is added as a
// multi-channel guest. Slack documents no default, so false is sent too.
func AdminUsersAssignOptionIsRestricted(isRestricted bool) AdminUsersAssignOption {
	return func(params *adminUsersAssignParams) {
		params.isRestricted = &isRestricted
	}
}

// AdminUsersAssignOptionIsUltraRestricted sets whether the user is added as a
// single-channel guest. Slack documents no default, so false is sent too.
func AdminUsersAssignOptionIsUltraRestricted(isUltraRestricted bool) AdminUsersAssignOption {
	return func(params *adminUsersAssignParams) {
		params.isUltraRestricted = &isUltraRestricted
	}
}

// AdminUsersAssign adds an org member to a workspace, or reinstates them if they
// were removed or left.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.assign
func (api *Client) AdminUsersAssign(teamID, userID string, options ...AdminUsersAssignOption) error {
	return api.AdminUsersAssignContext(context.Background(), teamID, userID, options...)
}

// AdminUsersAssignContext adds a user to a workspace with a custom context.
func (api *Client) AdminUsersAssignContext(ctx context.Context, teamID, userID string, options ...AdminUsersAssignOption) error {
	params := adminUsersAssignParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}

	values := url.Values{
		"token":   {api.token},
		"team_id": {teamID},
		"user_id": {userID},
	}
	if len(params.channelIDs) > 0 {
		values.Add("channel_ids", strings.Join(params.channelIDs, ","))
	}
	if params.isRestricted != nil {
		values.Add("is_restricted", strconv.FormatBool(*params.isRestricted))
	}
	if params.isUltraRestricted != nil {
		values.Add("is_ultra_restricted", strconv.FormatBool(*params.isUltraRestricted))
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "admin.users.assign", values, response); err != nil {
		return err
	}
	return response.Err()
}

type adminUsersInviteParams struct {
	customMessage              string
	realName                   string
	resend                     bool
	isRestricted               bool
	isUltraRestricted          bool
	guestExpirationTS          string
	emailPasswordPolicyEnabled bool
}

// AdminUsersInviteOption is an option for AdminUsersInvite.
type AdminUsersInviteOption func(*adminUsersInviteParams)

// AdminUsersInviteOptionCustomMessage adds a message to the invite email.
func AdminUsersInviteOptionCustomMessage(message string) AdminUsersInviteOption {
	return func(params *adminUsersInviteParams) {
		params.customMessage = message
	}
}

// AdminUsersInviteOptionRealName sets the invited person's full name.
func AdminUsersInviteOptionRealName(realName string) AdminUsersInviteOption {
	return func(params *adminUsersInviteParams) {
		params.realName = realName
	}
}

// AdminUsersInviteOptionResend allows the invite to be resent from the Slack UI.
func AdminUsersInviteOptionResend(resend bool) AdminUsersInviteOption {
	return func(params *adminUsersInviteParams) {
		params.resend = resend
	}
}

// AdminUsersInviteOptionIsRestricted invites the person as a multi-channel guest.
func AdminUsersInviteOptionIsRestricted(isRestricted bool) AdminUsersInviteOption {
	return func(params *adminUsersInviteParams) {
		params.isRestricted = isRestricted
	}
}

// AdminUsersInviteOptionIsUltraRestricted invites the person as a single-channel guest.
func AdminUsersInviteOptionIsUltraRestricted(isUltraRestricted bool) AdminUsersInviteOption {
	return func(params *adminUsersInviteParams) {
		params.isUltraRestricted = isUltraRestricted
	}
}

// AdminUsersInviteOptionGuestExpirationTS sets when an invited guest's account is
// disabled, as a Unix timestamp.
func AdminUsersInviteOptionGuestExpirationTS(expirationTS string) AdminUsersInviteOption {
	return func(params *adminUsersInviteParams) {
		params.guestExpirationTS = expirationTS
	}
}

// AdminUsersInviteOptionEmailPasswordPolicyEnabled allows the invited person to
// sign in with an email and password.
func AdminUsersInviteOptionEmailPasswordPolicyEnabled(enabled bool) AdminUsersInviteOption {
	return func(params *adminUsersInviteParams) {
		params.emailPasswordPolicyEnabled = enabled
	}
}

// AdminUsersInvite invites a person to a workspace and at least one channel.
//
// Unlike InviteToTeam, InviteGuest, and InviteRestricted, this calls the
// documented Admin API method.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.invite
func (api *Client) AdminUsersInvite(teamID, email string, channelIDs []string, options ...AdminUsersInviteOption) error {
	return api.AdminUsersInviteContext(context.Background(), teamID, email, channelIDs, options...)
}

// AdminUsersInviteContext invites a person to a workspace with a custom context.
func (api *Client) AdminUsersInviteContext(ctx context.Context, teamID, email string, channelIDs []string, options ...AdminUsersInviteOption) error {
	params := adminUsersInviteParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}

	values := url.Values{
		"token":       {api.token},
		"team_id":     {teamID},
		"email":       {email},
		"channel_ids": {strings.Join(channelIDs, ",")},
	}
	if params.customMessage != "" {
		values.Add("custom_message", params.customMessage)
	}
	if params.realName != "" {
		values.Add("real_name", params.realName)
	}
	if params.resend {
		values.Add("resend", "true")
	}
	if params.isRestricted {
		values.Add("is_restricted", "true")
	}
	if params.isUltraRestricted {
		values.Add("is_ultra_restricted", "true")
	}
	if params.guestExpirationTS != "" {
		values.Add("guest_expiration_ts", params.guestExpirationTS)
	}
	if params.emailPasswordPolicyEnabled {
		values.Add("email_password_policy_enabled", "true")
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "admin.users.invite", values, response); err != nil {
		return err
	}
	return response.Err()
}

// AdminUsersRemove removes a user from a workspace.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.remove
func (api *Client) AdminUsersRemove(teamID, userID string) error {
	return api.AdminUsersRemoveContext(context.Background(), teamID, userID)
}

// AdminUsersRemoveContext removes a user from a workspace with a custom context.
func (api *Client) AdminUsersRemoveContext(ctx context.Context, teamID, userID string) error {
	values := url.Values{
		"token":   {api.token},
		"team_id": {teamID},
		"user_id": {userID},
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "admin.users.remove", values, response); err != nil {
		return err
	}
	return response.Err()
}

type adminUsersUnsupportedVersionsExportParams struct {
	dateEndOfSupport    *int64
	dateSessionsStarted *int64
}

// AdminUsersUnsupportedVersionsExportOption is an option for
// AdminUsersUnsupportedVersionsExport.
type AdminUsersUnsupportedVersionsExportOption func(*adminUsersUnsupportedVersionsExportParams)

// AdminUsersUnsupportedVersionsExportOptionDateEndOfSupport limits the export to
// the end of support cycle at this Unix timestamp. Slack includes every
// announced cycle when it is not set.
func AdminUsersUnsupportedVersionsExportOptionDateEndOfSupport(timestamp int64) AdminUsersUnsupportedVersionsExportOption {
	return func(params *adminUsersUnsupportedVersionsExportParams) {
		params.dateEndOfSupport = &timestamp
	}
}

// AdminUsersUnsupportedVersionsExportOptionDateSessionsStarted sets the Unix
// timestamp to look for sessions from. Slack starts six months ago when it is
// not set.
func AdminUsersUnsupportedVersionsExportOptionDateSessionsStarted(timestamp int64) AdminUsersUnsupportedVersionsExportOption {
	return func(params *adminUsersUnsupportedVersionsExportParams) {
		params.dateSessionsStarted = &timestamp
	}
}

// AdminUsersUnsupportedVersionsExport asks Slackbot to send the caller a zipped
// CSV of members using unsupported Slack clients.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.unsupportedVersions.export
func (api *Client) AdminUsersUnsupportedVersionsExport(options ...AdminUsersUnsupportedVersionsExportOption) error {
	return api.AdminUsersUnsupportedVersionsExportContext(context.Background(), options...)
}

// AdminUsersUnsupportedVersionsExportContext requests the export with a custom context.
func (api *Client) AdminUsersUnsupportedVersionsExportContext(ctx context.Context, options ...AdminUsersUnsupportedVersionsExportOption) error {
	params := adminUsersUnsupportedVersionsExportParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}

	values := url.Values{
		"token": {api.token},
	}
	if params.dateEndOfSupport != nil {
		values.Add("date_end_of_support", strconv.FormatInt(*params.dateEndOfSupport, 10))
	}
	if params.dateSessionsStarted != nil {
		values.Add("date_sessions_started", strconv.FormatInt(*params.dateSessionsStarted, 10))
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "admin.users.unsupportedVersions.export", values, response); err != nil {
		return err
	}
	return response.Err()
}
