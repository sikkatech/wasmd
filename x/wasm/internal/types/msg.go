package types

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MaxWasmSize   = 500 * 1024
	BuildTagRegex = "^cosmwasm-opt:"
)

type MsgStoreCode struct {
	Sender sdk.AccAddress `json:"sender" yaml:"sender"`
	// WASMByteCode can be raw or gzip compressed
	WASMByteCode []byte `json:"wasm_byte_code" yaml:"wasm_byte_code"`
	// Source is a valid URI reference to the contract's source code, optional
	Source string `json:"source" yaml:"source"`
	// Builder is a docker tag, optional
	Builder string `json:"builder" yaml:"builder"`
}

func (msg MsgStoreCode) Route() string {
	return RouterKey
}

func (msg MsgStoreCode) Type() string {
	return "store-code"
}

func (msg MsgStoreCode) ValidateBasic() sdk.Error {
	if len(msg.WASMByteCode) == 0 {
		return sdk.ErrInternal("empty wasm code")
	}

	if len(msg.WASMByteCode) > MaxWasmSize {
		return sdk.ErrInternal("wasm code too large")
	}

	if msg.Source != "" {
		u, err := url.Parse(msg.Source)
		if err != nil {
			return sdk.ErrInternal("source should be a valid url")
		}

		if !u.IsAbs() {
			return sdk.ErrInternal("source should be an absolute url")
		}

		// check if the source is reachable
		resp, err := http.Get(msg.Source)
		if err != nil || resp.StatusCode != 200 {
			return sdk.ErrInternal("source url is not reachable")
		}
	}

	if msg.Builder != "" {
		ok, err := regexp.MatchString(BuildTagRegex, msg.Builder)
		if err != nil || !ok {
			return sdk.ErrInternal("invalid tag supplied for builder")
		}
	}

	return nil
}

func (msg MsgStoreCode) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgStoreCode) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

type MsgInstantiateContract struct {
	Sender    sdk.AccAddress  `json:"sender" yaml:"sender"`
	Code      uint64          `json:"code_id" yaml:"code_id"`
	InitMsg   json.RawMessage `json:"init_msg" yaml:"init_msg"`
	InitFunds sdk.Coins       `json:"init_funds" yaml:"init_funds"`
}

func (msg MsgInstantiateContract) Route() string {
	return RouterKey
}

func (msg MsgInstantiateContract) Type() string {
	return "instantiate"
}

func (msg MsgInstantiateContract) ValidateBasic() sdk.Error {
	if msg.InitFunds.IsAnyNegative() {
		return sdk.ErrInvalidCoins("negative InitFunds")
	}
	return nil
}

func (msg MsgInstantiateContract) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgInstantiateContract) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

type MsgExecuteContract struct {
	Sender    sdk.AccAddress  `json:"sender" yaml:"sender"`
	Contract  sdk.AccAddress  `json:"contract" yaml:"contract"`
	Msg       json.RawMessage `json:"msg" yaml:"msg"`
	SentFunds sdk.Coins       `json:"sent_funds" yaml:"sent_funds"`
}

func (msg MsgExecuteContract) Route() string {
	return RouterKey
}

func (msg MsgExecuteContract) Type() string {
	return "execute"
}

func (msg MsgExecuteContract) ValidateBasic() sdk.Error {
	if msg.SentFunds.IsAnyNegative() {
		return sdk.ErrInvalidCoins("negative SentFunds")
	}
	return nil
}

func (msg MsgExecuteContract) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgExecuteContract) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}
