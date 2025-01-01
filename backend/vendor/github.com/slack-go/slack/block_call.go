package slack

// CallBlock defines data that is used to display a call in slack.
//
// More Information: https://api.slack.com/apis/calls#post_to_channel
type CallBlock struct {
	Type    MessageBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
	CallID  string           `json:"call_id"`
}

// BlockType returns the type of the block
func (s CallBlock) BlockType() MessageBlockType {
	return s.Type
}

// NewFileBlock returns a new instance of a file block
func NewCallBlock(callID string) *CallBlock {
	return &CallBlock{
		Type:   MBTCall,
		CallID: callID,
	}
}
