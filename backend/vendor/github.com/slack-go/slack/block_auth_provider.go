package slack

import "encoding/json"

// AuthProviderBlock displays an authentication provider and its connection state.
//
// Slack SDK schema: https://docs.slack.dev/tools/node-slack-sdk/reference/types/interfaces/AuthProviderBlock/
type AuthProviderBlock struct {
	Type              MessageBlockType      `json:"type"`
	BlockID           string                `json:"block_id,omitempty"`
	Value             string                `json:"value"`
	Name              *TextBlockObject      `json:"name"`
	IsConnected       bool                  `json:"is_connected"`
	Description       *TextBlockObject      `json:"description,omitempty"`
	IconURL           string                `json:"icon_url,omitempty"`
	AuthURL           string                `json:"auth_url,omitempty"`
	DisconnectURL     string                `json:"disconnect_url,omitempty"`
	ConnectedIdentity *TextBlockObject      `json:"connected_identity,omitempty"`
	ConnectedContext  *TextBlockObject      `json:"connected_context,omitempty"`
	Accessory         *OverflowBlockElement `json:"accessory,omitempty"`
	Fallback          *Blocks               `json:"fallback,omitempty"`
}

// UnmarshalJSON preserves an explicitly empty fallback array.
func (b *AuthProviderBlock) UnmarshalJSON(data []byte) error {
	type alias AuthProviderBlock
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.Fallback != nil && decoded.Fallback.BlockSet == nil {
		decoded.Fallback.BlockSet = []Block{}
	}
	*b = AuthProviderBlock(decoded)
	return nil
}

// BlockType returns the type of the block.
func (b AuthProviderBlock) BlockType() MessageBlockType { return b.Type }

// ID returns the block ID.
func (b AuthProviderBlock) ID() string { return b.BlockID }

// AuthProviderBlockOption configures an authentication provider block.
type AuthProviderBlockOption func(*AuthProviderBlock)

// AuthProviderBlockOptionBlockID sets the block ID.
func AuthProviderBlockOptionBlockID(blockID string) AuthProviderBlockOption {
	return func(block *AuthProviderBlock) { block.BlockID = blockID }
}

// NewAuthProviderBlock creates an authentication provider block.
// Value identifies the provider; name must be a plain_text object.
// Set optional fields on the returned block.
func NewAuthProviderBlock(value string, name *TextBlockObject, isConnected bool, options ...AuthProviderBlockOption) *AuthProviderBlock {
	block := &AuthProviderBlock{Type: MBTAuthProvider, Value: value, Name: name, IsConnected: isConnected}
	for _, option := range options {
		if option != nil {
			option(block)
		}
	}
	return block
}
