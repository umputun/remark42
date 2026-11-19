package slack

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// LayoutWidth controls an item's proportional share of a layout row. Narrow,
// standard (the default), and wide have relative weights of 1, 2, and 3.
type LayoutWidth string

const (
	LayoutWidthNarrow   LayoutWidth = "narrow"
	LayoutWidthStandard LayoutWidth = "standard"
	LayoutWidthWide     LayoutWidth = "wide"
)

// LayoutHorizontalAlignment positions content within an item. When unset,
// content fills the item. Start and end follow the reader's text direction.
type LayoutHorizontalAlignment string

const (
	LayoutHorizontalAlignmentStart  LayoutHorizontalAlignment = "start"
	LayoutHorizontalAlignmentCenter LayoutHorizontalAlignment = "center"
	LayoutHorizontalAlignmentEnd    LayoutHorizontalAlignment = "end"
)

// LayoutGridLines controls borders around and between layout rows.
type LayoutGridLines string

const (
	LayoutGridLinesNone  LayoutGridLines = "none"
	LayoutGridLinesRows  LayoutGridLines = "rows"
	LayoutGridLinesOuter LayoutGridLines = "outer"
)

// LayoutBlock arranges child blocks into rows of side-by-side items. Items wrap
// in source order on narrow viewports and stack on mobile.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/layout-block/
type LayoutBlock struct {
	Type    MessageBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
	// Label is required, with a maximum of 150 characters. It is only displayed
	// and exposed to screen readers when IsLabelVisible is true.
	Label          string          `json:"label"`
	Rows           []*LayoutRow    `json:"rows"`
	IsLabelVisible bool            `json:"is_label_visible,omitempty"`
	GridLines      LayoutGridLines `json:"grid_lines,omitempty"`
	// Fallback supplies stable blocks for older clients. Slack does not generate
	// fallback content. A nil pointer omits the field; an empty Blocks value
	// with a non-nil BlockSet preserves an explicit empty array from Slack.
	Fallback *Blocks `json:"fallback,omitempty"`
}

// LayoutRow contains 1 to 3 items, rendered horizontally in source order.
type LayoutRow struct {
	Items []*LayoutItem `json:"items"`
}

// LayoutItem wraps a single child block within a layout row. The child must be
// allowed on the enclosing surface. Markdown blocks are not supported.
type LayoutItem struct {
	Block               Block                     `json:"block"`
	Width               LayoutWidth               `json:"width,omitempty"`
	HorizontalAlignment LayoutHorizontalAlignment `json:"horizontal_alignment,omitempty"`
}

// BlockType returns the type of the block.
func (s LayoutBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block.
func (s LayoutBlock) ID() string {
	return s.BlockID
}

// NewLayoutBlock returns a layout with the required label and rows. Slack
// requires 1 to 5 rows. Use With* methods to set optional fields.
func NewLayoutBlock(label string, rows ...*LayoutRow) *LayoutBlock {
	return &LayoutBlock{Type: MBTLayout, Label: label, Rows: rows}
}

// WithBlockID sets the block ID for the LayoutBlock.
func (s *LayoutBlock) WithBlockID(blockID string) *LayoutBlock {
	s.BlockID = blockID
	return s
}

// WithLabelVisible controls whether the layout label is displayed.
func (s *LayoutBlock) WithLabelVisible(visible bool) *LayoutBlock {
	s.IsLabelVisible = visible
	return s
}

// WithGridLines sets the borders around and between layout rows.
func (s *LayoutBlock) WithGridLines(lines LayoutGridLines) *LayoutBlock {
	s.GridLines = lines
	return s
}

// WithFallback sets blocks rendered by clients that do not support layouts.
// Calling it without arguments sends an explicit empty array.
func (s *LayoutBlock) WithFallback(blocks ...Block) *LayoutBlock {
	s.Fallback = &Blocks{BlockSet: append([]Block{}, blocks...)}
	return s
}

// AddRow appends a row to the layout.
func (s *LayoutBlock) AddRow(row *LayoutRow) *LayoutBlock {
	s.Rows = append(s.Rows, row)
	return s
}

// NewLayoutRow returns a row containing the given items.
func NewLayoutRow(items ...*LayoutItem) *LayoutRow {
	return &LayoutRow{Items: items}
}

// AddItem appends an item to the row.
func (s *LayoutRow) AddItem(item *LayoutItem) *LayoutRow {
	s.Items = append(s.Items, item)
	return s
}

// NewLayoutItem wraps a child block in a layout item.
func NewLayoutItem(block Block) *LayoutItem {
	return &LayoutItem{Block: block}
}

// WithWidth sets the item's proportional share of its row.
func (s *LayoutItem) WithWidth(width LayoutWidth) *LayoutItem {
	s.Width = width
	return s
}

// WithHorizontalAlignment positions the child's content within its item.
func (s *LayoutItem) WithHorizontalAlignment(alignment LayoutHorizontalAlignment) *LayoutItem {
	s.HorizontalAlignment = alignment
	return s
}

// UnmarshalJSON decodes the child through Blocks so all supported block types,
// including nested containers and unknown future blocks, use the same decoder.
func (s *LayoutItem) UnmarshalJSON(data []byte) error {
	var raw struct {
		Block               json.RawMessage           `json:"block"`
		Width               LayoutWidth               `json:"width,omitempty"`
		HorizontalAlignment LayoutHorizontalAlignment `json:"horizontal_alignment,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw.Block) == 0 || string(raw.Block) == "null" {
		return fmt.Errorf("layout item requires a block")
	}
	var blocks Blocks
	if err := json.Unmarshal([]byte("["+string(raw.Block)+"]"), &blocks); err != nil {
		return err
	}
	*s = LayoutItem{Block: blocks.BlockSet[0], Width: raw.Width, HorizontalAlignment: raw.HorizontalAlignment}
	return nil
}

// UnmarshalJSON preserves the explicit empty fallback array returned by Slack.
func (s *LayoutBlock) UnmarshalJSON(data []byte) error {
	type alias LayoutBlock
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.Fallback != nil && decoded.Fallback.BlockSet == nil {
		decoded.Fallback.BlockSet = []Block{}
	}
	*s = LayoutBlock(decoded)
	return nil
}

// Validate checks layout fields, row/item counts, and allowed child and fallback
// block types. Slack enforces surface-specific restrictions, layout/container
// nesting limits, and sizing restrictions on carousel, data_table, and
// data_visualization children. Validation is not performed automatically.
func (s LayoutBlock) Validate() error {
	if s.Type != MBTLayout {
		return fmt.Errorf("type must be %q", MBTLayout)
	}
	if s.Label == "" || runeLen(s.Label) > 150 {
		return fmt.Errorf("label must contain 1 to 150 characters")
	}
	if len(s.Rows) < 1 || len(s.Rows) > 5 {
		return fmt.Errorf("rows must contain 1 to 5 rows")
	}
	switch s.GridLines {
	case "", LayoutGridLinesNone, LayoutGridLinesRows, LayoutGridLinesOuter:
	default:
		return fmt.Errorf("grid_lines must be one of none, rows, or outer")
	}
	for i, row := range s.Rows {
		if row == nil || len(row.Items) < 1 || len(row.Items) > 3 {
			return fmt.Errorf("row %d must contain 1 to 3 items", i)
		}
		for j, item := range row.Items {
			if item == nil || !isLayoutChildBlock(item.Block, false) {
				return fmt.Errorf("row %d item %d requires an allowed child block", i, j)
			}
			switch item.Width {
			case "", LayoutWidthNarrow, LayoutWidthStandard, LayoutWidthWide:
			default:
				return fmt.Errorf("row %d item %d width must be narrow, standard, or wide", i, j)
			}
			switch item.HorizontalAlignment {
			case "", LayoutHorizontalAlignmentStart, LayoutHorizontalAlignmentCenter, LayoutHorizontalAlignmentEnd:
			default:
				return fmt.Errorf("row %d item %d horizontal_alignment must be start, center, or end", i, j)
			}
		}
	}
	if s.Fallback != nil {
		for i, block := range s.Fallback.BlockSet {
			if !isLayoutChildBlock(block, true) {
				return fmt.Errorf("fallback block %d must be a stable top-level block", i)
			}
		}
	}
	return nil
}

func isLayoutChildBlock(block Block, fallback bool) bool {
	if block == nil {
		return false
	}
	value := reflect.ValueOf(block)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return false
	}
	switch block.BlockType() {
	case MBTContext, MBTDivider, MBTHeader, MBTImage, MBTRichText, MBTSection, MBTTable, MBTVideo:
		return true
	case MBTAction, MBTAlert, MBTCard, MBTCarousel, MBTContainer, MBTDataTable, MBTDataVisualization, MBTFile, MBTInput:
		return !fallback
	default:
		return false
	}
}
