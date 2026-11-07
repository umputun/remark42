package slack

import "encoding/json"

// ListAccessLevel is an access level for a Slack List.
type ListAccessLevel string

const (
	ListAccessRead  ListAccessLevel = "read"
	ListAccessWrite ListAccessLevel = "write"
	ListAccessOwner ListAccessLevel = "owner"
)

// ListDownloadFormat is a List export format.
type ListDownloadFormat string

const (
	ListDownloadCSV  ListDownloadFormat = "csv"
	ListDownloadJSON ListDownloadFormat = "json"
)

// ListColumnType is a List column type.
type ListColumnType string

const (
	ListColumnText           ListColumnType = "text"
	ListColumnRichText       ListColumnType = "rich_text"
	ListColumnMessage        ListColumnType = "message"
	ListColumnNumber         ListColumnType = "number"
	ListColumnSelect         ListColumnType = "select"
	ListColumnMultiSelect    ListColumnType = "multi_select"
	ListColumnDate           ListColumnType = "date"
	ListColumnUser           ListColumnType = "user"
	ListColumnAttachment     ListColumnType = "attachment"
	ListColumnCheckbox       ListColumnType = "checkbox"
	ListColumnEmail          ListColumnType = "email"
	ListColumnPhone          ListColumnType = "phone"
	ListColumnChannel        ListColumnType = "channel"
	ListColumnRating         ListColumnType = "rating"
	ListColumnCreatedBy      ListColumnType = "created_by"
	ListColumnLastEditedBy   ListColumnType = "last_edited_by"
	ListColumnCreatedTime    ListColumnType = "created_time"
	ListColumnLastEditedTime ListColumnType = "last_edited_time"
	ListColumnVote           ListColumnType = "vote"
	ListColumnCanvas         ListColumnType = "canvas"
	ListColumnReference      ListColumnType = "reference"
	ListColumnLink           ListColumnType = "link"
	ListColumnAssignee       ListColumnType = "assignee"
	ListColumnDueDate        ListColumnType = "due_date"
	ListColumnCompleted      ListColumnType = "completed"
	ListColumnTodoAssignee   ListColumnType = "todo_assignee"
	ListColumnTodoDueDate    ListColumnType = "todo_due_date"
	ListColumnTodoCompleted  ListColumnType = "todo_completed"
)

// ListColumnFormat controls selection in a List column.
type ListColumnFormat string

const (
	ListColumnFormatSingleSelect ListColumnFormat = "single_select"
	ListColumnFormatMultiSelect  ListColumnFormat = "multi_select"
	ListColumnFormatSingleEntity ListColumnFormat = "single_entity"
	ListColumnFormatMultiEntity  ListColumnFormat = "multi_entity"
)

// ListDateFormat controls how a List date is displayed.
type ListDateFormat string

const (
	ListDateFormatDefault          ListDateFormat = "default"
	ListDateFormatDayMonthYear     ListDateFormat = "DD/MM/YYYY"
	ListDateFormatMonthDayYear     ListDateFormat = "MM/DD/YYYY"
	ListDateFormatYearMonthDay     ListDateFormat = "YYYY/MM/DD"
	ListDateFormatLongMonthDayYear ListDateFormat = "MMMM DD, YYYY"
	ListDateFormatDayLongMonthYear ListDateFormat = "DD MMMM YYYY"
)

// ListChoiceColor is the color of a select option.
type ListChoiceColor string

const (
	ListChoiceIndigo ListChoiceColor = "indigo"
	ListChoiceBlue   ListChoiceColor = "blue"
	ListChoiceCyan   ListChoiceColor = "cyan"
	ListChoicePink   ListChoiceColor = "pink"
	ListChoiceYellow ListChoiceColor = "yellow"
	ListChoiceGreen  ListChoiceColor = "green"
	ListChoiceGray   ListChoiceColor = "gray"
	ListChoiceRed    ListChoiceColor = "red"
	ListChoicePurple ListChoiceColor = "purple"
	ListChoiceOrange ListChoiceColor = "orange"
	ListChoiceBrown  ListChoiceColor = "brown"
)

// ListColumn defines a List column. Slack assigns ID in responses.
type ListColumn struct {
	Key             string             `json:"key"`
	Name            string             `json:"name"`
	Type            ListColumnType     `json:"type"`
	ID              string             `json:"id,omitempty"`
	IsPrimaryColumn *bool              `json:"is_primary_column,omitempty"`
	Options         *ListColumnOptions `json:"options,omitempty"`
}

// ListColumnOptions configures a List column.
type ListColumnOptions struct {
	Choices           []ListColumnChoice      `json:"choices,omitempty"`
	Format            ListColumnFormat        `json:"format,omitempty"`
	Precision         *int                    `json:"precision,omitempty"`
	DateFormat        ListDateFormat          `json:"date_format,omitempty"`
	Emoji             string                  `json:"emoji,omitempty"`
	EmojiTeamID       string                  `json:"emoji_team_id,omitempty"`
	Max               *int                    `json:"max,omitempty"`
	DefaultValueTyped *ListColumnDefaultValue `json:"default_value_typed,omitempty"`
	DefaultValue      json.RawMessage         `json:"default_value,omitempty"`
	ShowMemberName    *bool                   `json:"show_member_name,omitempty"`
	NotifyUsers       *bool                   `json:"notify_users,omitempty"`
}

// ListColumnChoice is an option in a select column.
type ListColumnChoice struct {
	Value string          `json:"value"`
	Label string          `json:"label"`
	Color ListChoiceColor `json:"color"`
}

// ListColumnDefaultValue contains typed defaults for a column.
type ListColumnDefaultValue struct {
	User    []string `json:"user,omitempty"`
	Channel []string `json:"channel,omitempty"`
	Select  []string `json:"select,omitempty"`
}

// ListItemField supplies a cell value when creating or updating an item.
// Set one value field. Nil slices omit the value; empty slices send an empty
// array to clear it. Text cells use RichText, even for plain text.
type ListItemField struct {
	ColumnID   string              `json:"column_id"`
	RichText   []RichTextBlock     `json:"rich_text,omitzero"`
	User       []string            `json:"user,omitzero"`
	Date       []string            `json:"date,omitzero"`
	Select     []string            `json:"select,omitzero"`
	Checkbox   *bool               `json:"checkbox,omitempty"`
	Number     []float64           `json:"number,omitzero"`
	Email      []string            `json:"email,omitzero"`
	Phone      []string            `json:"phone,omitzero"`
	Attachment []string            `json:"attachment,omitzero"`
	Link       []ListItemLink      `json:"link,omitzero"`
	Message    []string            `json:"message,omitzero"`
	Rating     []int               `json:"rating,omitzero"`
	Timestamp  []int64             `json:"timestamp,omitzero"`
	Channel    []string            `json:"channel,omitzero"`
	Reference  []ListItemReference `json:"reference,omitzero"`
}

// ListItemLink supplies a link cell value.
type ListItemLink struct {
	OriginalURL  string `json:"original_url"`
	DisplayAsURL bool   `json:"display_as_url"`
	DisplayName  string `json:"display_name"`
}

// ListItemCell identifies a cell to update.
type ListItemCell struct {
	ListItemField
	RowID         string `json:"row_id,omitempty"`
	RowIDToCreate bool   `json:"row_id_to_create,omitempty"`
}

// ListItemFieldValue is a returned cell, including its formatted values.
// Value keeps Slack's legacy boolean, string, number, or null value as JSON.
type ListItemFieldValue struct {
	ColumnID   string              `json:"column_id,omitempty"`
	Key        string              `json:"key,omitempty"`
	Value      json.RawMessage     `json:"value,omitempty"`
	Text       string              `json:"text,omitempty"`
	RichText   []RichTextBlock     `json:"rich_text,omitempty"`
	User       []string            `json:"user,omitempty"`
	Date       []string            `json:"date,omitempty"`
	Select     []string            `json:"select,omitempty"`
	Checkbox   *bool               `json:"checkbox,omitempty"`
	Number     []float64           `json:"number,omitempty"`
	Email      []string            `json:"email,omitempty"`
	Phone      []string            `json:"phone,omitempty"`
	Attachment []string            `json:"attachment,omitempty"`
	Link       []ListItemLinkValue `json:"link,omitempty"`
	Message    []ListItemMessage   `json:"message,omitempty"`
	Rating     []int               `json:"rating,omitempty"`
	Timestamp  []int64             `json:"timestamp,omitempty"`
	Channel    []string            `json:"channel,omitempty"`
	Reference  []ListItemReference `json:"reference,omitempty"`
}

// ListItemLinkValue is a link returned in a cell. Its JSON keys differ from requests.
type ListItemLinkValue struct {
	OriginalURL  string          `json:"originalUrl"`
	Attachment   json.RawMessage `json:"attachment,omitempty"`
	DisplayAsURL *bool           `json:"displayAsUrl,omitempty"`
	DisplayName  string          `json:"displayName,omitempty"`
}

// ListItemMessage identifies a message returned in a cell or reference.
type ListItemMessage struct {
	Value           string `json:"value,omitempty"`
	ChannelID       string `json:"channel_id"`
	Timestamp       string `json:"ts"`
	ThreadTimestamp string `json:"thread_ts,omitempty"`
}

// ListItemReference refers to a Slack entity. Requests support File references.
type ListItemReference struct {
	File          *ListItemFileReference   `json:"file,omitempty"`
	Message       *ListItemMessage         `json:"message,omitempty"`
	ListRecord    *ListItemRecordReference `json:"list_record,omitempty"`
	CanvasSection *ListItemCanvasReference `json:"canvas_section,omitempty"`
}

// ListItemFileReference identifies a file.
type ListItemFileReference struct {
	FileID string `json:"file_id"`
}

// ListItemRecordReference identifies a List item.
type ListItemRecordReference struct {
	ListID string `json:"list_id"`
	RowID  string `json:"row_id"`
}

// ListItemCanvasReference identifies a canvas section.
type ListItemCanvasReference struct {
	FileID    string `json:"file_id"`
	SectionID string `json:"section_id"`
}

// ListItem is a row in a Slack List.
type ListItem struct {
	ID               string                      `json:"id"`
	ListID           string                      `json:"list_id"`
	DateCreated      int64                       `json:"date_created"`
	CreatedBy        string                      `json:"created_by,omitempty"`
	UpdatedBy        string                      `json:"updated_by,omitempty"`
	UpdatedTimestamp string                      `json:"updated_timestamp,omitempty"`
	Fields           []ListItemFieldValue        `json:"fields"`
	ParentRecordID   string                      `json:"parent_record_id,omitempty"`
	Archived         bool                        `json:"archived,omitempty"`
	IsSubscribed     *bool                       `json:"is_subscribed,omitempty"`
	Saved            *ListItemReminder           `json:"saved,omitempty"`
	SavedFields      map[string]ListItemReminder `json:"saved_fields,omitempty"`
}

// ListItemReminderState is the state of a List reminder.
type ListItemReminderState string

const (
	ListItemReminderArchived   ListItemReminderState = "archived"
	ListItemReminderInProgress ListItemReminderState = "in_progress"
	ListItemReminderCompleted  ListItemReminderState = "completed"
)

// ListItemReminder is reminder information for an item or cell.
type ListItemReminder struct {
	IsArchived    bool                  `json:"is_archived"`
	DateDue       int64                 `json:"date_due,omitempty"`
	DateCompleted int64                 `json:"date_completed,omitempty"`
	State         ListItemReminderState `json:"state"`
}

// List is a Slack file with List metadata and limits.
type List struct {
	File
	UserTeam                   string          `json:"user_team"`
	DisplayAsBot               bool            `json:"display_as_bot"`
	Username                   string          `json:"username"`
	ListMetadata               ListMetadata    `json:"list_metadata"`
	ListLimits                 ListLimits      `json:"list_limits"`
	LastEditor                 string          `json:"last_editor"`
	ListCSVDownloadURL         string          `json:"list_csv_download_url"`
	Updated                    JSONTime        `json:"updated"`
	SkippedShares              bool            `json:"skipped_shares"`
	TeamsSharedWith            []string        `json:"teams_shared_with"`
	IsRestrictedSharingEnabled bool            `json:"is_restricted_sharing_enabled"`
	HasRichPreview             bool            `json:"has_rich_preview"`
	FileAccess                 string          `json:"file_access"`
	Access                     ListAccessLevel `json:"access"`
	OrgOrWorkspaceAccess       string          `json:"org_or_workspace_access"`
	IsAISuggested              bool            `json:"is_ai_suggested"`
	IsModifiedByAI             bool            `json:"is_modified_by_ai"`
}

// ListMetadata describes a List's schema, views, and configuration.
type ListMetadata struct {
	Schema            []ListColumn        `json:"schema"`
	SubtaskSchema     []ListColumn        `json:"subtask_schema"`
	Views             []ListView          `json:"views"`
	Integrations      []json.RawMessage   `json:"integrations"`
	Icon              string              `json:"icon"`
	Description       string              `json:"description"`
	DescriptionBlocks []RichTextBlock     `json:"description_blocks"`
	IsTrial           bool                `json:"is_trial"`
	CreationSource    *ListCreationSource `json:"creation_source,omitempty"`
	TodoMode          bool                `json:"todo_mode"`
	DefaultView       string              `json:"default_view"`
}

// ListCreationSource identifies the source of a copied List.
type ListCreationSource struct {
	Type        string `json:"type"`
	ReferenceID string `json:"reference_id"`
}

// ListView describes a List view.
type ListView struct {
	ID                 string           `json:"id"`
	Name               string           `json:"name"`
	Type               string           `json:"type"`
	IsLocked           bool             `json:"is_locked"`
	Position           string           `json:"position"`
	Columns            []ListViewColumn `json:"columns"`
	DateCreated        int64            `json:"date_created"`
	CreatedBy          string           `json:"created_by"`
	StickColumnLeft    bool             `json:"stick_column_left"`
	IsAllItemsView     bool             `json:"is_all_items_view"`
	DefaultViewKey     string           `json:"default_view_key,omitempty"`
	ShowCompletedItems *bool            `json:"show_completed_items,omitempty"`
}

// ListViewColumn controls a column's visibility and position in a view.
type ListViewColumn struct {
	Visible  bool   `json:"visible"`
	Key      string `json:"key"`
	ID       string `json:"id"`
	Position string `json:"position"`
}

// ListLimits contains List resource counts and limits.
type ListLimits struct {
	OverRowMaximum        bool `json:"over_row_maximum"`
	RowCountLimit         int  `json:"row_count_limit"`
	RowCount              int  `json:"row_count"`
	ArchivedRowCount      int  `json:"archived_row_count"`
	OverColumnMaximum     bool `json:"over_column_maximum"`
	ColumnCount           int  `json:"column_count"`
	ColumnCountLimit      int  `json:"column_count_limit"`
	OverViewMaximum       bool `json:"over_view_maximum"`
	ViewCount             int  `json:"view_count"`
	ViewCountLimit        int  `json:"view_count_limit"`
	MaxAttachmentsPerCell int  `json:"max_attachments_per_cell"`
}
