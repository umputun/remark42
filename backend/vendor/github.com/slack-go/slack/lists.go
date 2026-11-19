package slack

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// CreateListResponse is the response from slackLists.create.
type CreateListResponse struct {
	SlackResponse
	ListID       string       `json:"list_id"`
	ListMetadata ListMetadata `json:"list_metadata"`
}

// StartListDownloadResponse is the response from slackLists.download.start.
type StartListDownloadResponse struct {
	SlackResponse
	JobID string `json:"job_id"`
}

// GetListDownloadResponse is the response from slackLists.download.get.
type GetListDownloadResponse struct {
	SlackResponse
	Status      string `json:"status"`
	DownloadURL string `json:"download_url"`
}

// CreateListItemResponse is the response from slackLists.items.create.
type CreateListItemResponse struct {
	SlackResponse
	Item ListItem `json:"item"`
}

// GetListItemInfoResponse is the response from slackLists.items.info.
type GetListItemInfoResponse struct {
	SlackResponse
	List     List       `json:"list"`
	Record   ListItem   `json:"record"`
	Subtasks []ListItem `json:"subtasks"`
}

// GetListItemsResponse is the response from slackLists.items.list.
type GetListItemsResponse struct {
	SlackResponse
	Items []ListItem `json:"items"`
	List  *List      `json:"list,omitempty"`
}

type createListParams struct {
	descriptionBlocks        []RichTextBlock
	schema                   []ListColumn
	copyFromListID           string
	includeCopiedListRecords *bool
	todoMode                 *bool
}

// CreateListOption is an option for CreateList.
type CreateListOption func(*createListParams)

// CreateListOptionDescriptionBlocks sets the rich text description.
func CreateListOptionDescriptionBlocks(descriptionBlocks []RichTextBlock) CreateListOption {
	return func(params *createListParams) {
		params.descriptionBlocks = descriptionBlocks
	}
}

// CreateListOptionSchema sets the column definitions.
func CreateListOptionSchema(schema []ListColumn) CreateListOption {
	return func(params *createListParams) {
		params.schema = schema
	}
}

// CreateListOptionCopyFromListID sets the List to copy.
func CreateListOptionCopyFromListID(copyFromListID string) CreateListOption {
	return func(params *createListParams) {
		params.copyFromListID = copyFromListID
	}
}

// CreateListOptionIncludeCopiedListRecords sets whether to copy the records.
func CreateListOptionIncludeCopiedListRecords(includeCopiedListRecords bool) CreateListOption {
	return func(params *createListParams) {
		params.includeCopiedListRecords = &includeCopiedListRecords
	}
}

// CreateListOptionTodoMode sets whether to track tasks.
func CreateListOptionTodoMode(todoMode bool) CreateListOption {
	return func(params *createListParams) {
		params.todoMode = &todoMode
	}
}

// CreateList creates a Slack List.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.create
func (api *Client) CreateList(name string, options ...CreateListOption) (*CreateListResponse, error) {
	return api.CreateListContext(context.Background(), name, options...)
}

// CreateListContext creates a Slack List with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.create
func (api *Client) CreateListContext(ctx context.Context, name string, options ...CreateListOption) (*CreateListResponse, error) {
	params := createListParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token": {api.token},
		"name":  {name},
	}

	if params.descriptionBlocks != nil {
		descriptionBlocksJSON, err := json.Marshal(params.descriptionBlocks)
		if err != nil {
			return nil, err
		}
		values.Add("description_blocks", string(descriptionBlocksJSON))
	}

	if params.schema != nil {
		schemaJSON, err := json.Marshal(params.schema)
		if err != nil {
			return nil, err
		}
		values.Add("schema", string(schemaJSON))
	}

	if params.copyFromListID != "" {
		values.Add("copy_from_list_id", params.copyFromListID)
	}

	if params.includeCopiedListRecords != nil {
		values.Add("include_copied_list_records", strconv.FormatBool(*params.includeCopiedListRecords))
	}

	if params.todoMode != nil {
		values.Add("todo_mode", strconv.FormatBool(*params.todoMode))
	}

	response := &CreateListResponse{}
	if err := api.postMethod(ctx, "slackLists.create", values, response); err != nil {
		return nil, err
	}

	return response, response.Err()
}

type updateListParams struct {
	name              *string
	descriptionBlocks []RichTextBlock
	todoMode          *bool
}

// UpdateListOption is an option for UpdateList.
type UpdateListOption func(*updateListParams)

// UpdateListOptionName sets the name.
func UpdateListOptionName(name string) UpdateListOption {
	return func(params *updateListParams) {
		params.name = &name
	}
}

// UpdateListOptionDescriptionBlocks sets the description; an empty slice clears it.
func UpdateListOptionDescriptionBlocks(descriptionBlocks []RichTextBlock) UpdateListOption {
	return func(params *updateListParams) {
		params.descriptionBlocks = descriptionBlocks
	}
}

// UpdateListOptionTodoMode sets whether to track tasks.
func UpdateListOptionTodoMode(todoMode bool) UpdateListOption {
	return func(params *updateListParams) {
		params.todoMode = &todoMode
	}
}

// UpdateList updates a Slack List.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.update
func (api *Client) UpdateList(listID string, options ...UpdateListOption) error {
	return api.UpdateListContext(context.Background(), listID, options...)
}

// UpdateListContext updates a Slack List with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.update
func (api *Client) UpdateListContext(ctx context.Context, listID string, options ...UpdateListOption) error {
	params := updateListParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token": {api.token},
		"id":    {listID},
	}

	if params.name != nil {
		values.Add("name", *params.name)
	}

	if params.descriptionBlocks != nil {
		descriptionBlocksJSON, err := json.Marshal(params.descriptionBlocks)
		if err != nil {
			return err
		}
		values.Add("description_blocks", string(descriptionBlocksJSON))
	}

	if params.todoMode != nil {
		values.Add("todo_mode", strconv.FormatBool(*params.todoMode))
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "slackLists.update", values, response); err != nil {
		return err
	}

	return response.Err()
}

type setListAccessParams struct {
	channelIDs []string
	userIDs    []string
}

// SetListAccessOption is an option for SetListAccess.
type SetListAccessOption func(*setListAccessParams)

// SetListAccessOptionChannelIDs sets the channels to grant access; use this or UserIDs.
func SetListAccessOptionChannelIDs(channelIDs []string) SetListAccessOption {
	return func(params *setListAccessParams) {
		params.channelIDs = channelIDs
	}
}

// SetListAccessOptionUserIDs sets the users to grant access; use this or ChannelIDs.
func SetListAccessOptionUserIDs(userIDs []string) SetListAccessOption {
	return func(params *setListAccessParams) {
		params.userIDs = userIDs
	}
}

// SetListAccess sets access to a Slack List.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.access.set
func (api *Client) SetListAccess(listID string, accessLevel ListAccessLevel, options ...SetListAccessOption) error {
	return api.SetListAccessContext(context.Background(), listID, accessLevel, options...)
}

// SetListAccessContext sets access to a Slack List with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.access.set
func (api *Client) SetListAccessContext(ctx context.Context, listID string, accessLevel ListAccessLevel, options ...SetListAccessOption) error {
	params := setListAccessParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token":        {api.token},
		"list_id":      {listID},
		"access_level": {string(accessLevel)},
	}

	if params.channelIDs != nil {
		channelIDsJSON, err := json.Marshal(params.channelIDs)
		if err != nil {
			return err
		}
		values.Add("channel_ids", string(channelIDsJSON))
	}

	if params.userIDs != nil {
		userIDsJSON, err := json.Marshal(params.userIDs)
		if err != nil {
			return err
		}
		values.Add("user_ids", string(userIDsJSON))
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "slackLists.access.set", values, response); err != nil {
		return err
	}

	return response.Err()
}

type deleteListAccessParams struct {
	channelIDs []string
	userIDs    []string
}

// DeleteListAccessOption is an option for DeleteListAccess.
type DeleteListAccessOption func(*deleteListAccessParams)

// DeleteListAccessOptionChannelIDs sets the channels to revoke access from; use this or UserIDs.
func DeleteListAccessOptionChannelIDs(channelIDs []string) DeleteListAccessOption {
	return func(params *deleteListAccessParams) {
		params.channelIDs = channelIDs
	}
}

// DeleteListAccessOptionUserIDs sets the users to revoke access from; use this or ChannelIDs.
func DeleteListAccessOptionUserIDs(userIDs []string) DeleteListAccessOption {
	return func(params *deleteListAccessParams) {
		params.userIDs = userIDs
	}
}

// DeleteListAccess removes access to a Slack List.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.access.delete
func (api *Client) DeleteListAccess(listID string, options ...DeleteListAccessOption) error {
	return api.DeleteListAccessContext(context.Background(), listID, options...)
}

// DeleteListAccessContext removes access to a Slack List with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.access.delete
func (api *Client) DeleteListAccessContext(ctx context.Context, listID string, options ...DeleteListAccessOption) error {
	params := deleteListAccessParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token":   {api.token},
		"list_id": {listID},
	}

	if params.channelIDs != nil {
		channelIDsJSON, err := json.Marshal(params.channelIDs)
		if err != nil {
			return err
		}
		values.Add("channel_ids", string(channelIDsJSON))
	}

	if params.userIDs != nil {
		userIDsJSON, err := json.Marshal(params.userIDs)
		if err != nil {
			return err
		}
		values.Add("user_ids", string(userIDsJSON))
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "slackLists.access.delete", values, response); err != nil {
		return err
	}

	return response.Err()
}

type startListDownloadParams struct {
	includeArchived    *bool
	format             ListDownloadFormat
	includeThreads     *bool
	includeAttachments *bool
}

// StartListDownloadOption is an option for StartListDownload.
type StartListDownloadOption func(*startListDownloadParams)

// StartListDownloadOptionIncludeArchived sets whether to include archived items.
func StartListDownloadOptionIncludeArchived(includeArchived bool) StartListDownloadOption {
	return func(params *startListDownloadParams) {
		params.includeArchived = &includeArchived
	}
}

// StartListDownloadOptionFormat sets the export format.
func StartListDownloadOptionFormat(format ListDownloadFormat) StartListDownloadOption {
	return func(params *startListDownloadParams) {
		params.format = format
	}
}

// StartListDownloadOptionIncludeThreads sets whether a JSON export includes item threads.
func StartListDownloadOptionIncludeThreads(includeThreads bool) StartListDownloadOption {
	return func(params *startListDownloadParams) {
		params.includeThreads = &includeThreads
	}
}

// StartListDownloadOptionIncludeAttachments sets whether a JSON export includes attachment metadata.
func StartListDownloadOptionIncludeAttachments(includeAttachments bool) StartListDownloadOption {
	return func(params *startListDownloadParams) {
		params.includeAttachments = &includeAttachments
	}
}

// StartListDownload starts a List export.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.download.start
func (api *Client) StartListDownload(listID string, options ...StartListDownloadOption) (*StartListDownloadResponse, error) {
	return api.StartListDownloadContext(context.Background(), listID, options...)
}

// StartListDownloadContext starts a List export with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.download.start
func (api *Client) StartListDownloadContext(ctx context.Context, listID string, options ...StartListDownloadOption) (*StartListDownloadResponse, error) {
	params := startListDownloadParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token":   {api.token},
		"list_id": {listID},
	}

	if params.includeArchived != nil {
		values.Add("include_archived", strconv.FormatBool(*params.includeArchived))
	}

	if params.format != "" {
		values.Add("format", string(params.format))
	}

	if params.includeThreads != nil {
		values.Add("include_threads", strconv.FormatBool(*params.includeThreads))
	}

	if params.includeAttachments != nil {
		values.Add("include_attachments", strconv.FormatBool(*params.includeAttachments))
	}

	response := &StartListDownloadResponse{}
	if err := api.postMethod(ctx, "slackLists.download.start", values, response); err != nil {
		return nil, err
	}

	return response, response.Err()
}

type getListDownloadParams struct {
	format             ListDownloadFormat
	includeThreads     *bool
	includeAttachments *bool
}

// GetListDownloadOption is an option for GetListDownload.
type GetListDownloadOption func(*getListDownloadParams)

// GetListDownloadOptionFormat sets the format used to start the export.
func GetListDownloadOptionFormat(format ListDownloadFormat) GetListDownloadOption {
	return func(params *getListDownloadParams) {
		params.format = format
	}
}

// GetListDownloadOptionIncludeThreads sets the include_threads value used to start the export.
func GetListDownloadOptionIncludeThreads(includeThreads bool) GetListDownloadOption {
	return func(params *getListDownloadParams) {
		params.includeThreads = &includeThreads
	}
}

// GetListDownloadOptionIncludeAttachments sets the include_attachments value used to start the export.
func GetListDownloadOptionIncludeAttachments(includeAttachments bool) GetListDownloadOption {
	return func(params *getListDownloadParams) {
		params.includeAttachments = &includeAttachments
	}
}

// GetListDownload gets a List export status and download URL.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.download.get
func (api *Client) GetListDownload(listID string, jobID string, options ...GetListDownloadOption) (*GetListDownloadResponse, error) {
	return api.GetListDownloadContext(context.Background(), listID, jobID, options...)
}

// GetListDownloadContext gets a List export status and download URL with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.download.get
func (api *Client) GetListDownloadContext(ctx context.Context, listID string, jobID string, options ...GetListDownloadOption) (*GetListDownloadResponse, error) {
	params := getListDownloadParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token":   {api.token},
		"list_id": {listID},
		"job_id":  {jobID},
	}

	if params.format != "" {
		values.Add("format", string(params.format))
	}

	if params.includeThreads != nil {
		values.Add("include_threads", strconv.FormatBool(*params.includeThreads))
	}

	if params.includeAttachments != nil {
		values.Add("include_attachments", strconv.FormatBool(*params.includeAttachments))
	}

	response := &GetListDownloadResponse{}
	if err := api.postMethod(ctx, "slackLists.download.get", values, response); err != nil {
		return nil, err
	}

	return response, response.Err()
}

type createListItemParams struct {
	duplicatedItemID string
	parentItemID     string
	initialFields    []ListItemField
}

// CreateListItemOption is an option for CreateListItem.
type CreateListItemOption func(*createListItemParams)

// CreateListItemOptionDuplicatedItemID sets the item to copy.
func CreateListItemOptionDuplicatedItemID(duplicatedItemID string) CreateListItemOption {
	return func(params *createListItemParams) {
		params.duplicatedItemID = duplicatedItemID
	}
}

// CreateListItemOptionParentItemID sets the parent item for a subtask.
func CreateListItemOptionParentItemID(parentItemID string) CreateListItemOption {
	return func(params *createListItemParams) {
		params.parentItemID = parentItemID
	}
}

// CreateListItemOptionInitialFields sets the initial cell values.
func CreateListItemOptionInitialFields(initialFields []ListItemField) CreateListItemOption {
	return func(params *createListItemParams) {
		params.initialFields = initialFields
	}
}

// CreateListItem creates an item in a Slack List.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.create
func (api *Client) CreateListItem(listID string, options ...CreateListItemOption) (*CreateListItemResponse, error) {
	return api.CreateListItemContext(context.Background(), listID, options...)
}

// CreateListItemContext creates an item in a Slack List with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.create
func (api *Client) CreateListItemContext(ctx context.Context, listID string, options ...CreateListItemOption) (*CreateListItemResponse, error) {
	params := createListItemParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token":   {api.token},
		"list_id": {listID},
	}

	if params.duplicatedItemID != "" {
		values.Add("duplicated_item_id", params.duplicatedItemID)
	}

	if params.parentItemID != "" {
		values.Add("parent_item_id", params.parentItemID)
	}

	if params.initialFields != nil {
		initialFieldsJSON, err := json.Marshal(params.initialFields)
		if err != nil {
			return nil, err
		}
		values.Add("initial_fields", string(initialFieldsJSON))
	}

	response := &CreateListItemResponse{}
	if err := api.postMethod(ctx, "slackLists.items.create", values, response); err != nil {
		return nil, err
	}

	return response, response.Err()
}

// DeleteListItem deletes an item from a Slack List.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.delete
func (api *Client) DeleteListItem(listID string, itemID string) error {
	return api.DeleteListItemContext(context.Background(), listID, itemID)
}

// DeleteListItemContext deletes an item from a Slack List with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.delete
func (api *Client) DeleteListItemContext(ctx context.Context, listID string, itemID string) error {
	values := url.Values{
		"token":   {api.token},
		"list_id": {listID},
		"id":      {itemID},
	}

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "slackLists.items.delete", values, response); err != nil {
		return err
	}

	return response.Err()
}

// DeleteListItems deletes multiple items from a Slack List.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.deleteMultiple
func (api *Client) DeleteListItems(listID string, itemIDs []string) error {
	return api.DeleteListItemsContext(context.Background(), listID, itemIDs)
}

// DeleteListItemsContext deletes multiple items from a Slack List with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.deleteMultiple
func (api *Client) DeleteListItemsContext(ctx context.Context, listID string, itemIDs []string) error {
	values := url.Values{
		"token":   {api.token},
		"list_id": {listID},
	}

	itemIDsJSON, err := json.Marshal(itemIDs)
	if err != nil {
		return err
	}
	values.Add("ids", string(itemIDsJSON))

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "slackLists.items.deleteMultiple", values, response); err != nil {
		return err
	}

	return response.Err()
}

type getListItemInfoParams struct {
	includeIsSubscribed *bool
}

// GetListItemInfoOption is an option for GetListItemInfo.
type GetListItemInfoOption func(*getListItemInfoParams)

// GetListItemInfoOptionIncludeIsSubscribed sets whether to include subscription status.
func GetListItemInfoOptionIncludeIsSubscribed(includeIsSubscribed bool) GetListItemInfoOption {
	return func(params *getListItemInfoParams) {
		params.includeIsSubscribed = &includeIsSubscribed
	}
}

// GetListItemInfo gets an item, its subtasks, and its List.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.info
func (api *Client) GetListItemInfo(listID string, itemID string, options ...GetListItemInfoOption) (*GetListItemInfoResponse, error) {
	return api.GetListItemInfoContext(context.Background(), listID, itemID, options...)
}

// GetListItemInfoContext gets an item, its subtasks, and its List with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.info
func (api *Client) GetListItemInfoContext(ctx context.Context, listID string, itemID string, options ...GetListItemInfoOption) (*GetListItemInfoResponse, error) {
	params := getListItemInfoParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token":   {api.token},
		"list_id": {listID},
		"id":      {itemID},
	}

	if params.includeIsSubscribed != nil {
		values.Add("include_is_subscribed", strconv.FormatBool(*params.includeIsSubscribed))
	}

	response := &GetListItemInfoResponse{}
	if err := api.postMethod(ctx, "slackLists.items.info", values, response); err != nil {
		return nil, err
	}

	return response, response.Err()
}

type getListItemsParams struct {
	limit       int
	cursor      string
	archived    *bool
	includeList *bool
}

// GetListItemsOption is an option for GetListItems.
type GetListItemsOption func(*getListItemsParams)

// GetListItemsOptionLimit sets the maximum number of items to return.
func GetListItemsOptionLimit(limit int) GetListItemsOption {
	return func(params *getListItemsParams) {
		params.limit = limit
	}
}

// GetListItemsOptionCursor sets the pagination cursor.
func GetListItemsOptionCursor(cursor string) GetListItemsOption {
	return func(params *getListItemsParams) {
		params.cursor = cursor
	}
}

// GetListItemsOptionArchived selects archived or active items.
func GetListItemsOptionArchived(archived bool) GetListItemsOption {
	return func(params *getListItemsParams) {
		params.archived = &archived
	}
}

// GetListItemsOptionIncludeList sets whether to return the parent List.
func GetListItemsOptionIncludeList(includeList bool) GetListItemsOption {
	return func(params *getListItemsParams) {
		params.includeList = &includeList
	}
}

// GetListItems gets a page of items from a Slack List.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.list
func (api *Client) GetListItems(listID string, options ...GetListItemsOption) (*GetListItemsResponse, error) {
	return api.GetListItemsContext(context.Background(), listID, options...)
}

// GetListItemsContext gets a page of items from a Slack List with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.list
func (api *Client) GetListItemsContext(ctx context.Context, listID string, options ...GetListItemsOption) (*GetListItemsResponse, error) {
	params := getListItemsParams{}
	for _, option := range options {
		if option != nil {
			option(&params)
		}
	}

	values := url.Values{
		"token":   {api.token},
		"list_id": {listID},
	}

	if params.limit > 0 {
		values.Add("limit", strconv.Itoa(params.limit))
	}

	if params.cursor != "" {
		values.Add("cursor", params.cursor)
	}

	if params.archived != nil {
		values.Add("archived", strconv.FormatBool(*params.archived))
	}

	if params.includeList != nil {
		values.Add("include_list", strconv.FormatBool(*params.includeList))
	}

	response := &GetListItemsResponse{}
	if err := api.postMethod(ctx, "slackLists.items.list", values, response); err != nil {
		return nil, err
	}

	return response, response.Err()
}

// UpdateListItems updates cells in a Slack List.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.update
func (api *Client) UpdateListItems(listID string, cells []ListItemCell) error {
	return api.UpdateListItemsContext(context.Background(), listID, cells)
}

// UpdateListItemsContext updates cells in a Slack List with a custom context.
//
// Slack API docs: https://docs.slack.dev/reference/methods/slackLists.items.update
func (api *Client) UpdateListItemsContext(ctx context.Context, listID string, cells []ListItemCell) error {
	values := url.Values{
		"token":   {api.token},
		"list_id": {listID},
	}

	cellsJSON, err := json.Marshal(cells)
	if err != nil {
		return err
	}
	values.Add("cells", string(cellsJSON))

	response := &SlackResponse{}
	if err := api.postMethod(ctx, "slackLists.items.update", values, response); err != nil {
		return err
	}

	return response.Err()
}
