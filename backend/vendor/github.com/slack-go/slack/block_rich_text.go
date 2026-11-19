package slack

import (
	"encoding/json"
)

// RichTextBlock defines a new block of type rich_text.
// More Information: https://api.slack.com/changelog/2019-09-what-they-see-is-what-you-get-and-more-and-less
type RichTextBlock struct {
	Type     MessageBlockType  `json:"type"`
	BlockID  string            `json:"block_id,omitempty"`
	Elements []RichTextElement `json:"elements"`
}

func (b RichTextBlock) BlockType() MessageBlockType {
	return b.Type
}

// ID returns the ID of the block
func (s RichTextBlock) ID() string {
	return s.BlockID
}

func (e *RichTextBlock) UnmarshalJSON(b []byte) error {
	var raw struct {
		Type        MessageBlockType  `json:"type"`
		BlockID     string            `json:"block_id"`
		RawElements []json.RawMessage `json:"elements"`
	}
	if string(b) == "{}" {
		return nil
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	elems, err := unmarshalRichTextElements(raw.RawElements)
	if err != nil {
		return err
	}
	*e = RichTextBlock{
		Type:     raw.Type,
		BlockID:  raw.BlockID,
		Elements: elems,
	}
	return nil
}

// NewRichTextBlock returns a new instance of RichText Block.
func NewRichTextBlock(blockID string, elements ...RichTextElement) *RichTextBlock {
	return &RichTextBlock{
		Type:     MBTRichText,
		BlockID:  blockID,
		Elements: elements,
	}
}

type RichTextElementType string

type RichTextElement interface {
	RichTextElementType() RichTextElementType
}

const (
	RTEList         RichTextElementType = "rich_text_list"
	RTEPreformatted RichTextElementType = "rich_text_preformatted"
	RTEQuote        RichTextElementType = "rich_text_quote"
	RTESection      RichTextElementType = "rich_text_section"
	RTEUnknown      RichTextElementType = "rich_text_unknown"
)

type RichTextUnknown struct {
	Type RichTextElementType
	Raw  string
}

func (u RichTextUnknown) RichTextElementType() RichTextElementType {
	return u.Type
}

func (u RichTextUnknown) MarshalJSON() ([]byte, error) {
	return []byte(u.Raw), nil
}

type RichTextListElementType string

const (
	RTEListOrdered RichTextListElementType = "ordered"
	RTEListBullet  RichTextListElementType = "bullet"
)

type RichTextList struct {
	Type     RichTextElementType     `json:"type"`
	Elements []RichTextElement       `json:"elements"`
	Style    RichTextListElementType `json:"style"`
	Indent   int                     `json:"indent"`
	Border   int                     `json:"border"`
	Offset   int                     `json:"offset"`
}

// NewRichTextList returns a new rich text list element.
func NewRichTextList(style RichTextListElementType, indent int, elements ...RichTextElement) *RichTextList {
	return &RichTextList{
		Type:     RTEList,
		Elements: elements,
		Style:    style,
		Indent:   indent,
	}
}

// RichTextElementType returns the type of the Element
func (s RichTextList) RichTextElementType() RichTextElementType {
	return s.Type
}

func (e *RichTextList) UnmarshalJSON(b []byte) error {
	var raw struct {
		RawElements []json.RawMessage       `json:"elements"`
		Style       RichTextListElementType `json:"style"`
		Indent      int                     `json:"indent"`
		Border      int                     `json:"border"`
		Offset      int                     `json:"offset"`
	}
	if string(b) == "{}" {
		return nil
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	elems, err := unmarshalRichTextElements(raw.RawElements)
	if err != nil {
		return err
	}
	*e = RichTextList{
		Type:     RTEList,
		Elements: elems,
		Style:    raw.Style,
		Indent:   raw.Indent,
		Border:   raw.Border,
		Offset:   raw.Offset,
	}
	return nil
}

func unmarshalRichTextElements(raw []json.RawMessage) ([]RichTextElement, error) {
	elems := make([]RichTextElement, 0, len(raw))
	for _, r := range raw {
		var s struct {
			Type RichTextElementType `json:"type"`
		}
		if err := json.Unmarshal(r, &s); err != nil {
			return nil, err
		}
		var elem RichTextElement
		switch s.Type {
		case RTESection:
			elem = &RichTextSection{}
		case RTEList:
			elem = &RichTextList{}
		case RTEQuote:
			elem = &RichTextQuote{}
		case RTEPreformatted:
			elem = &RichTextPreformatted{}
		default:
			elems = append(elems, &RichTextUnknown{
				Type: s.Type,
				Raw:  string(r),
			})
			continue
		}
		if err := json.Unmarshal(r, elem); err != nil {
			return nil, err
		}
		elems = append(elems, elem)
	}
	return elems, nil
}

type RichTextSection struct {
	Type     RichTextElementType      `json:"type"`
	Elements []RichTextSectionElement `json:"elements"`
}

// RichTextElementType returns the type of the Element
func (s RichTextSection) RichTextElementType() RichTextElementType {
	return s.Type
}

func (e *RichTextSection) UnmarshalJSON(b []byte) error {
	var raw struct {
		RawElements []json.RawMessage   `json:"elements"`
		Type        RichTextElementType `json:"type"`
	}
	if string(b) == "{}" {
		return nil
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	elems := make([]RichTextSectionElement, 0, len(raw.RawElements))
	for _, r := range raw.RawElements {
		var s struct {
			Type RichTextSectionElementType `json:"type"`
		}
		if err := json.Unmarshal(r, &s); err != nil {
			return err
		}
		var elem RichTextSectionElement
		switch s.Type {
		case RTSEText:
			elem = &RichTextSectionTextElement{}
		case RTSEAttachmentMention:
			elem = &RichTextSectionAttachmentMentionElement{}
		case RTSETag:
			elem = &RichTextSectionTagElement{}
		case RTSECanvas:
			elem = &RichTextSectionCanvasElement{}
		case RTSECanvasMessageUnfurl:
			elem = &RichTextSectionCanvasMessageUnfurlElement{}
		case RTSECanvasUserMention:
			elem = &RichTextSectionCanvasUserMentionElement{}
		case RTSEChannel:
			elem = &RichTextSectionChannelElement{}
		case RTSECitation:
			elem = &RichTextSectionCitationElement{}
		case RTSEUser:
			elem = &RichTextSectionUserElement{}
		case RTSEEmoji:
			elem = &RichTextSectionEmojiElement{}
		case RTSEFile:
			elem = &RichTextSectionFileElement{}
		case RTSELink:
			elem = &RichTextSectionLinkElement{}
		case RTSEListRecord:
			elem = &RichTextSectionListRecordElement{}
		case RTSEMessageMention:
			elem = &RichTextSectionMessageMentionElement{}
		case RTSESalesforceDataField:
			elem = &RichTextSectionSalesforceDataFieldElement{}
		case RTSETeam:
			elem = &RichTextSectionTeamElement{}
		case RTSEUserGroup:
			elem = &RichTextSectionUserGroupElement{}
		case RTSEWorkObjectMention:
			elem = &RichTextSectionWorkObjectMentionElement{}
		case RTSEWorkflowMention:
			elem = &RichTextSectionWorkflowMentionElement{}
		case RTSEDate:
			elem = &RichTextSectionDateElement{}
		case RTSEBroadcast:
			elem = &RichTextSectionBroadcastElement{}
		case RTSEColor:
			elem = &RichTextSectionColorElement{}
		default:
			elems = append(elems, &RichTextSectionUnknownElement{
				Type: s.Type,
				Raw:  string(r),
			})
			continue
		}
		if err := json.Unmarshal(r, elem); err != nil {
			return err
		}
		elems = append(elems, elem)
	}
	if raw.Type == "" {
		raw.Type = RTESection
	}
	*e = RichTextSection{
		Type:     raw.Type,
		Elements: elems,
	}
	return nil
}

// NewRichTextSection creates a new rich text section from the provided elements. The
// section type will default to "rich_text_section", as it's the only currently supported
// section type.
func NewRichTextSection(elements ...RichTextSectionElement) *RichTextSection {
	return &RichTextSection{
		Type:     RTESection,
		Elements: elements,
	}
}

type RichTextSectionElementType string

const (
	RTSEAttachmentMention   RichTextSectionElementType = "attachment_mention"
	RTSEBroadcast           RichTextSectionElementType = "broadcast"
	RTSECanvas              RichTextSectionElementType = "canvas"
	RTSECanvasMessageUnfurl RichTextSectionElementType = "canvas_message_unfurl"
	RTSECanvasUserMention   RichTextSectionElementType = "canvas_user_mention"
	RTSEChannel             RichTextSectionElementType = "channel"
	RTSECitation            RichTextSectionElementType = "citation"
	RTSEColor               RichTextSectionElementType = "color"
	RTSEDate                RichTextSectionElementType = "date"
	RTSEEmoji               RichTextSectionElementType = "emoji"
	RTSEFile                RichTextSectionElementType = "file"
	RTSELink                RichTextSectionElementType = "link"
	RTSEListRecord          RichTextSectionElementType = "list_record"
	RTSEMessageMention      RichTextSectionElementType = "message_mention"
	RTSESalesforceDataField RichTextSectionElementType = "salesforce_data_field"
	RTSETag                 RichTextSectionElementType = "tag"
	RTSETeam                RichTextSectionElementType = "team"
	RTSEText                RichTextSectionElementType = "text"
	RTSEUser                RichTextSectionElementType = "user"
	RTSEUserGroup           RichTextSectionElementType = "usergroup"
	RTSEWorkObjectMention   RichTextSectionElementType = "work_object_mention"
	RTSEWorkflowMention     RichTextSectionElementType = "workflow_mention"

	RTSEUnknown RichTextSectionElementType = "unknown"
)

type RichTextSectionElement interface {
	RichTextSectionElementType() RichTextSectionElementType
}

type RichTextSectionTextStyle struct {
	Bold            bool `json:"bold,omitempty"`
	Italic          bool `json:"italic,omitempty"`
	Strike          bool `json:"strike,omitempty"`
	Code            bool `json:"code,omitempty"`
	Underline       bool `json:"underline,omitempty"`
	Highlight       bool `json:"highlight,omitempty"`
	ClientHighlight bool `json:"client_highlight,omitempty"`
	Unlink          bool `json:"unlink,omitempty"`
}

type RichTextSectionTextElement struct {
	Type  RichTextSectionElementType `json:"type"`
	Text  string                     `json:"text"`
	Style *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionTextElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionTextElement(text string, style *RichTextSectionTextStyle) *RichTextSectionTextElement {
	return &RichTextSectionTextElement{
		Type:  RTSEText,
		Text:  text,
		Style: style,
	}
}

// RichTextTagColor specifies a rich-text tag's color.
type RichTextTagColor string

const (
	RichTextTagColorGray   RichTextTagColor = "gray"
	RichTextTagColorBrown  RichTextTagColor = "brown"
	RichTextTagColorPurple RichTextTagColor = "purple"
	RichTextTagColorIndigo RichTextTagColor = "indigo"
	RichTextTagColorBlue   RichTextTagColor = "blue"
	RichTextTagColorGreen  RichTextTagColor = "green"
	RichTextTagColorYellow RichTextTagColor = "yellow"
	RichTextTagColorOrange RichTextTagColor = "orange"
	RichTextTagColorRed    RichTextTagColor = "red"
)

// RichTextSectionTagElement represents a tag within rich text.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/tag-element/
type RichTextSectionTagElement struct {
	Type  RichTextSectionElementType `json:"type"`
	Text  string                     `json:"text"`
	Color RichTextTagColor           `json:"color,omitempty"`
	Style *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionTagElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionTagElement(text string, style *RichTextSectionTextStyle) *RichTextSectionTagElement {
	return &RichTextSectionTagElement{Type: RTSETag, Text: text, Style: style}
}

type RichTextSectionChannelElement struct {
	Type      RichTextSectionElementType `json:"type"`
	ChannelID string                     `json:"channel_id"`
	TabID     string                     `json:"tab_id,omitempty"`
	FromLLM   *bool                      `json:"from_llm,omitempty"`
	Style     *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionChannelElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionChannelElement(channelID string, style *RichTextSectionTextStyle) *RichTextSectionChannelElement {
	return &RichTextSectionChannelElement{
		Type:      RTSEChannel,
		ChannelID: channelID,
		Style:     style,
	}
}

type RichTextSectionUserElement struct {
	Type    RichTextSectionElementType `json:"type"`
	UserID  string                     `json:"user_id"`
	FromLLM *bool                      `json:"from_llm,omitempty"`
	Style   *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionUserElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionUserElement(userID string, style *RichTextSectionTextStyle) *RichTextSectionUserElement {
	return &RichTextSectionUserElement{
		Type:   RTSEUser,
		UserID: userID,
		Style:  style,
	}
}

type RichTextSectionEmojiElement struct {
	Type     RichTextSectionElementType `json:"type"`
	Name     string                     `json:"name"`
	SkinTone int                        `json:"skin_tone,omitempty"`
	Unicode  string                     `json:"unicode,omitempty"`
	// URL is used for custom emoji shared across workspaces.
	// Slack's Node SDK documents this field; the Block Kit reference omits it.
	// https://docs.slack.dev/tools/node-slack-sdk/reference/types/interfaces/RichTextEmoji/#url
	URL   string                    `json:"url,omitempty"`
	Style *RichTextSectionTextStyle `json:"style,omitempty"`
}

func (r RichTextSectionEmojiElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionEmojiElement(name string, skinTone int, style *RichTextSectionTextStyle) *RichTextSectionEmojiElement {
	return &RichTextSectionEmojiElement{
		Type:     RTSEEmoji,
		Name:     name,
		SkinTone: skinTone,
		Style:    style,
	}
}

// RichTextSectionAttachmentMentionElement references an app attachment within rich text.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/attachment-mention-element/
type RichTextSectionAttachmentMentionElement struct {
	Type                   RichTextSectionElementType `json:"type"`
	URL                    string                     `json:"url"`
	Text                   string                     `json:"text,omitempty"`
	AppID                  string                     `json:"app_id,omitempty"`
	EntityID               string                     `json:"entity_id,omitempty"`
	IconURL                string                     `json:"icon_url,omitempty"`
	ChannelID              string                     `json:"channel_id,omitempty"`
	Timestamp              string                     `json:"ts,omitempty"`
	FullSizePreviewEnabled *bool                      `json:"full_size_preview_enabled,omitempty"`
	IconName               string                     `json:"icon_name,omitempty"`
	ReferenceObjectType    string                     `json:"reference_object_type,omitempty"`
	ProductName            string                     `json:"product_name,omitempty"`
	Style                  *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionAttachmentMentionElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionAttachmentMentionElement(url string, style *RichTextSectionTextStyle) *RichTextSectionAttachmentMentionElement {
	return &RichTextSectionAttachmentMentionElement{Type: RTSEAttachmentMention, URL: url, Style: style}
}

// RichTextSectionWorkObjectMentionElement references a Work Object within rich text.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/work-object-mention-element/
type RichTextSectionWorkObjectMentionElement struct {
	Type                   RichTextSectionElementType `json:"type"`
	EntityID               string                     `json:"entity_id"`
	AppID                  string                     `json:"app_id"`
	Text                   string                     `json:"text"`
	URL                    string                     `json:"url"`
	IconURL                string                     `json:"icon_url,omitempty"`
	FullSizePreviewEnabled *bool                      `json:"full_size_preview_enabled,omitempty"`
	ProductName            string                     `json:"product_name,omitempty"`
	Style                  *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionWorkObjectMentionElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionWorkObjectMentionElement(entityID, appID, text, url string, style *RichTextSectionTextStyle) *RichTextSectionWorkObjectMentionElement {
	return &RichTextSectionWorkObjectMentionElement{
		Type: RTSEWorkObjectMention, EntityID: entityID, AppID: appID, Text: text, URL: url, Style: style,
	}
}

// RichTextSectionWorkflowMentionElement references a workflow within rich text.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/workflow-mention-element/
type RichTextSectionWorkflowMentionElement struct {
	Type              RichTextSectionElementType `json:"type"`
	WorkflowID        string                     `json:"workflow_id"`
	FunctionTriggerID string                     `json:"function_trigger_id"`
	Text              string                     `json:"text"`
	URL               string                     `json:"url,omitempty"`
	ChannelID         string                     `json:"channel_id,omitempty"`
	Timestamp         string                     `json:"ts,omitempty"`
	Style             *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionWorkflowMentionElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionWorkflowMentionElement(workflowID, functionTriggerID, text string, style *RichTextSectionTextStyle) *RichTextSectionWorkflowMentionElement {
	return &RichTextSectionWorkflowMentionElement{
		Type: RTSEWorkflowMention, WorkflowID: workflowID, FunctionTriggerID: functionTriggerID, Text: text, Style: style,
	}
}

// RichTextSectionSalesforceDataFieldElement references a Salesforce record field within rich text.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/salesforce-data-field-element/
type RichTextSectionSalesforceDataFieldElement struct {
	Type              RichTextSectionElementType `json:"type"`
	RecordID          string                     `json:"salesforce_record_id"`
	FieldLabel        string                     `json:"salesforce_field_label,omitempty"`
	FieldAPIName      string                     `json:"salesforce_field_api_name,omitempty"`
	IncludeFieldLabel *bool                      `json:"salesforce_include_field_label,omitempty"`
	// Observed in live responses; absent from Slack's reference and SDK models.
	HasWithheldFieldNames *bool                     `json:"has_withheld_salesforce_field_names,omitempty"`
	Style                 *RichTextSectionTextStyle `json:"style,omitempty"`
}

func (r RichTextSectionSalesforceDataFieldElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionSalesforceDataFieldElement(recordID string, style *RichTextSectionTextStyle) *RichTextSectionSalesforceDataFieldElement {
	return &RichTextSectionSalesforceDataFieldElement{Type: RTSESalesforceDataField, RecordID: recordID, Style: style}
}

// RichTextSectionCanvasElement references a canvas within rich text.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/canvas-element/
type RichTextSectionCanvasElement struct {
	Type              RichTextSectionElementType `json:"type"`
	FileID            string                     `json:"file_id"`
	Label             *string                    `json:"label,omitempty"` // Slack preserves an explicitly empty label.
	HideTitle         *bool                      `json:"hide_title,omitempty"`
	SectionID         string                     `json:"section_id,omitempty"`
	Style             *RichTextSectionTextStyle  `json:"style,omitempty"`
	Text              string                     `json:"text,omitempty"`
	URL               string                     `json:"url,omitempty"`
	IsSkillInvocation *bool                      `json:"is_skill_invocation,omitempty"`
}

func (r RichTextSectionCanvasElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionCanvasElement(fileID string, style *RichTextSectionTextStyle) *RichTextSectionCanvasElement {
	return &RichTextSectionCanvasElement{Type: RTSECanvas, FileID: fileID, Style: style}
}

// RichTextSectionCanvasUserMentionElement represents a user mention in canvas content.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/canvas-user-mention-element/
type RichTextSectionCanvasUserMentionElement struct {
	Type     RichTextSectionElementType `json:"type"`
	UserID   string                     `json:"user_id"`
	ThreadID string                     `json:"thread_id,omitempty"`
	Style    *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionCanvasUserMentionElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionCanvasUserMentionElement(userID string, style *RichTextSectionTextStyle) *RichTextSectionCanvasUserMentionElement {
	return &RichTextSectionCanvasUserMentionElement{Type: RTSECanvasUserMention, UserID: userID, Style: style}
}

// RichTextSectionCanvasMessageUnfurlElement represents a message reference in canvas content.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/canvas-message-unfurl-element/
type RichTextSectionCanvasMessageUnfurlElement struct {
	Type                 RichTextSectionElementType `json:"type"`
	RootMessageChannel   string                     `json:"root_message_channel"`
	RootMessageTimestamp string                     `json:"root_message_ts"`
	Style                *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionCanvasMessageUnfurlElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionCanvasMessageUnfurlElement(rootMessageChannel, rootMessageTimestamp string, style *RichTextSectionTextStyle) *RichTextSectionCanvasMessageUnfurlElement {
	return &RichTextSectionCanvasMessageUnfurlElement{
		Type: RTSECanvasMessageUnfurl, RootMessageChannel: rootMessageChannel, RootMessageTimestamp: rootMessageTimestamp, Style: style,
	}
}

// RichTextSectionFileElement references a file within rich text.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/file-element/
type RichTextSectionFileElement struct {
	Type              RichTextSectionElementType `json:"type"`
	FileID            string                     `json:"file_id"`
	Text              string                     `json:"text,omitempty"`
	URL               string                     `json:"url,omitempty"`
	Style             *RichTextSectionTextStyle  `json:"style,omitempty"`
	IsSkillInvocation *bool                      `json:"is_skill_invocation,omitempty"`
}

func (r RichTextSectionFileElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionFileElement(fileID string, style *RichTextSectionTextStyle) *RichTextSectionFileElement {
	return &RichTextSectionFileElement{Type: RTSEFile, FileID: fileID, Style: style}
}

// RichTextSectionListRecordElement references a Slack List or item within rich text.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/list-record-element/
type RichTextSectionListRecordElement struct {
	Type     RichTextSectionElementType `json:"type"`
	FileID   string                     `json:"file_id"`
	RecordID string                     `json:"record_id,omitempty"`
	ViewID   string                     `json:"view_id,omitempty"`
	Style    *RichTextSectionTextStyle  `json:"style,omitempty"`
	Text     string                     `json:"text,omitempty"`
	URL      string                     `json:"url,omitempty"`
}

func (r RichTextSectionListRecordElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionListRecordElement(fileID string, style *RichTextSectionTextStyle) *RichTextSectionListRecordElement {
	return &RichTextSectionListRecordElement{Type: RTSEListRecord, FileID: fileID, Style: style}
}

// RichTextSectionMessageMentionElement references a message within rich text.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/message-mention-element/
type RichTextSectionMessageMentionElement struct {
	Type             RichTextSectionElementType `json:"type"`
	ChannelID        string                     `json:"channel_id"`
	MessageTimestamp string                     `json:"message_ts"`
	AuthorID         string                     `json:"author_id,omitempty"`
	ThreadTimestamp  string                     `json:"thread_ts,omitempty"`
	Text             string                     `json:"text,omitempty"`
	URL              string                     `json:"url,omitempty"`
	Style            *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionMessageMentionElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionMessageMentionElement(channelID, messageTimestamp string, style *RichTextSectionTextStyle) *RichTextSectionMessageMentionElement {
	return &RichTextSectionMessageMentionElement{
		Type: RTSEMessageMention, ChannelID: channelID, MessageTimestamp: messageTimestamp, Style: style,
	}
}

type RichTextSectionLinkElement struct {
	Type       RichTextSectionElementType `json:"type"`
	URL        string                     `json:"url"`
	Text       string                     `json:"text,omitempty"`
	Unsafe     *bool                      `json:"unsafe,omitempty"`
	FromLLM    *bool                      `json:"from_llm,omitempty"`
	IsSlackURL *bool                      `json:"is_slack_url,omitempty"`
	Truncated  *bool                      `json:"truncated,omitempty"`
	Style      *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionLinkElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionLinkElement(url, text string, style *RichTextSectionTextStyle) *RichTextSectionLinkElement {
	return &RichTextSectionLinkElement{
		Type:  RTSELink,
		URL:   url,
		Text:  text,
		Style: style,
	}
}

type RichTextSectionTeamElement struct {
	Type   RichTextSectionElementType `json:"type"`
	TeamID string                     `json:"team_id"`
	Style  *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionTeamElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionTeamElement(teamID string, style *RichTextSectionTextStyle) *RichTextSectionTeamElement {
	return &RichTextSectionTeamElement{
		Type:   RTSETeam,
		TeamID: teamID,
		Style:  style,
	}
}

type RichTextSectionUserGroupElement struct {
	Type        RichTextSectionElementType `json:"type"`
	UsergroupID string                     `json:"usergroup_id"`
	Style       *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionUserGroupElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionUserGroupElement(usergroupID string) *RichTextSectionUserGroupElement {
	return &RichTextSectionUserGroupElement{
		Type:        RTSEUserGroup,
		UsergroupID: usergroupID,
	}
}

type RichTextSectionDateElement struct {
	Type      RichTextSectionElementType `json:"type"`
	Timestamp JSONTime                   `json:"timestamp"`
	Format    string                     `json:"format"`
	Timezone  string                     `json:"timezone,omitempty"`
	URL       *string                    `json:"url,omitempty"`
	Fallback  *string                    `json:"fallback,omitempty"`
	Style     *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionDateElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionDateElement(timestamp int64, format string, url *string, fallback *string) *RichTextSectionDateElement {
	return &RichTextSectionDateElement{
		Type:      RTSEDate,
		Timestamp: JSONTime(timestamp),
		Format:    format,
		URL:       url,
		Fallback:  fallback,
	}
}

type RichTextSectionBroadcastElement struct {
	Type  RichTextSectionElementType `json:"type"`
	Range string                     `json:"range"`
	Style *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionBroadcastElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionBroadcastElement(rangeStr string) *RichTextSectionBroadcastElement {
	return &RichTextSectionBroadcastElement{
		Type:  RTSEBroadcast,
		Range: rangeStr,
	}
}

type RichTextSectionColorElement struct {
	Type  RichTextSectionElementType `json:"type"`
	Value string                     `json:"value"`
	Style *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionColorElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionColorElement(value string) *RichTextSectionColorElement {
	return &RichTextSectionColorElement{
		Type:  RTSEColor,
		Value: value,
	}
}

type RichTextSectionUnknownElement struct {
	Type RichTextSectionElementType `json:"type"`
	Raw  string
}

func (r RichTextSectionUnknownElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func (r RichTextSectionUnknownElement) MarshalJSON() ([]byte, error) {
	return []byte(r.Raw), nil
}

// RichTextQuote represents rich_text_quote element type.
type RichTextQuote struct {
	Type     RichTextElementType      `json:"type"`
	Elements []RichTextSectionElement `json:"elements"`
	Border   int                      `json:"border,omitempty"`
}

// RichTextElementType returns the type of the Element
func (s *RichTextQuote) RichTextElementType() RichTextElementType {
	return s.Type
}

func (s *RichTextQuote) UnmarshalJSON(b []byte) error {
	// reusing the RichTextSection struct, as it's the same as RichTextQuote.
	var rts RichTextSection
	if err := json.Unmarshal(b, &rts); err != nil {
		return err
	}
	var standalone struct {
		Border int `json:"border"`
	}
	if err := json.Unmarshal(b, &standalone); err != nil {
		return err
	}
	*s = RichTextQuote{
		Type:     RTEQuote,
		Elements: rts.Elements,
		Border:   standalone.Border,
	}
	return nil
}

// RichTextPreformatted represents rich_text_quote element type.
type RichTextPreformatted struct {
	Type     RichTextElementType      `json:"type"`
	Elements []RichTextSectionElement `json:"elements"`
	Border   int                      `json:"border"`
	Language string                   `json:"language,omitempty"`
}

// RichTextElementType returns the type of the Element
func (s *RichTextPreformatted) RichTextElementType() RichTextElementType {
	return s.Type
}

func (s *RichTextPreformatted) UnmarshalJSON(b []byte) error {
	var rts RichTextSection
	if err := json.Unmarshal(b, &rts); err != nil {
		return err
	}
	// we define standalone fields because we need to unmarshal the border
	// field.  We can not directly unmarshal the data into
	// RichTextPreformatted because it will cause an infinite loop.  We also
	// can not define a struct with embedded RichTextSection and Border fields
	// because the json package will not unmarshal the data into the
	// standalone fields, once it sees UnmarshalJSON method on the embedded
	// struct.  The drawback is that we have to process the data twice, and
	// have to define a standalone struct with the same set of fields as the
	// original struct, which may become a maintenance burden (i.e. update the
	// fields in two places, should it ever change).
	var standalone struct {
		Border   int    `json:"border"`
		Language string `json:"language"`
	}
	if err := json.Unmarshal(b, &standalone); err != nil {
		return err
	}
	*s = RichTextPreformatted{
		Type:     RTEPreformatted,
		Elements: rts.Elements,
		Border:   standalone.Border,
		Language: standalone.Language,
	}
	return nil
}
