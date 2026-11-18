package slack

import (
	"context"
	"net/url"
	"strings"
)

// AdminUsersAccessInvalidUser is a user that Slack rejected and didn't queue,
// such as one with Reason user_not_frontline_member.
type AdminUsersAccessInvalidUser struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason"`
}

// AdminUsersAccessResponse represents the response from admin.users.access.pause
// and admin.users.access.resume. When Err returns failed_for_some_users, Slack
// still queued the users missing from InvalidUsers; retry only the listed users.
type AdminUsersAccessResponse struct {
	SlackResponse
	InvalidUsers []AdminUsersAccessInvalidUser `json:"invalid_users"`
}

type adminUsersAccessParams struct {
	reason string
}

// AdminUsersAccessPauseOption is an option for AdminUsersAccessPause.
type AdminUsersAccessPauseOption func(*adminUsersAccessParams)

// AdminUsersAccessPauseOptionReason sets the explanation for the access change.
func AdminUsersAccessPauseOptionReason(reason string) AdminUsersAccessPauseOption {
	return func(params *adminUsersAccessParams) {
		params.reason = reason
	}
}

// AdminUsersAccessPause queues a job that pauses org-wide access for up to 1,000
// frontline members. A nil error means the job was queued, not that it finished.
// The response is also returned with an error, so InvalidUsers can be inspected.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.access.pause
func (api *Client) AdminUsersAccessPause(userIDs []string, options ...AdminUsersAccessPauseOption) (*AdminUsersAccessResponse, error) {
	return api.AdminUsersAccessPauseContext(context.Background(), userIDs, options...)
}

// AdminUsersAccessPauseContext pauses access with a custom context.
func (api *Client) AdminUsersAccessPauseContext(ctx context.Context, userIDs []string, options ...AdminUsersAccessPauseOption) (*AdminUsersAccessResponse, error) {
	params := adminUsersAccessParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}
	return api.adminUsersAccess(ctx, "admin.users.access.pause", userIDs, params)
}

// AdminUsersAccessResumeOption is an option for AdminUsersAccessResume.
type AdminUsersAccessResumeOption func(*adminUsersAccessParams)

// AdminUsersAccessResumeOptionReason sets the explanation for the access change.
func AdminUsersAccessResumeOptionReason(reason string) AdminUsersAccessResumeOption {
	return func(params *adminUsersAccessParams) {
		params.reason = reason
	}
}

// AdminUsersAccessResume queues a job that resumes org-wide access for up to
// 1,000 frontline members. A nil error means the job was queued, not that it
// finished. The response is also returned with an error, so InvalidUsers can be
// inspected.
//
// Slack API docs: https://docs.slack.dev/reference/methods/admin.users.access.resume
func (api *Client) AdminUsersAccessResume(userIDs []string, options ...AdminUsersAccessResumeOption) (*AdminUsersAccessResponse, error) {
	return api.AdminUsersAccessResumeContext(context.Background(), userIDs, options...)
}

// AdminUsersAccessResumeContext resumes access with a custom context.
func (api *Client) AdminUsersAccessResumeContext(ctx context.Context, userIDs []string, options ...AdminUsersAccessResumeOption) (*AdminUsersAccessResponse, error) {
	params := adminUsersAccessParams{}
	for _, opt := range options {
		if opt != nil {
			opt(&params)
		}
	}
	return api.adminUsersAccess(ctx, "admin.users.access.resume", userIDs, params)
}

func (api *Client) adminUsersAccess(ctx context.Context, method string, userIDs []string, params adminUsersAccessParams) (*AdminUsersAccessResponse, error) {
	values := url.Values{
		"token":    {api.token},
		"user_ids": {strings.Join(userIDs, ",")},
	}
	if params.reason != "" {
		values.Add("reason", params.reason)
	}

	response := &AdminUsersAccessResponse{}
	if err := api.postMethod(ctx, method, values, response); err != nil {
		return nil, err
	}
	return response, response.Err()
}
