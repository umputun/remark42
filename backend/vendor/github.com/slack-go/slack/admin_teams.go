package slack

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

type adminTeamsListParams struct {
	limit  int
	cursor string
}

// AdminTeamsListOption is an option for AdminTeamsList.
type AdminTeamsListOption func(*adminTeamsListParams)

// AdminTeamsListOptionLimit sets the maximum number of workspaces to return.
func AdminTeamsListOptionLimit(limit int) AdminTeamsListOption {
	return func(params *adminTeamsListParams) {
		params.limit = limit
	}
}

// AdminTeamsListOptionCursor sets the cursor for pagination.
func AdminTeamsListOptionCursor(cursor string) AdminTeamsListOption {
	return func(params *adminTeamsListParams) {
		params.cursor = cursor
	}
}

// AdminTeamPrimaryOwner represents the primary owner of a workspace.
type AdminTeamPrimaryOwner struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

// AdminTeam represents a workspace in an Enterprise organization.
type AdminTeam struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Discoverability TeamDiscoverability   `json:"discoverability"`
	PrimaryOwner    AdminTeamPrimaryOwner `json:"primary_owner"`
	TeamURL         string                `json:"team_url"`
}

// AdminTeamsListResponse represents the response from admin.teams.list.
type AdminTeamsListResponse struct {
	SlackResponse
	Teams []AdminTeam `json:"teams"`
}

// AdminTeamsList lists all workspaces in an Enterprise organization.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.teams.list
func (api *Client) AdminTeamsList(ctx context.Context, options ...AdminTeamsListOption) (*AdminTeamsListResponse, error) {
	params := adminTeamsListParams{}
	for _, opt := range options {
		opt(&params)
	}

	values := url.Values{
		"token": {api.token},
	}

	if params.limit > 0 {
		values.Add("limit", strconv.Itoa(params.limit))
	}

	if params.cursor != "" {
		values.Add("cursor", params.cursor)
	}

	response := &AdminTeamsListResponse{}
	err := api.postMethod(ctx, "admin.teams.list", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// AdminTeamSettings contains workspace settings returned by admin.teams.settings.info.
type AdminTeamSettings struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	URL              string           `json:"url"`
	Domain           string           `json:"domain"`
	EmailDomain      string           `json:"email_domain"`
	AvatarBaseURL    string           `json:"avatar_base_url"`
	IsVerified       bool             `json:"is_verified"`
	Icon             TeamSettingsIcon `json:"icon"`
	EnterpriseID     string           `json:"enterprise_id"`
	EnterpriseName   string           `json:"enterprise_name"`
	EnterpriseDomain string           `json:"enterprise_domain"`
	DefaultChannels  []string         `json:"default_channels"`
}

// TeamSettingsIcon contains team icon URLs and a default flag.
type TeamSettingsIcon struct {
	ImageDefault bool   `json:"image_default"`
	Image34      string `json:"image_34"`
	Image44      string `json:"image_44"`
	Image68      string `json:"image_68"`
	Image88      string `json:"image_88"`
	Image102     string `json:"image_102"`
	Image132     string `json:"image_132"`
	Image230     string `json:"image_230"`
}

// TeamDiscoverability represents the discoverability setting for a workspace.
type TeamDiscoverability string

const (
	TeamDiscoverabilityOpen       TeamDiscoverability = "open"
	TeamDiscoverabilityInviteOnly TeamDiscoverability = "invite_only"
	TeamDiscoverabilityClosed     TeamDiscoverability = "closed"
	TeamDiscoverabilityUnlisted   TeamDiscoverability = "unlisted"
)

type adminTeamSettingsInfoResponse struct {
	Team AdminTeamSettings `json:"team"`
	SlackResponse
}

// AdminTeamsSettingsInfo returns workspace settings.
// Slack API docs: https://docs.slack.dev/reference/methods/admin.teams.settings.info
func (api *Client) AdminTeamsSettingsInfo(ctx context.Context, teamID string) (*AdminTeamSettings, error) {
	values := url.Values{
		"token":   {api.token},
		"team_id": {teamID},
	}

	response := &adminTeamSettingsInfoResponse{}
	err := api.postMethod(ctx, "admin.teams.settings.info", values, response)
	if err != nil {
		return nil, err
	}

	return &response.Team, response.Err()
}

// AdminTeamsSettingsSetDefaultChannels sets the default channels for a workspace.
// Slack API docs: https://docs.slack.dev/reference/methods/admin.teams.settings.setDefaultChannels
func (api *Client) AdminTeamsSettingsSetDefaultChannels(ctx context.Context, teamID string, channelIDs ...string) error {
	values := url.Values{
		"token":       {api.token},
		"team_id":     {teamID},
		"channel_ids": {strings.Join(channelIDs, ",")},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.teams.settings.setDefaultChannels", values, response)
	if err != nil {
		return err
	}
	return response.Err()
}

// AdminTeamsSettingsSetDescription sets the description for a workspace.
// Slack API docs: https://docs.slack.dev/reference/methods/admin.teams.settings.setDescription
func (api *Client) AdminTeamsSettingsSetDescription(ctx context.Context, teamID, description string) error {
	values := url.Values{
		"token":       {api.token},
		"team_id":     {teamID},
		"description": {description},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.teams.settings.setDescription", values, response)
	if err != nil {
		return err
	}
	return response.Err()
}

// AdminTeamsSettingsSetDiscoverability sets the discoverability for a workspace.
// The discoverability parameter must be one of: open, invite_only, closed, or unlisted.
// Slack API docs: https://docs.slack.dev/reference/methods/admin.teams.settings.setDiscoverability
func (api *Client) AdminTeamsSettingsSetDiscoverability(ctx context.Context, teamID string, discoverability TeamDiscoverability) error {
	values := url.Values{
		"token":           {api.token},
		"team_id":         {teamID},
		"discoverability": {string(discoverability)},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.teams.settings.setDiscoverability", values, response)
	if err != nil {
		return err
	}
	return response.Err()
}

// AdminTeamsSettingsSetIcon sets the icon for a workspace.
// Slack API docs: https://docs.slack.dev/reference/methods/admin.teams.settings.setIcon
func (api *Client) AdminTeamsSettingsSetIcon(ctx context.Context, teamID, imageURL string) error {
	values := url.Values{
		"token":     {api.token},
		"team_id":   {teamID},
		"image_url": {imageURL},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.teams.settings.setIcon", values, response)
	if err != nil {
		return err
	}
	return response.Err()
}

// AdminTeamsSettingsSetName sets the name for a workspace.
// Slack API docs: https://docs.slack.dev/reference/methods/admin.teams.settings.setName
func (api *Client) AdminTeamsSettingsSetName(ctx context.Context, teamID, name string) error {
	values := url.Values{
		"token":   {api.token},
		"team_id": {teamID},
		"name":    {name},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "admin.teams.settings.setName", values, response)
	if err != nil {
		return err
	}
	return response.Err()
}
