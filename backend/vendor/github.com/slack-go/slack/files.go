package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

const (
	// Add here the defaults in the site
	DEFAULT_FILES_USER        = ""
	DEFAULT_FILES_CHANNEL     = ""
	DEFAULT_FILES_TS_FROM     = 0
	DEFAULT_FILES_TS_TO       = -1
	DEFAULT_FILES_TYPES       = "all"
	DEFAULT_FILES_COUNT       = 100
	DEFAULT_FILES_PAGE        = 1
	DEFAULT_FILES_SHOW_HIDDEN = false
)

// File contains all the information for a file
type File struct {
	ID        string   `json:"id"`
	Created   JSONTime `json:"created"`
	Timestamp JSONTime `json:"timestamp"`

	Name              string `json:"name"`
	Title             string `json:"title"`
	Mimetype          string `json:"mimetype"`
	ImageExifRotation int    `json:"image_exif_rotation"`
	Filetype          string `json:"filetype"`
	PrettyType        string `json:"pretty_type"`
	User              string `json:"user"`
	UserTeam          string `json:"user_team,omitempty"`
	AltTxt            string `json:"alt_txt,omitempty"`

	Mode         string  `json:"mode"`
	Editable     bool    `json:"editable"`
	IsExternal   bool    `json:"is_external"`
	ExternalType string  `json:"external_type"`
	ExternalID   string  `json:"external_id,omitempty"`
	ExternalURL  *string `json:"external_url,omitempty"`

	AppID                      string `json:"app_id,omitempty"`
	AppProvidesFileWorkObjects *bool  `json:"app_provides_file_work_objects,omitempty"`

	Size int `json:"size"`

	URL                string `json:"url"`          // Deprecated - never set
	URLDownload        string `json:"url_download"` // Deprecated - never set
	URLPrivate         string `json:"url_private"`
	URLPrivateDownload string `json:"url_private_download"`

	OriginalH   int    `json:"original_h"`
	OriginalW   int    `json:"original_w"`
	Thumb64     string `json:"thumb_64"`
	Thumb80     string `json:"thumb_80"`
	Thumb160    string `json:"thumb_160"`
	Thumb360    string `json:"thumb_360"`
	Thumb360Gif string `json:"thumb_360_gif"`
	Thumb360W   int    `json:"thumb_360_w"`
	Thumb360H   int    `json:"thumb_360_h"`
	Thumb480    string `json:"thumb_480"`
	Thumb480W   int    `json:"thumb_480_w"`
	Thumb480H   int    `json:"thumb_480_h"`
	Thumb480Gif string `json:"thumb_480_gif,omitempty"`
	Thumb720    string `json:"thumb_720"`
	Thumb720W   int    `json:"thumb_720_w"`
	Thumb720H   int    `json:"thumb_720_h"`
	Thumb800    string `json:"thumb_800,omitempty"`
	Thumb800W   int    `json:"thumb_800_w,omitempty"`
	Thumb800H   int    `json:"thumb_800_h,omitempty"`
	ThumbTiny   string `json:"thumb_tiny,omitempty"`
	ThumbPDF    string `json:"thumb_pdf,omitempty"`
	ThumbPDFH   int    `json:"thumb_pdf_h,omitempty"`
	ThumbPDFW   int    `json:"thumb_pdf_w,omitempty"`
	Thumb960    string `json:"thumb_960"`
	Thumb960W   int    `json:"thumb_960_w"`
	Thumb960H   int    `json:"thumb_960_h"`
	Thumb1024   string `json:"thumb_1024"`
	Thumb1024W  int    `json:"thumb_1024_w"`
	Thumb1024H  int    `json:"thumb_1024_h"`
	Deanimate   string `json:"deanimate,omitempty"`

	ThumbVideo          string `json:"thumb_video"`
	ThumbVideoW         int    `json:"thumb_video_w"`
	ThumbVideoH         int    `json:"thumb_video_h"`
	ThumbVideoTimestamp string `json:"thumb_video_ts"`

	DurationMs       int    `json:"duration_ms"`
	HLS              string `json:"hls"`
	HLSEmbed         string `json:"hls_embed"`
	HLSPreview       string `json:"hls_preview"`
	MediaDisplayType string `json:"media_display_type"`
	MP4              string `json:"mp4"`
	MP4Low           string `json:"mp4_low"`
	VTT              string `json:"vtt"`

	AAC                            string             `json:"aac,omitempty"`
	AudioWaveSamples               []int              `json:"audio_wave_samples,omitempty"`
	IsTranscriptionRegionSupported *bool              `json:"is_transcription_region_supported,omitempty"`
	Transcription                  *FileTranscription `json:"transcription,omitempty"`

	Permalink       string `json:"permalink"`
	PermalinkPublic string `json:"permalink_public"`

	EditLink         string `json:"edit_link"`
	Preview          string `json:"preview"`
	PreviewHighlight string `json:"preview_highlight"`
	Lines            int    `json:"lines"`
	LinesMore        int    `json:"lines_more"`

	IsPublic        bool     `json:"is_public"`
	PublicURLShared bool     `json:"public_url_shared"`
	Channels        []string `json:"channels"`
	Groups          []string `json:"groups"`
	IMs             []string `json:"ims"`
	InitialComment  Comment  `json:"initial_comment"`
	CommentsCount   int      `json:"comments_count"`
	NumStars        int      `json:"num_stars"`
	IsStarred       bool     `json:"is_starred"`
	Shares          Share    `json:"shares"`
	HasMoreShares   *bool    `json:"has_more_shares,omitempty"`
	FileAccess      string   `json:"file_access,omitempty"`

	Access                             string           `json:"access,omitempty"`
	DMMPDMUsersWithFileAccess          []FileUserAccess `json:"dm_mpdm_users_with_file_access,omitempty"`
	PrivateChannelsWithFileAccessCount *int             `json:"private_channels_with_file_access_count,omitempty"`
	TeamsSharedWith                    []string         `json:"teams_shared_with,omitempty"`
	IsRestrictedSharingEnabled         *bool            `json:"is_restricted_sharing_enabled,omitempty"`
	OrgOrWorkspaceAccess               string           `json:"org_or_workspace_access,omitempty"`
	// Canvas responses can supply last_read in milliseconds; preserve the raw value.
	LastRead *int64 `json:"last_read,omitempty"`

	Subject string              `json:"subject"`
	To      []EmailFileUserInfo `json:"to"`
	From    []EmailFileUserInfo `json:"from"`
	Cc      []EmailFileUserInfo `json:"cc"`
	Headers EmailHeaders        `json:"headers"`

	PlainText          string `json:"plain_text"`
	PreviewPlainText   string `json:"preview_plain_text"`
	PreviewIsTruncated *bool  `json:"preview_is_truncated,omitempty"`

	URLStaticPreview              string    `json:"url_static_preview,omitempty"`
	QuipThreadID                  string    `json:"quip_thread_id,omitempty"`
	Updated                       *JSONTime `json:"updated,omitempty"`
	UpdateNotification            *JSONTime `json:"update_notification,omitempty"`
	CanvasCreatorID               string    `json:"canvas_creator_id,omitempty"`
	CanvasReadTime                *float64  `json:"canvas_readtime,omitempty"`
	CanvasPrintingEnabled         *bool     `json:"canvas_printing_enabled,omitempty"`
	TeamPrefVersionHistoryEnabled *bool     `json:"team_pref_version_history_enabled,omitempty"`
	ExcludedFromSlackAI           *bool     `json:"excluded_from_slack_ai,omitempty"`
	IsAISuggested                 *bool     `json:"is_ai_suggested,omitempty"`
	ShowBadge                     *bool     `json:"show_badge,omitempty"`
	TitleBlocks                   *Blocks   `json:"title_blocks,omitempty"`
	// A pointer preserves an explicitly empty editor list.
	Editors *[]string `json:"editors,omitempty"`

	ListMetadata       *ListMetadata `json:"list_metadata,omitempty"`
	ListLimits         *ListLimits   `json:"list_limits,omitempty"`
	ListCSVDownloadURL string        `json:"list_csv_download_url,omitempty"`
	LastEditor         *string       `json:"last_editor,omitempty"`
	EditTimestamp      *JSONTime     `json:"edit_timestamp,omitempty"`
}

// FileUserAccess is a user's access to a file through a DM or group DM.
type FileUserAccess struct {
	UserID string `json:"user_id"`
	Access string `json:"access"`
}

// FileTranscription contains a file's transcription status and preview.
type FileTranscription struct {
	Status  string                    `json:"status,omitempty"`
	Locale  string                    `json:"locale,omitempty"`
	Preview *FileTranscriptionPreview `json:"preview,omitempty"`
}

// FileTranscriptionPreview is the available text of a file transcription.
type FileTranscriptionPreview struct {
	Content string `json:"content,omitempty"`
	HasMore *bool  `json:"has_more,omitempty"`
}

type EmailFileUserInfo struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Original string `json:"original"`
}

type EmailHeaders struct {
	Date      string `json:"date"`
	InReplyTo string `json:"in_reply_to"`
	ReplyTo   string `json:"reply_to"`
	MessageID string `json:"message_id"`
}

type Share struct {
	Public  map[string][]ShareFileInfo `json:"public"`
	Private map[string][]ShareFileInfo `json:"private"`
}

type ShareFileInfo struct {
	ReplyUsers      []string `json:"reply_users"`
	ReplyUsersCount int      `json:"reply_users_count"`
	ReplyCount      int      `json:"reply_count"`
	Ts              string   `json:"ts"`
	ThreadTs        string   `json:"thread_ts"`
	LatestReply     string   `json:"latest_reply"`
	ChannelName     string   `json:"channel_name"`
	TeamID          string   `json:"team_id"`
}

// GetFilesParameters contains all the parameters necessary (including the optional ones) for a GetFiles() request
type GetFilesParameters struct {
	User          string
	Channel       string
	TeamID        string
	TimestampFrom JSONTime
	TimestampTo   JSONTime
	Types         string
	Count         int
	Page          int
	ShowHidden    bool
}

// ListFilesParameters contains all the parameters necessary (including the optional ones) for a ListFiles() request
type ListFilesParameters struct {
	Limit   int
	User    string
	Channel string
	TeamID  string
	Types   string
	Cursor  string
}

type UploadFileParameters struct {
	File            string
	FileSize        int
	Content         string
	Reader          io.Reader
	Filename        string
	Title           string
	InitialComment  string
	Blocks          Blocks
	Channel         string
	Channels        []string
	ThreadTimestamp string
	AltTxt          string
	SnippetType     string
}

type GetUploadURLExternalParameters struct {
	AltTxt      string
	FileSize    int
	FileName    string
	SnippetType string
}

type GetUploadURLExternalResponse struct {
	UploadURL string `json:"upload_url"`
	FileID    string `json:"file_id"`
	SlackResponse
}

type UploadToURLParameters struct {
	UploadURL string
	Reader    io.Reader
	File      string
	Content   string
	Filename  string
}

type FileSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type CompleteUploadExternalParameters struct {
	Files           []FileSummary
	Blocks          Blocks
	Channel         string
	Channels        []string
	InitialComment  string
	ThreadTimestamp string
}

type CompleteUploadExternalResponse struct {
	SlackResponse
	Files []FileSummary `json:"files"`
}

type fileResponseFull struct {
	File     `json:"file"`
	Paging   `json:"paging"`
	Comments []Comment        `json:"comments"`
	Files    []File           `json:"files"`
	Metadata ResponseMetadata `json:"response_metadata"`

	SlackResponse
}

// NewGetFilesParameters provides an instance of GetFilesParameters with all the sane default values set
func NewGetFilesParameters() GetFilesParameters {
	return GetFilesParameters{
		User:          DEFAULT_FILES_USER,
		Channel:       DEFAULT_FILES_CHANNEL,
		TimestampFrom: DEFAULT_FILES_TS_FROM,
		TimestampTo:   DEFAULT_FILES_TS_TO,
		Types:         DEFAULT_FILES_TYPES,
		Count:         DEFAULT_FILES_COUNT,
		Page:          DEFAULT_FILES_PAGE,
		ShowHidden:    DEFAULT_FILES_SHOW_HIDDEN,
	}
}

func (api *Client) fileRequest(ctx context.Context, path string, values url.Values) (*fileResponseFull, error) {
	response := &fileResponseFull{}
	err := api.postMethod(ctx, path, values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// GetFileInfo retrieves a file and related comments.
// For more details, see GetFileInfoContext documentation.
func (api *Client) GetFileInfo(fileID string, count, page int) (*File, []Comment, *Paging, error) {
	return api.GetFileInfoContext(context.Background(), fileID, count, page)
}

// GetFileInfoContext retrieves a file and related comments with a custom context.
// Slack API docs: https://api.slack.com/methods/files.info
func (api *Client) GetFileInfoContext(ctx context.Context, fileID string, count, page int) (*File, []Comment, *Paging, error) {
	values := url.Values{
		"token": {api.token},
		"file":  {fileID},
		"count": {strconv.Itoa(count)},
		"page":  {strconv.Itoa(page)},
	}

	response, err := api.fileRequest(ctx, "files.info", values)
	if err != nil {
		return nil, nil, nil, err
	}
	return &response.File, response.Comments, &response.Paging, nil
}

// GetFile retrieves a given file from its private download URL.
//
// The request carries the token, so GetFile accepts only https URLs on slack.com or
// slack-gov.com (and their subdomains) and URLs on the host set with OptionAPIURL. It
// returns an error for any other URL. A remote or external file (File.IsExternal) has
// a url_private outside Slack: fetch it without the Slack client.
func (api *Client) GetFile(downloadURL string, writer io.Writer) error {
	return api.GetFileContext(context.Background(), downloadURL, writer)
}

// GetFileContext retrieves a given file from its private download URL with a custom context.
// For more details, see GetFile documentation.
func (api *Client) GetFileContext(ctx context.Context, downloadURL string, writer io.Writer) error {
	return downloadFile(ctx, api.httpclient, api.token, api.endpoint, downloadURL, writer, api)
}

// GetFiles retrieves all files according to the parameters given.
// For more details, see GetFilesContext documentation.
func (api *Client) GetFiles(params GetFilesParameters) ([]File, *Paging, error) {
	return api.GetFilesContext(context.Background(), params)
}

// GetFilesContext retrieves all files according to the parameters given with a custom context.
// Slack API docs: https://api.slack.com/methods/files.list
func (api *Client) GetFilesContext(ctx context.Context, params GetFilesParameters) ([]File, *Paging, error) {
	values := url.Values{
		"token": {api.token},
	}
	if params.User != DEFAULT_FILES_USER {
		values.Add("user", params.User)
	}
	if params.Channel != DEFAULT_FILES_CHANNEL {
		values.Add("channel", params.Channel)
	}
	if params.TeamID != "" {
		values.Add("team_id", params.TeamID)
	}
	if params.TimestampFrom != DEFAULT_FILES_TS_FROM {
		values.Add("ts_from", strconv.FormatInt(int64(params.TimestampFrom), 10))
	}
	if params.TimestampTo != DEFAULT_FILES_TS_TO {
		values.Add("ts_to", strconv.FormatInt(int64(params.TimestampTo), 10))
	}
	if params.Types != DEFAULT_FILES_TYPES {
		values.Add("types", params.Types)
	}
	if params.Count != DEFAULT_FILES_COUNT {
		values.Add("count", strconv.Itoa(params.Count))
	}
	if params.Page != DEFAULT_FILES_PAGE {
		values.Add("page", strconv.Itoa(params.Page))
	}
	//nolint:staticcheck // S1002: keep the explicit comparison against the constant.
	if params.ShowHidden != DEFAULT_FILES_SHOW_HIDDEN {
		values.Add("show_files_hidden_by_limit", strconv.FormatBool(params.ShowHidden))
	}

	response, err := api.fileRequest(ctx, "files.list", values)
	if err != nil {
		return nil, nil, err
	}
	return response.Files, &response.Paging, nil
}

// ListFiles retrieves a page of files using cursor-based pagination.
// For pagination caveats, see [Client.ListFilesContext].
func (api *Client) ListFiles(params ListFilesParameters) ([]File, *ListFilesParameters, error) {
	return api.ListFilesContext(context.Background(), params)
}

// ListFilesContext retrieves a page of files with a custom context.
// Its cursor and limit parameters are not documented in Slack's files.list reference.
// [Client.GetFilesContext] uses the documented count/page parameters.
//
// Slack API docs: https://docs.slack.dev/reference/methods/files.list
func (api *Client) ListFilesContext(ctx context.Context, params ListFilesParameters) ([]File, *ListFilesParameters, error) {
	values := url.Values{
		"token": {api.token},
	}

	if params.User != DEFAULT_FILES_USER {
		values.Add("user", params.User)
	}
	if params.Channel != DEFAULT_FILES_CHANNEL {
		values.Add("channel", params.Channel)
	}
	if params.TeamID != "" {
		values.Add("team_id", params.TeamID)
	}
	if params.Types != "" {
		values.Add("types", params.Types)
	}
	if params.Limit != DEFAULT_FILES_COUNT {
		values.Add("limit", strconv.Itoa(params.Limit))
	}
	if params.Cursor != "" {
		values.Add("cursor", params.Cursor)
	}

	response, err := api.fileRequest(ctx, "files.list", values)
	if err != nil {
		return nil, nil, err
	}

	params.Cursor = response.Metadata.Cursor

	return response.Files, &params, nil
}

// DeleteFileComment deletes a file's comment.
// For more details, see DeleteFileCommentContext documentation.
func (api *Client) DeleteFileComment(fileID, commentID string) error {
	return api.DeleteFileCommentContext(context.Background(), fileID, commentID)
}

// DeleteFileCommentContext deletes a file's comment with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/files.comments.delete
func (api *Client) DeleteFileCommentContext(ctx context.Context, fileID, commentID string) (err error) {
	if fileID == "" || commentID == "" {
		return ErrParametersMissing
	}
	// DeleteFileComment took (commentID, fileID) until it changed to match this
	// function. Comment IDs start with "Fc", so reject the old order instead of
	// sending swapped IDs.
	if strings.HasPrefix(fileID, "Fc") && !strings.HasPrefix(commentID, "Fc") {
		return fmt.Errorf("files.comments.delete: %q looks like a comment ID; the arguments are (fileID, commentID)", fileID)
	}

	values := url.Values{
		"token": {api.token},
		"file":  {fileID},
		"id":    {commentID},
	}
	_, err = api.fileRequest(ctx, "files.comments.delete", values)
	return err
}

// DeleteFile deletes a file.
// For more details, see DeleteFileContext documentation.
func (api *Client) DeleteFile(fileID string) error {
	return api.DeleteFileContext(context.Background(), fileID)
}

// DeleteFileContext deletes a file with a custom context.
// Slack API docs: https://api.slack.com/methods/files.delete
func (api *Client) DeleteFileContext(ctx context.Context, fileID string) (err error) {
	values := url.Values{
		"token": {api.token},
		"file":  {fileID},
	}

	_, err = api.fileRequest(ctx, "files.delete", values)
	return err
}

// RevokeFilePublicURL disables public/external sharing for a file.
// For more details, see RevokeFilePublicURLContext documentation.
func (api *Client) RevokeFilePublicURL(fileID string) (*File, error) {
	return api.RevokeFilePublicURLContext(context.Background(), fileID)
}

// RevokeFilePublicURLContext disables public/external sharing for a file with a custom context.
// Slack API docs: https://api.slack.com/methods/files.revokePublicURL
func (api *Client) RevokeFilePublicURLContext(ctx context.Context, fileID string) (*File, error) {
	values := url.Values{
		"token": {api.token},
		"file":  {fileID},
	}

	response, err := api.fileRequest(ctx, "files.revokePublicURL", values)
	if err != nil {
		return nil, err
	}
	return &response.File, nil
}

// ShareFilePublicURL enabled public/external sharing for a file.
// For more details, see ShareFilePublicURLContext documentation.
func (api *Client) ShareFilePublicURL(fileID string) (*File, []Comment, *Paging, error) {
	return api.ShareFilePublicURLContext(context.Background(), fileID)
}

// ShareFilePublicURLContext enabled public/external sharing for a file with a custom context.
// Slack API docs: https://api.slack.com/methods/files.sharedPublicURL
func (api *Client) ShareFilePublicURLContext(ctx context.Context, fileID string) (*File, []Comment, *Paging, error) {
	values := url.Values{
		"token": {api.token},
		"file":  {fileID},
	}

	response, err := api.fileRequest(ctx, "files.sharedPublicURL", values)
	if err != nil {
		return nil, nil, nil, err
	}
	return &response.File, response.Comments, &response.Paging, nil
}

// GetUploadURLExternalContext gets a URL and fileID from slack which can later be used to upload a file.
// Slack API docs: https://api.slack.com/methods/files.getUploadURLExternal
func (api *Client) GetUploadURLExternalContext(ctx context.Context, params GetUploadURLExternalParameters) (*GetUploadURLExternalResponse, error) {
	if params.FileName == "" {
		return nil, fmt.Errorf("FileName cannot be empty")
	}
	if params.FileSize == 0 {
		return nil, fmt.Errorf("FileSize cannot be 0")
	}

	values := url.Values{
		"token":    {api.token},
		"filename": {params.FileName},
		"length":   {strconv.Itoa(params.FileSize)},
	}
	if params.AltTxt != "" {
		values.Add("alt_txt", params.AltTxt)
	}
	if params.SnippetType != "" {
		values.Add("snippet_type", params.SnippetType)
	}
	response := &GetUploadURLExternalResponse{}
	err := api.postMethod(ctx, "files.getUploadURLExternal", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// UploadToURL uploads the file to the provided URL using post method
// This is not a Slack API method, but a helper function to upload files to the URL
//
// The request carries the token, so the URL must pass the same check as in GetFile.
func (api *Client) UploadToURL(ctx context.Context, params UploadToURLParameters) (err error) {
	if err := ensureURLMayReceiveToken(api.endpoint, params.UploadURL); err != nil {
		return err
	}
	values := url.Values{}
	switch {
	case params.Content != "":
		contentReader := strings.NewReader(params.Content)
		err = postWithMultipartResponse(ctx, api.httpclient, params.UploadURL, params.Filename, "file", api.token, values, contentReader, nil, api)
	case params.File != "":
		err = postLocalWithMultipartResponse(ctx, api.httpclient, params.UploadURL, params.File, "file", api.token, values, nil, api)
	case params.Reader != nil:
		err = postWithMultipartResponse(ctx, api.httpclient, params.UploadURL, params.Filename, "file", api.token, values, params.Reader, nil, api)
	}
	return err
}

// CompleteUploadExternalContext once files are uploaded, this completes the upload and shares it to the specified channels
// Slack API docs: https://api.slack.com/methods/files.completeUploadExternal
func (api *Client) CompleteUploadExternalContext(ctx context.Context, params CompleteUploadExternalParameters) (file *CompleteUploadExternalResponse, err error) {
	filesBytes, err := json.Marshal(params.Files)
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"token": {api.token},
		"files": {string(filesBytes)},
	}

	if params.Channel != "" {
		values.Add("channel_id", params.Channel)
	}
	if len(params.Channels) > 0 {
		values.Add("channels", strings.Join(params.Channels, ","))
	}
	if params.InitialComment != "" {
		values.Add("initial_comment", params.InitialComment)
	}
	if params.Blocks.BlockSet != nil && params.InitialComment == "" {
		blocksBytes, err := json.Marshal(params.Blocks)
		if err != nil {
			return nil, err
		}
		values.Add("blocks", string(blocksBytes))
	}
	if params.ThreadTimestamp != "" {
		values.Add("thread_ts", params.ThreadTimestamp)
	}
	response := &CompleteUploadExternalResponse{}
	err = api.postMethod(ctx, "files.completeUploadExternal", values, response)
	if err != nil {
		return nil, err
	}
	if response.Err() != nil {
		return nil, response.Err()
	}
	return response, nil
}

// UploadFile uploads file to a given slack channel using 3 steps.
// For more details, see UploadFileContext documentation.
func (api *Client) UploadFile(params UploadFileParameters) (*FileSummary, error) {
	return api.UploadFileContext(context.Background(), params)
}

// UploadFileContext uploads file to a given slack channel using 3 steps -
//  1. Get an upload URL using files.getUploadURLExternal API
//  2. Send the file as a post to the URL provided by slack
//  3. Complete the upload and share it to the specified channels using files.completeUploadExternal
//
// Slack Docs: https://api.slack.com/messaging/files#uploading_files
func (api *Client) UploadFileContext(ctx context.Context, params UploadFileParameters) (file *FileSummary, err error) {
	if params.Filename == "" {
		return nil, fmt.Errorf("file.upload.v2: filename cannot be empty")
	}
	if params.FileSize == 0 {
		return nil, fmt.Errorf("file.upload.v2: file size cannot be 0")
	}

	u, err := api.GetUploadURLExternalContext(ctx, GetUploadURLExternalParameters{
		AltTxt:      params.AltTxt,
		FileName:    params.Filename,
		FileSize:    params.FileSize,
		SnippetType: params.SnippetType,
	})
	if err != nil {
		return nil, fmt.Errorf("GetUploadURLExternal: %w", err)
	}

	err = api.UploadToURL(ctx, UploadToURLParameters{
		UploadURL: u.UploadURL,
		Reader:    params.Reader,
		File:      params.File,
		Content:   params.Content,
		Filename:  params.Filename,
	})
	if err != nil {
		return nil, fmt.Errorf("UploadToURL: %w", err)
	}

	c, err := api.CompleteUploadExternalContext(ctx, CompleteUploadExternalParameters{
		Files: []FileSummary{{
			ID:    u.FileID,
			Title: params.Title,
		}},
		Channel:         params.Channel,
		Channels:        params.Channels,
		InitialComment:  params.InitialComment,
		ThreadTimestamp: params.ThreadTimestamp,
		Blocks:          params.Blocks,
	})
	if err != nil {
		return nil, fmt.Errorf("CompleteUploadExternal: %w", err)
	}
	if len(c.Files) != 1 {
		return nil, fmt.Errorf("file.upload.v2: something went wrong; received %d files instead of 1", len(c.Files))
	}

	return &c.Files[0], nil
}
