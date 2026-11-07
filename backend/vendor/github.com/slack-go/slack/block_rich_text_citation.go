package slack

import "encoding/json"

// RichTextSectionCitationElement references a source within rich text.
//
// Slack API docs: https://docs.slack.dev/reference/block-kit/block-elements/citation-element/
type RichTextSectionCitationElement struct {
	Type RichTextSectionElementType `json:"type"`
	URL  string                     `json:"url"`
	Text string                     `json:"text"`
	// Slack accepts and returns fractional index values.
	Index      float64                 `json:"index"`
	Details    RichTextCitationDetails `json:"details"`
	FromLLM    *bool                   `json:"from_llm,omitempty"`
	IsSlackURL *bool                   `json:"is_slack_url,omitempty"`
}

func (r RichTextSectionCitationElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionCitationElement(url, text string, index float64, details RichTextCitationDetails) *RichTextSectionCitationElement {
	return &RichTextSectionCitationElement{
		Type:    RTSECitation,
		URL:     url,
		Text:    text,
		Index:   index,
		Details: details,
	}
}

// UnmarshalJSON selects the details type by citation_type and preserves unknown variants.
func (r *RichTextSectionCitationElement) UnmarshalJSON(data []byte) error {
	type citationFields RichTextSectionCitationElement
	var raw struct {
		citationFields
		Details json.RawMessage `json:"details"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	details, err := unmarshalRichTextCitationDetails(raw.Details)
	if err != nil {
		return err
	}
	raw.citationFields.Details = details
	*r = RichTextSectionCitationElement(raw.citationFields)
	return nil
}

func unmarshalRichTextCitationDetails(data json.RawMessage) (RichTextCitationDetails, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	var probe struct {
		Type RichTextCitationType `json:"citation_type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}
	var details RichTextCitationDetails
	switch probe.Type {
	case RichTextCitationFile:
		details = &RichTextCitationFileDetails{}
	case RichTextCitationExternal:
		details = &RichTextCitationExternalDetails{}
	case RichTextCitationWeb:
		details = &RichTextCitationWebDetails{}
	case RichTextCitationMessage:
		details = &RichTextCitationMessageDetails{}
	case RichTextCitationMemory:
		details = &RichTextCitationMemoryDetails{}
	default:
		return &RichTextCitationUnknownDetails{Type: probe.Type, Raw: data}, nil
	}
	if err := json.Unmarshal(data, details); err != nil {
		return nil, err
	}
	return details, nil
}

// RichTextCitationType identifies the source of a citation.
type RichTextCitationType string

const (
	RichTextCitationFile     RichTextCitationType = "file"
	RichTextCitationExternal RichTextCitationType = "external"
	RichTextCitationWeb      RichTextCitationType = "web"
	RichTextCitationMessage  RichTextCitationType = "message"
	RichTextCitationMemory   RichTextCitationType = "memory"
)

// RichTextCitationDetails describes the source of a citation.
type RichTextCitationDetails interface {
	CitationType() RichTextCitationType
}

// RichTextCitationFileDetails identifies a Slack file.
type RichTextCitationFileDetails struct {
	Type       RichTextCitationType `json:"citation_type"`
	FileID     string               `json:"file_id"`
	Descriptor *string              `json:"descriptor,omitempty"`
}

func (d RichTextCitationFileDetails) CitationType() RichTextCitationType {
	return d.Type
}

func NewRichTextCitationFileDetails(fileID string) *RichTextCitationFileDetails {
	return &RichTextCitationFileDetails{Type: RichTextCitationFile, FileID: fileID}
}

// RichTextCitationExternalDetails identifies an external app.
type RichTextCitationExternalDetails struct {
	Type       RichTextCitationType `json:"citation_type"`
	AppName    string               `json:"app_name"`
	AppIconURL string               `json:"app_icon_url"`
}

func (d RichTextCitationExternalDetails) CitationType() RichTextCitationType {
	return d.Type
}

func NewRichTextCitationExternalDetails(appName, appIconURL string) *RichTextCitationExternalDetails {
	return &RichTextCitationExternalDetails{Type: RichTextCitationExternal, AppName: appName, AppIconURL: appIconURL}
}

// RichTextCitationWebDetails describes a web source.
type RichTextCitationWebDetails struct {
	Type        RichTextCitationType `json:"citation_type"`
	DisplayName string               `json:"display_name"`
	Title       *string              `json:"title,omitempty"`
	Snippet     *string              `json:"snippet,omitempty"`
}

func (d RichTextCitationWebDetails) CitationType() RichTextCitationType {
	return d.Type
}

func NewRichTextCitationWebDetails(displayName string) *RichTextCitationWebDetails {
	return &RichTextCitationWebDetails{Type: RichTextCitationWeb, DisplayName: displayName}
}

// RichTextCitationMessageDetails identifies a message within a channel.
type RichTextCitationMessageDetails struct {
	Type             RichTextCitationType `json:"citation_type"`
	Channel          string               `json:"channel"`
	MessageTimestamp string               `json:"message_ts"`
}

func (d RichTextCitationMessageDetails) CitationType() RichTextCitationType {
	return d.Type
}

func NewRichTextCitationMessageDetails(channel, messageTimestamp string) *RichTextCitationMessageDetails {
	return &RichTextCitationMessageDetails{Type: RichTextCitationMessage, Channel: channel, MessageTimestamp: messageTimestamp}
}

// RichTextCitationMemoryDetails identifies a Slack memory.
type RichTextCitationMemoryDetails struct {
	Type     RichTextCitationType `json:"citation_type"`
	MemoryID string               `json:"memory_id"`
}

func (d RichTextCitationMemoryDetails) CitationType() RichTextCitationType {
	return d.Type
}

func NewRichTextCitationMemoryDetails(memoryID string) *RichTextCitationMemoryDetails {
	return &RichTextCitationMemoryDetails{Type: RichTextCitationMemory, MemoryID: memoryID}
}

// RichTextCitationUnknownDetails preserves a citation source that the library does not model.
type RichTextCitationUnknownDetails struct {
	Type RichTextCitationType
	Raw  json.RawMessage
}

func (d RichTextCitationUnknownDetails) CitationType() RichTextCitationType {
	return d.Type
}

func (d RichTextCitationUnknownDetails) MarshalJSON() ([]byte, error) {
	return d.Raw, nil
}
