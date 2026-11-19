package slack

import (
	"cmp"
	"context"
	"errors"
	"io"
	"net/url"
)

// SetAppIconParameters contains the image for SetAppIcon. Set exactly one of
// URL or File.
type SetAppIconParameters struct {
	URL      string
	File     io.Reader
	Filename string
}

// SetAppIcon sets the icon of an app.
// For more details, see SetAppIconContext documentation.
func (api *Client) SetAppIcon(token, appID string, icon SetAppIconParameters) error {
	return api.SetAppIconContext(context.Background(), token, appID, icon)
}

// SetAppIconContext sets the icon of an app with a custom context. An empty token
// falls back to the configuration token set with OptionConfigToken.
//
// Slack API docs: https://docs.slack.dev/reference/methods/apps.icon.set
func (api *Client) SetAppIconContext(ctx context.Context, token, appID string, icon SetAppIconParameters) error {
	if token == "" {
		token = api.configToken
	}
	if token == "" {
		return errors.New("apps.icon.set: token cannot be empty")
	}
	if appID == "" {
		return errors.New("apps.icon.set: app ID cannot be empty")
	}
	if (icon.URL == "") == (icon.File == nil) {
		return errors.New("apps.icon.set: exactly one of URL or file must be provided")
	}

	values := url.Values{
		"app_id": {appID},
	}
	response := &SlackResponse{}

	if icon.URL != "" {
		values.Add("token", token)
		values.Add("url", icon.URL)
		if err := api.postMethod(ctx, "apps.icon.set", values, response); err != nil {
			return err
		}
	} else {
		filename := cmp.Or(icon.Filename, "icon.png")
		if err := postWithMultipartResponse(ctx, api.httpclient, api.endpoint+"apps.icon.set", filename, "file", token, values, icon.File, response, api); err != nil {
			return err
		}
	}

	return response.Err()
}
