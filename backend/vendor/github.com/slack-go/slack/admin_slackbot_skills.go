package slack

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// AdminSlackbotSkill is inventory metadata, including invite-only skills.
// It omits instructions, prompts, apps, and effective access and does not grant
// canvas access. Built-in skills and skill sets are excluded. Counts are approximate.
type AdminSlackbotSkill struct {
	ID           string                   `json:"id"`
	DisplayName  string                   `json:"display_name"`
	Description  string                   `json:"description"`
	Category     string                   `json:"category"`
	CreatorID    string                   `json:"creator_id"`
	DateCreated  int64                    `json:"date_created"`
	DateUpdated  *int64                   `json:"date_updated"`
	Visibility   string                   `json:"visibility"`
	IsOfficial   bool                     `json:"is_official"`
	UsageCount   int                      `json:"usage_count"`
	LibraryCount int                      `json:"library_count"`
	AutoEnabled  SlackbotSkillAutoEnabled `json:"auto_enabled"`
}

// SlackbotSkillAutoEnabled describes deployment independently of publication.
// PermissionType is everyone, named_entities, or no_one. UsergroupIDs is empty
// for everyone/no_one and excludes deleted and foreign-org groups in inventory.
type SlackbotSkillAutoEnabled struct {
	PermissionType string   `json:"permission_type"`
	UsergroupIDs   []string `json:"usergroup_ids"`
}

// AdminSlackbotSkillsListResponse is one page of organization inventory.
type AdminSlackbotSkillsListResponse struct {
	SlackResponse
	Skills []AdminSlackbotSkill `json:"skills"`
}

type adminSlackbotSkillsListParams struct {
	values     url.Values
	creatorIDs []string
}

// AdminSlackbotSkillsListOption configures an inventory listing.
type AdminSlackbotSkillsListOption func(*adminSlackbotSkillsListParams)

// AdminSlackbotSkillsListOptionCategory filters by skill category.
func AdminSlackbotSkillsListOptionCategory(category string) AdminSlackbotSkillsListOption {
	return func(p *adminSlackbotSkillsListParams) { p.values.Set("category", category) }
}

// AdminSlackbotSkillsListOptionCursor continues with unchanged filters and sorting.
// Empty pages may carry a continuation. Cursors are bound to the caller and org.
func AdminSlackbotSkillsListOptionCursor(cursor string) AdminSlackbotSkillsListOption {
	return func(p *adminSlackbotSkillsListParams) { p.values.Set("cursor", cursor) }
}

// AdminSlackbotSkillsListOptionLimit sets a page size of 1–200 (default 100).
func AdminSlackbotSkillsListOptionLimit(limit int) AdminSlackbotSkillsListOption {
	return func(p *adminSlackbotSkillsListParams) { p.values.Set("limit", strconv.Itoa(limit)) }
}

// AdminSlackbotSkillsListOptionCreatorIDs filters by at most 100 U or W user IDs.
// Calling it without IDs sends an explicit empty array.
func AdminSlackbotSkillsListOptionCreatorIDs(ids ...string) AdminSlackbotSkillsListOption {
	return func(p *adminSlackbotSkillsListParams) { p.creatorIDs = append([]string{}, ids...) }
}

// AdminSlackbotSkillsListOptionPublishStatus selects all (default), published, or unpublished.
func AdminSlackbotSkillsListOptionPublishStatus(status string) AdminSlackbotSkillsListOption {
	return func(p *adminSlackbotSkillsListParams) { p.values.Set("publish_status", status) }
}

// AdminSlackbotSkillsListOptionAutoEnabledStatus selects all (default), enabled, or disabled.
func AdminSlackbotSkillsListOptionAutoEnabledStatus(status string) AdminSlackbotSkillsListOption {
	return func(p *adminSlackbotSkillsListParams) { p.values.Set("auto_enabled_status", status) }
}

// AdminSlackbotSkillsListOptionSort selects name (default), usage_count, or
// date_updated. Name sorting ignores case. Null update dates use creation time.
func AdminSlackbotSkillsListOptionSort(sort string) AdminSlackbotSkillsListOption {
	return func(p *adminSlackbotSkillsListParams) { p.values.Set("sort", sort) }
}

// AdminSlackbotSkillsListOptionSortDir selects asc (default) or desc.
// File ID breaks ties in the requested direction.
func AdminSlackbotSkillsListOptionSortDir(direction string) AdminSlackbotSkillsListOption {
	return func(p *adminSlackbotSkillsListParams) { p.values.Set("sort_dir", direction) }
}

// AdminSlackbotSkillsList lists metadata owned by an Enterprise organization.
// Requires an org admin/owner's user token from an org-level app installation
// with admin.skills:read. Pagination is not a snapshot and indexing is asynchronous.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.slackbot.skills.list
func (api *Client) AdminSlackbotSkillsList(options ...AdminSlackbotSkillsListOption) (*AdminSlackbotSkillsListResponse, error) {
	return api.AdminSlackbotSkillsListContext(context.Background(), options...)
}

// AdminSlackbotSkillsListContext lists one inventory page with a custom context.
// Follow ResponseMetadata.Cursor until empty, even when Skills is empty.
func (api *Client) AdminSlackbotSkillsListContext(ctx context.Context, options ...AdminSlackbotSkillsListOption) (*AdminSlackbotSkillsListResponse, error) {
	params := adminSlackbotSkillsListParams{values: url.Values{"token": {api.token}}}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}
	if params.creatorIDs != nil {
		creatorIDs, err := json.Marshal(params.creatorIDs)
		if err != nil {
			return nil, err
		}
		params.values.Set("creator_ids", string(creatorIDs))
	}
	response := &AdminSlackbotSkillsListResponse{}
	if err := api.postMethod(ctx, "admin.slackbot.skills.list", params.values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// SlackbotSkillFailure is a per-skill failure in a publication batch. Error can
// be skill_not_found, method_not_supported_for_skill_type, restricted_action,
// or internal_error. Successful items are not rolled back.
type SlackbotSkillFailure struct {
	SkillID string `json:"skill_id"`
	Error   string `json:"error"`
}

// AdminSlackbotSkillsResponse preserves individual failures when Err returns
// failed_for_some_skills. Always inspect the response even when err is non-nil.
type AdminSlackbotSkillsResponse struct {
	SlackResponse
	FailedSkills []SlackbotSkillFailure `json:"failed_skills,omitempty"`
}

// AdminSlackbotSkillsPublish publishes 1–50 canvas skills to the org catalog.
// Requires an Enterprise org admin/owner's user token from an org-level app
// installation with admin.skills:write. Publishing shares canvases org-wide if
// needed, requiring edit/share permission. Existing sharing levels are retained.
// It does not auto-enable skills. Already-published skills succeed. Duplicate IDs
// count toward the limit but are processed once. Partial successes remain applied;
// sharing can succeed even if publication fails, and remains after that failure.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.slackbot.skills.publish
func (api *Client) AdminSlackbotSkillsPublish(skillIDs []string) (*AdminSlackbotSkillsResponse, error) {
	return api.AdminSlackbotSkillsPublishContext(context.Background(), skillIDs)
}

// AdminSlackbotSkillsPublishContext publishes skills with a custom context.
func (api *Client) AdminSlackbotSkillsPublishContext(ctx context.Context, skillIDs []string) (*AdminSlackbotSkillsResponse, error) {
	ids, err := json.Marshal(append([]string{}, skillIDs...))
	if err != nil {
		return nil, err
	}
	values := url.Values{"token": {api.token}, "skill_ids": {string(ids)}}
	response := &AdminSlackbotSkillsResponse{}
	if err := api.postMethod(ctx, "admin.slackbot.skills.publish", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// AdminSlackbotSkillsSetAutoEnabledResponse contains the resulting deployment,
// or audience validation details on an error. UsergroupIDs contains inaccessible
// groups for skill_not_visible_to_audience; InvalidUsergroupIDs contains invalid
// groups for usergroup_not_found. Successful group IDs are sorted by Slack.
type AdminSlackbotSkillsSetAutoEnabledResponse struct {
	SlackResponse
	SlackbotSkillAutoEnabled
	SkillID             string   `json:"skill_id"`
	InvalidUsergroupIDs []string `json:"invalid_usergroup_ids,omitempty"`
}

type adminSlackbotSkillsSetAutoEnabledParams struct {
	usergroupIDs []string
}

// AdminSlackbotSkillsSetAutoEnabledOption configures optional deployment arguments.
type AdminSlackbotSkillsSetAutoEnabledOption func(*adminSlackbotSkillsSetAutoEnabledParams)

// AdminSlackbotSkillsSetAutoEnabledOptionUsergroupIDs supplies 1–100 groups for
// named_entities. Omit this option for everyone/no_one. Duplicate IDs count toward
// the input limit, but Slack stores them once. No IDs sends an explicit [].
func AdminSlackbotSkillsSetAutoEnabledOptionUsergroupIDs(ids ...string) AdminSlackbotSkillsSetAutoEnabledOption {
	return func(p *adminSlackbotSkillsSetAutoEnabledParams) { p.usergroupIDs = append([]string{}, ids...) }
}

// AdminSlackbotSkillsSetAutoEnabled deploys a canvas skill to everyone,
// named_entities, or no_one. Requires an Enterprise org admin/owner's user token
// from an org-level installation with admin.skills:write. Canvas access must
// already cover the audience; this call does not share or publish the skill.
// Without org-wide canvas access, Slack permits at most 1,000 member checks
// across groups, counting overlapping members once per group. Retrying the same
// request after internal_error is safe. Slack rechecks access when the skill runs.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.slackbot.skills.setAutoEnabled
func (api *Client) AdminSlackbotSkillsSetAutoEnabled(skillID, permissionType string, options ...AdminSlackbotSkillsSetAutoEnabledOption) (*AdminSlackbotSkillsSetAutoEnabledResponse, error) {
	return api.AdminSlackbotSkillsSetAutoEnabledContext(context.Background(), skillID, permissionType, options...)
}

// AdminSlackbotSkillsSetAutoEnabledContext changes deployment with a custom context.
// no_one disables auto-enablement. Everyone and no_one return empty group arrays.
func (api *Client) AdminSlackbotSkillsSetAutoEnabledContext(ctx context.Context, skillID, permissionType string, options ...AdminSlackbotSkillsSetAutoEnabledOption) (*AdminSlackbotSkillsSetAutoEnabledResponse, error) {
	params := adminSlackbotSkillsSetAutoEnabledParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}
	values := url.Values{
		"token": {api.token}, "skill_id": {skillID}, "permission_type": {permissionType},
	}
	if params.usergroupIDs != nil {
		usergroupIDs, err := json.Marshal(params.usergroupIDs)
		if err != nil {
			return nil, err
		}
		values.Set("usergroup_ids", string(usergroupIDs))
	}
	response := &AdminSlackbotSkillsSetAutoEnabledResponse{}
	if err := api.postMethod(ctx, "admin.slackbot.skills.setAutoEnabled", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}

// AdminSlackbotSkillsUnpublish removes 1–50 skills from the org catalog while
// preserving canvas access, libraries, and deployment. Requires an Enterprise
// org admin/owner's user token from an org-level installation with admin.skills:write.
// Already-unpublished skills succeed. Duplicates count toward the limit but are
// processed once. Partial successes remain applied; inspect FailedSkills on error.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.slackbot.skills.unpublish
func (api *Client) AdminSlackbotSkillsUnpublish(skillIDs []string) (*AdminSlackbotSkillsResponse, error) {
	return api.AdminSlackbotSkillsUnpublishContext(context.Background(), skillIDs)
}

// AdminSlackbotSkillsUnpublishContext unpublishes skills with a custom context.
func (api *Client) AdminSlackbotSkillsUnpublishContext(ctx context.Context, skillIDs []string) (*AdminSlackbotSkillsResponse, error) {
	ids, err := json.Marshal(append([]string{}, skillIDs...))
	if err != nil {
		return nil, err
	}
	values := url.Values{"token": {api.token}, "skill_ids": {string(ids)}}
	response := &AdminSlackbotSkillsResponse{}
	if err := api.postMethod(ctx, "admin.slackbot.skills.unpublish", values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}
