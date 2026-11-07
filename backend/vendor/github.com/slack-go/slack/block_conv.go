package slack

import (
	"encoding/json"
	"fmt"
)

type sumtype struct {
	TypeVal string `json:"type"`
}

// MarshalJSON encodes the underlying block slice.
func (b Blocks) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.BlockSet)
}

// UnmarshalJSON implements the Unmarshaller interface for Blocks, so that any JSON
// unmarshalling is delegated and proper type determination can be made before unmarshal
func (b *Blocks) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage

	if string(data) == "{}" {
		return nil
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	var blocks Blocks
	for _, r := range raw {
		s := sumtype{}
		err := json.Unmarshal(r, &s)
		if err != nil {
			return err
		}

		var block Block
		switch s.TypeVal {
		case "actions":
			block = &ActionBlock{}
		case "context":
			block = &ContextBlock{}
		case "context_actions":
			block = &ContextActionsBlock{}
		case "divider":
			block = &DividerBlock{}
		case "file":
			block = &FileBlock{}
		case "header":
			block = &HeaderBlock{}
		case "image":
			block = &ImageBlock{}
		case "input":
			block = &InputBlock{}
		case "markdown":
			block = &MarkdownBlock{}
		case "rich_text":
			block = &RichTextBlock{}
		case "rich_text_input":
			block = &RichTextBlock{}
		case "section":
			block = &SectionBlock{}
		case "call":
			block = &CallBlock{}
		case "video":
			block = &VideoBlock{}
		case "table":
			block = &TableBlock{}
		case "data_table":
			block = &DataTableBlock{}
		case "data_visualization":
			block = &DataVisualizationBlock{}
		case "task_card":
			block = &TaskCardBlock{}
		case "alert":
			block = &AlertBlock{}
		case "auth_provider":
			block = &AuthProviderBlock{}
		case "plan":
			block = &PlanBlock{}
		case "card":
			block = &CardBlock{}
		case "carousel":
			block = &CarouselBlock{}
		case "container":
			block = &ContainerBlock{}
		case "layout":
			block = &LayoutBlock{}
		default:
			block = &UnknownBlock{raw: r}
		}

		err = json.Unmarshal(r, block)
		if err != nil {
			return err
		}

		blocks.BlockSet = append(blocks.BlockSet, block)
	}

	*b = blocks
	return nil
}

// UnmarshalJSON implements the Unmarshaller interface for InputBlock, so that any JSON
// unmarshalling is delegated and proper type determination can be made before unmarshal
func (b *InputBlock) UnmarshalJSON(data []byte) error {
	type alias InputBlock
	a := struct {
		Element json.RawMessage `json:"element"`
		*alias
	}{
		alias: (*alias)(b),
	}

	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	// Preserve compatibility with partial blocks and null input.
	if len(a.Element) == 0 {
		return nil
	}

	s := sumtype{}
	if err := json.Unmarshal(a.Element, &s); err != nil {
		return err
	}

	var e BlockElement
	switch s.TypeVal {
	case "datepicker":
		e = &DatePickerBlockElement{}
	case "timepicker":
		e = &TimePickerBlockElement{}
	case "datetimepicker":
		e = &DateTimePickerBlockElement{}
	case "plain_text_input":
		e = &PlainTextInputBlockElement{}
	case "rich_text_input":
		e = &RichTextInputBlockElement{}
	case "email_text_input":
		e = &EmailTextInputBlockElement{}
	case "url_text_input":
		e = &URLTextInputBlockElement{}
	case "static_select", "external_select", "users_select", "conversations_select", "channels_select":
		e = &SelectBlockElement{}
	case "multi_static_select", "multi_external_select", "multi_users_select", "multi_conversations_select", "multi_channels_select":
		e = &MultiSelectBlockElement{}
	case "checkboxes":
		e = &CheckboxGroupsBlockElement{}
	case "overflow":
		e = &OverflowBlockElement{}
	case "radio_buttons":
		e = &RadioButtonsBlockElement{}
	case "number_input":
		e = &NumberInputBlockElement{}
	case "file_input":
		e = &FileInputBlockElement{}
	case "feedback_buttons":
		e = &FeedbackButtonsBlockElement{}
	case "icon_button":
		e = &IconButtonBlockElement{}
	case "workflow_button":
		e = &WorkflowButtonBlockElement{}
	default:
		return fmt.Errorf("unsupported block element type %v", s.TypeVal)
	}

	if err := json.Unmarshal(a.Element, e); err != nil {
		return err
	}
	b.Element = e

	return nil
}

// MarshalJSON encodes the underlying block element slice.
func (b *BlockElements) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.ElementSet)
}

// UnmarshalJSON implements the Unmarshaller interface for BlockElements, so that any JSON
// unmarshalling is delegated and proper type determination can be made before unmarshal
func (b *BlockElements) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage

	if string(data) == "{}" {
		return nil
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	var blockElements BlockElements
	for _, r := range raw {
		s := sumtype{}
		err := json.Unmarshal(r, &s)
		if err != nil {
			return err
		}

		var blockElement BlockElement
		switch s.TypeVal {
		case "image":
			blockElement = &ImageBlockElement{}
		case "button":
			blockElement = &ButtonBlockElement{}
		case "overflow":
			blockElement = &OverflowBlockElement{}
		case "datepicker":
			blockElement = &DatePickerBlockElement{}
		case "timepicker":
			blockElement = &TimePickerBlockElement{}
		case "datetimepicker":
			blockElement = &DateTimePickerBlockElement{}
		case "plain_text_input":
			blockElement = &PlainTextInputBlockElement{}
		case "rich_text_input":
			blockElement = &RichTextInputBlockElement{}
		case "email_text_input":
			blockElement = &EmailTextInputBlockElement{}
		case "url_text_input":
			blockElement = &URLTextInputBlockElement{}
		case "checkboxes":
			blockElement = &CheckboxGroupsBlockElement{}
		case "radio_buttons":
			blockElement = &RadioButtonsBlockElement{}
		case "static_select", "external_select", "users_select", "conversations_select", "channels_select":
			blockElement = &SelectBlockElement{}
		case "multi_static_select", "multi_external_select", "multi_users_select", "multi_conversations_select", "multi_channels_select":
			blockElement = &MultiSelectBlockElement{}
		case "number_input":
			blockElement = &NumberInputBlockElement{}
		case "file_input":
			blockElement = &FileInputBlockElement{}
		case "feedback_buttons":
			blockElement = &FeedbackButtonsBlockElement{}
		case "icon_button":
			blockElement = &IconButtonBlockElement{}
		case "workflow_button":
			blockElement = &WorkflowButtonBlockElement{}
		default:
			return fmt.Errorf("unsupported block element type %v", s.TypeVal)
		}

		err = json.Unmarshal(r, blockElement)
		if err != nil {
			return err
		}

		blockElements.ElementSet = append(blockElements.ElementSet, blockElement)
	}

	*b = blockElements
	return nil
}

// MarshalJSON encodes the selected accessory element.
func (a *Accessory) MarshalJSON() ([]byte, error) {
	return json.Marshal(toBlockElement(a))
}

// UnmarshalJSON implements the Unmarshaller interface for Accessory, so that any JSON
// unmarshalling is delegated and proper type determination can be made before unmarshal
// Note: datetimepicker is not supported in Accessory
func (a *Accessory) UnmarshalJSON(data []byte) error {
	var r json.RawMessage

	if string(data) == "{\"accessory\":null}" {
		return nil
	}

	err := json.Unmarshal(data, &r)
	if err != nil {
		return err
	}

	s := sumtype{}
	err = json.Unmarshal(r, &s)
	if err != nil {
		return err
	}

	switch s.TypeVal {
	case "image":
		element := &ImageBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.ImageElement = element
	case "button":
		element := &ButtonBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.ButtonElement = element
	case "overflow":
		element := &OverflowBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.OverflowElement = element
	case "datepicker":
		element := &DatePickerBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.DatePickerElement = element
	case "timepicker":
		element := &TimePickerBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.TimePickerElement = element
	case "plain_text_input":
		element := &PlainTextInputBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.PlainTextInputElement = element
	case "rich_text_input":
		element := &RichTextInputBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.RichTextInputElement = element
	case "radio_buttons":
		element := &RadioButtonsBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.RadioButtonsElement = element
	case "static_select", "external_select", "users_select", "conversations_select", "channels_select":
		element := &SelectBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.SelectElement = element
	case "multi_static_select", "multi_external_select", "multi_users_select", "multi_conversations_select", "multi_channels_select":
		element := &MultiSelectBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.MultiSelectElement = element
	case "checkboxes":
		element := &CheckboxGroupsBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.CheckboxGroupsBlockElement = element
	case "workflow_button":
		element := &WorkflowButtonBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.WorkflowButtonElement = element
	default:
		element := &UnknownBlockElement{}
		if err := json.Unmarshal(r, element); err != nil {
			return err
		}
		a.UnknownElement = element
	}

	return nil
}

func toBlockElement(element *Accessory) BlockElement {
	if element.ImageElement != nil {
		return element.ImageElement
	}
	if element.ButtonElement != nil {
		return element.ButtonElement
	}
	if element.OverflowElement != nil {
		return element.OverflowElement
	}
	if element.DatePickerElement != nil {
		return element.DatePickerElement
	}
	if element.TimePickerElement != nil {
		return element.TimePickerElement
	}
	if element.PlainTextInputElement != nil {
		return element.PlainTextInputElement
	}
	if element.RadioButtonsElement != nil {
		return element.RadioButtonsElement
	}
	if element.CheckboxGroupsBlockElement != nil {
		return element.CheckboxGroupsBlockElement
	}
	if element.SelectElement != nil {
		return element.SelectElement
	}
	if element.MultiSelectElement != nil {
		return element.MultiSelectElement
	}
	if element.RichTextInputElement != nil {
		return element.RichTextInputElement
	}
	if element.WorkflowButtonElement != nil {
		return element.WorkflowButtonElement
	}

	return nil
}

// MarshalJSON encodes the underlying context element slice.
func (e ContextElements) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Elements)
}

// UnmarshalJSON implements the Unmarshaller interface for ContextElements, so that any JSON
// unmarshalling is delegated and proper type determination can be made before unmarshal
func (e *ContextElements) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage

	if string(data) == "{\"elements\":null}" {
		return nil
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	for _, r := range raw {
		s := sumtype{}
		err := json.Unmarshal(r, &s)
		if err != nil {
			return err
		}

		switch s.TypeVal {
		case PlainTextType, MarkdownType:
			elem := &TextBlockObject{}
			if err := json.Unmarshal(r, elem); err != nil {
				return err
			}

			e.Elements = append(e.Elements, elem)
		case "image":
			elem := &ImageBlockElement{}
			if err := json.Unmarshal(r, elem); err != nil {
				return err
			}

			e.Elements = append(e.Elements, elem)
		default:
			return fmt.Errorf("unsupported context element type %v", s.TypeVal)
		}
	}

	return nil
}
