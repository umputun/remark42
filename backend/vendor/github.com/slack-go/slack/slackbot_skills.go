package slack

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// SlackbotSkill describes a canvas or built-in skill. Instructions are populated
// by GetSlackbotSkillInfo; other methods return null. Dates are Unix seconds.
// Slack does not document every field's nullability or all type/visibility values.
type SlackbotSkill struct {
	ID            string                   `json:"id"`
	Type          string                   `json:"type"`
	DisplayName   string                   `json:"display_name"`
	Name          *string                  `json:"name"`
	Description   string                   `json:"description"`
	Category      string                   `json:"category"`
	ExamplePrompt string                   `json:"example_prompt"`
	Instructions  *string                  `json:"instructions"`
	Visibility    string                   `json:"visibility"`
	IsOfficial    bool                     `json:"is_official"`
	Access        string                   `json:"access"`
	MCPServers    []SlackbotSkillMCPServer `json:"mcp_servers"`
	UsageCount    int                      `json:"usage_count"`
	LibraryCount  int                      `json:"library_count"`
	CreatorID     string                   `json:"creator_id"`
	DateCreated   int64                    `json:"date_created"`
	DateUpdated   *int64                   `json:"date_updated"`
	Permalink     string                   `json:"permalink"`
}

// SlackbotSkillMCPServer is an app declaration returned in a skill response.
// The request item schema is undocumented; MCPServers options accept raw JSON
// instead of assuming that requests use this response shape.
type SlackbotSkillMCPServer struct {
	ServiceType string `json:"service_type"`
	AppID       string `json:"app_id"`
	ServerID    string `json:"server_id"`
	DisplayName string `json:"display_name"`
	IsAvailable bool   `json:"is_available"`
}

// SlackbotSkillResponse is returned by skill create, info, and update calls.
// InvalidMCPServers preserves undocumented error details on failed writes.
type SlackbotSkillResponse struct {
	SlackResponse
	Skill             SlackbotSkill   `json:"skill,omitzero"`
	InvalidMCPServers json.RawMessage `json:"invalid_mcp_servers,omitempty"`
}

type createSlackbotSkillParams struct {
	values     url.Values
	mcpServers []json.RawMessage
}

// CreateSlackbotSkillOption configures an optional skill creation argument.
type CreateSlackbotSkillOption func(*createSlackbotSkillParams)

// CreateSlackbotSkillOptionInstructions sets Markdown (at most 500,000 bytes).
// Supply exactly one of Instructions and CanvasID.
func CreateSlackbotSkillOptionInstructions(instructions string) CreateSlackbotSkillOption {
	return func(p *createSlackbotSkillParams) { p.values.Set("instructions", instructions) }
}

// CreateSlackbotSkillOptionCanvasID converts a canvas owned by the caller.
// Supply exactly one of Instructions and CanvasID.
func CreateSlackbotSkillOptionCanvasID(canvasID string) CreateSlackbotSkillOption {
	return func(p *createSlackbotSkillParams) { p.values.Set("canvas_id", canvasID) }
}

// CreateSlackbotSkillOptionDescription describes when to use the skill (400 characters maximum).
func CreateSlackbotSkillOptionDescription(description string) CreateSlackbotSkillOption {
	return func(p *createSlackbotSkillParams) { p.values.Set("description", description) }
}

// CreateSlackbotSkillOptionCategory sets the skill category; Slack validates it.
func CreateSlackbotSkillOptionCategory(category string) CreateSlackbotSkillOption {
	return func(p *createSlackbotSkillParams) { p.values.Set("category", category) }
}

// CreateSlackbotSkillOptionExamplePrompt sets a prompt (200 characters maximum).
// Slack replaces {skill} with a mention of the skill.
func CreateSlackbotSkillOptionExamplePrompt(prompt string) CreateSlackbotSkillOption {
	return func(p *createSlackbotSkillParams) { p.values.Set("example_prompt", prompt) }
}

// CreateSlackbotSkillOptionMCPServers declares required apps without granting
// access to them. Entries use Slack's undocumented request schema. Calling this
// without entries sends []. The caller's MCP server limit applies.
func CreateSlackbotSkillOptionMCPServers(servers ...json.RawMessage) CreateSlackbotSkillOption {
	return func(p *createSlackbotSkillParams) { p.mcpServers = append([]json.RawMessage{}, servers...) }
}

// CreateSlackbotSkill creates a skill, attempting to add it to the caller's
// library. Markdown skills start invite-only; converted canvases retain sharing.
// Display names are not unique and have a 255-character limit. This is not an
// upsert: retain Skill.ID for future updates. Requires a user token with skills:write.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackbot.skills.create
func (api *Client) CreateSlackbotSkill(displayName string, options ...CreateSlackbotSkillOption) (*SlackbotSkillResponse, error) {
	return api.CreateSlackbotSkillContext(context.Background(), displayName, options...)
}

// CreateSlackbotSkillContext creates a skill with a custom context. Slack
// validates that exactly one of instructions and canvas_id is provided.
func (api *Client) CreateSlackbotSkillContext(ctx context.Context, displayName string, options ...CreateSlackbotSkillOption) (*SlackbotSkillResponse, error) {
	params := createSlackbotSkillParams{values: url.Values{
		"token": {api.token}, "display_name": {displayName},
	}}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}
	if params.mcpServers != nil {
		servers, err := json.Marshal(params.mcpServers)
		if err != nil {
			return nil, err
		}
		params.values.Set("mcp_servers", string(servers))
	}
	response := &SlackbotSkillResponse{}
	if err := api.postMethod(ctx, "slackbot.skills.create", params.values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// GetSlackbotSkillInfo returns a skill and its Markdown instructions. SkillID
// accepts a canvas file ID or builtin: reference. Requires a user token with
// skills:read and read access to the canvas; org admin status does not bypass it.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackbot.skills.info
func (api *Client) GetSlackbotSkillInfo(skillID string) (*SlackbotSkillResponse, error) {
	return api.GetSlackbotSkillInfoContext(context.Background(), skillID)
}

// GetSlackbotSkillInfoContext fetches a skill with a custom context.
func (api *Client) GetSlackbotSkillInfoContext(ctx context.Context, skillID string) (*SlackbotSkillResponse, error) {
	response := &SlackbotSkillResponse{}
	values := url.Values{"token": {api.token}, "skill_id": {skillID}}
	if err := api.postMethod(ctx, "slackbot.skills.info", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

type updateSlackbotSkillParams struct {
	values     url.Values
	mcpServers []json.RawMessage
}

// UpdateSlackbotSkillOption changes an explicitly supplied field; omitted fields
// are left unchanged. Empty strings and arrays are sent, not omitted.
type UpdateSlackbotSkillOption func(*updateSlackbotSkillParams)

// UpdateSlackbotSkillOptionDisplayName changes the label (255 characters maximum).
func UpdateSlackbotSkillOptionDisplayName(name string) UpdateSlackbotSkillOption {
	return func(p *updateSlackbotSkillParams) { p.values.Set("display_name", name) }
}

// UpdateSlackbotSkillOptionInstructions replaces the entire Markdown document
// (500,000 bytes maximum).
func UpdateSlackbotSkillOptionInstructions(instructions string) UpdateSlackbotSkillOption {
	return func(p *updateSlackbotSkillParams) { p.values.Set("instructions", instructions) }
}

// UpdateSlackbotSkillOptionDescription sets or clears the description (400 characters maximum).
func UpdateSlackbotSkillOptionDescription(description string) UpdateSlackbotSkillOption {
	return func(p *updateSlackbotSkillParams) { p.values.Set("description", description) }
}

// UpdateSlackbotSkillOptionCategory sets the category, or clears it with "".
func UpdateSlackbotSkillOptionCategory(category string) UpdateSlackbotSkillOption {
	return func(p *updateSlackbotSkillParams) { p.values.Set("category", category) }
}

// UpdateSlackbotSkillOptionExamplePrompt sets or clears the example prompt
// (200 characters maximum). Slack replaces {skill} with a skill mention.
func UpdateSlackbotSkillOptionExamplePrompt(prompt string) UpdateSlackbotSkillOption {
	return func(p *updateSlackbotSkillParams) { p.values.Set("example_prompt", prompt) }
}

// UpdateSlackbotSkillOptionMCPServers replaces the full list of required apps.
// With no entries it sends [] to clear the list. Entries use Slack's undocumented
// request schema. Declaring an app does not grant access to it.
func UpdateSlackbotSkillOptionMCPServers(servers ...json.RawMessage) UpdateSlackbotSkillOption {
	return func(p *updateSlackbotSkillParams) { p.mcpServers = append([]json.RawMessage{}, servers...) }
}

// UpdateSlackbotSkill changes a canvas skill's instructions or metadata using a
// user token with skills:write. Updates may partially apply; retrying the same
// request is safe. Markdown normalization may rewrite unchanged instructions.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackbot.skills.update
func (api *Client) UpdateSlackbotSkill(skillID string, options ...UpdateSlackbotSkillOption) (*SlackbotSkillResponse, error) {
	return api.UpdateSlackbotSkillContext(context.Background(), skillID, options...)
}

// UpdateSlackbotSkillContext updates a skill with a custom context.
func (api *Client) UpdateSlackbotSkillContext(ctx context.Context, skillID string, options ...UpdateSlackbotSkillOption) (*SlackbotSkillResponse, error) {
	params := updateSlackbotSkillParams{values: url.Values{
		"token": {api.token}, "skill_id": {skillID},
	}}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}
	if params.mcpServers != nil {
		servers, err := json.Marshal(params.mcpServers)
		if err != nil {
			return nil, err
		}
		params.values.Set("mcp_servers", string(servers))
	}
	response := &SlackbotSkillResponse{}
	if err := api.postMethod(ctx, "slackbot.skills.update", params.values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// ListSlackbotSkillsResponse is one page of skills, without instructions.
// Continue using ResponseMetadata.Cursor even if Skills is empty.
type ListSlackbotSkillsResponse struct {
	SlackResponse
	Skills []SlackbotSkill `json:"skills"`
}

type listSlackbotSkillsParams struct{ values url.Values }

// ListSlackbotSkillsOption configures a skill listing. Omitted values use Slack's defaults.
type ListSlackbotSkillsOption func(*listSlackbotSkillsParams)

// ListSlackbotSkillsOptionFilter selects library (default), owned, or catalog.
// Library includes skills from deployments and skill sets.
func ListSlackbotSkillsOptionFilter(filter string) ListSlackbotSkillsOption {
	return func(p *listSlackbotSkillsParams) { p.values.Set("filter", filter) }
}

// ListSlackbotSkillsOptionCategory filters by category.
func ListSlackbotSkillsOptionCategory(category string) ListSlackbotSkillsOption {
	return func(p *listSlackbotSkillsParams) { p.values.Set("category", category) }
}

// ListSlackbotSkillsOptionCursor continues a listing with the same filter and category.
func ListSlackbotSkillsOptionCursor(cursor string) ListSlackbotSkillsOption {
	return func(p *listSlackbotSkillsParams) { p.values.Set("cursor", cursor) }
}

// ListSlackbotSkillsOptionLimit sets the page size. Slack defaults to 100 and
// does not document the accepted range; fewer results may be returned.
func ListSlackbotSkillsOptionLimit(limit int) ListSlackbotSkillsOption {
	return func(p *listSlackbotSkillsParams) { p.values.Set("limit", strconv.Itoa(limit)) }
}

// ListSlackbotSkills lists skills from the caller's perspective without their
// instructions. Requires a user token with skills:read. Pagination is explicit.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackbot.skills.list
func (api *Client) ListSlackbotSkills(options ...ListSlackbotSkillsOption) (*ListSlackbotSkillsResponse, error) {
	return api.ListSlackbotSkillsContext(context.Background(), options...)
}

// ListSlackbotSkillsContext lists one page of skills with a custom context.
func (api *Client) ListSlackbotSkillsContext(ctx context.Context, options ...ListSlackbotSkillsOption) (*ListSlackbotSkillsResponse, error) {
	params := listSlackbotSkillsParams{values: url.Values{"token": {api.token}}}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}
	response := &ListSlackbotSkillsResponse{}
	if err := api.postMethod(ctx, "slackbot.skills.list", params.values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// AddSlackbotSkillToLibrary enables a readable skill for the caller. Requires a
// user token with skills:write and permission to use skills, not author them.
// Already-enabled skills succeed. For deployments and skill sets, this restores
// effective enablement without adding a separate library entry. Canvas skills
// must belong to the caller's organization. Sharing is unchanged.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackbot.skills.library.add
func (api *Client) AddSlackbotSkillToLibrary(skillID string) error {
	return api.AddSlackbotSkillToLibraryContext(context.Background(), skillID)
}

// AddSlackbotSkillToLibraryContext enables a skill with a custom context.
// Retrying an internal_error can finish a partially completed addition.
func (api *Client) AddSlackbotSkillToLibraryContext(ctx context.Context, skillID string) error {
	response := &SlackResponse{}
	values := url.Values{"token": {api.token}, "skill_id": {skillID}}
	if err := api.postMethod(ctx, "slackbot.skills.library.add", values, response); err != nil {
		return err
	}
	return response.Err()
}

// RemoveSlackbotSkillFromLibrary disables a readable skill for the caller,
// including skills supplied through deployments or skill sets. Already-disabled
// skills succeed. Does not delete, unpublish, or change sharing or other users.
// Requires a user token with skills:write and permission to use skills.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackbot.skills.library.remove
func (api *Client) RemoveSlackbotSkillFromLibrary(skillID string) error {
	return api.RemoveSlackbotSkillFromLibraryContext(context.Background(), skillID)
}

// RemoveSlackbotSkillFromLibraryContext disables a skill with a custom context.
func (api *Client) RemoveSlackbotSkillFromLibraryContext(ctx context.Context, skillID string) error {
	response := &SlackResponse{}
	values := url.Values{"token": {api.token}, "skill_id": {skillID}}
	if err := api.postMethod(ctx, "slackbot.skills.library.remove", values, response); err != nil {
		return err
	}
	return response.Err()
}

// DeleteSlackbotSkill deletes a canvas skill and its underlying canvas, removing
// it from all libraries, skill sets, and the org catalog. Built-in skills cannot
// be deleted. Requires a user token with skills:write and deletion permission.
// Use RemoveSlackbotSkillFromLibrary to disable a skill only for the caller.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackbot.skills.delete
func (api *Client) DeleteSlackbotSkill(skillID string) error {
	return api.DeleteSlackbotSkillContext(context.Background(), skillID)
}

// DeleteSlackbotSkillContext deletes a skill with a custom context.
func (api *Client) DeleteSlackbotSkillContext(ctx context.Context, skillID string) error {
	response := &SlackResponse{}
	values := url.Values{"token": {api.token}, "skill_id": {skillID}}
	if err := api.postMethod(ctx, "slackbot.skills.delete", values, response); err != nil {
		return err
	}
	return response.Err()
}
