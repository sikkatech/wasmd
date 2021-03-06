package keeper

import (
	"encoding/json"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkErrors "github.com/cosmos/cosmos-sdk/types/errors"

	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"

	"github.com/cosmwasm/wasmd/x/wasm/internal/types"
)

const (
	QueryListContracts    = "list-contracts"
	QueryGetContract      = "contract-info"
	QueryGetContractState = "contract-state"
	QueryGetCode          = "code"
	QueryListCode         = "list-code"
)

const (
	QueryMethodContractStateSmart = "smart"
	QueryMethodContractStateAll   = "all"
	QueryMethodContractStateRaw   = "raw"
)

// controls error output on querier - set true when testing/debugging
const debug = false

// NewQuerier creates a new querier
func NewQuerier(keeper Keeper) sdk.Querier {
	q := newQuerier(keeper)
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, sdk.Error) {
		res, err := q(ctx, path, req)
		// convert returned errors
		if err != nil {
			space, code, log := sdkErrors.ABCIInfo(err, debug)
			sdkErr := sdk.NewError(sdk.CodespaceType(space), sdk.CodeType(code), log)
			return nil, sdkErr
		}
		return res, nil
	}
}

// pull this out as a separate function for testing.
// this returns proper error before redacting, NewQuerier is adapting to 0.37 style
func newQuerier(keeper Keeper) func(sdk.Context, []string, abci.RequestQuery) ([]byte, error) {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, error) {
		switch path[0] {
		case QueryGetContract:
			return queryContractInfo(ctx, path[1], req, keeper)
		case QueryListContracts:
			return queryContractList(ctx, req, keeper)
		case QueryGetContractState:
			if len(path) < 3 {
				return nil, sdkErrors.Wrap(sdkErrors.ErrUnknownRequest, "unknown data query endpoint")
			}
			return queryContractState(ctx, path[1], path[2], req, keeper)
		case QueryGetCode:
			return queryCode(ctx, path[1], req, keeper)
		case QueryListCode:
			return queryCodeList(ctx, req, keeper)
		default:
			return nil, sdkErrors.Wrap(sdkErrors.ErrUnknownRequest, "unknown data query endpoint")
		}
	}
}

func queryContractInfo(ctx sdk.Context, bech string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	addr, err := sdk.AccAddressFromBech32(bech)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(err.Error())
	}
	info := keeper.GetContractInfo(ctx, addr)

	bz, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, sdk.ErrInvalidAddress(err.Error())
	}
	return bz, nil
}

func queryContractList(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	var addrs []string
	keeper.ListContractInfo(ctx, func(addr sdk.AccAddress, _ types.ContractInfo) bool {
		addrs = append(addrs, addr.String())
		return false
	})
	bz, err := json.MarshalIndent(addrs, "", "  ")
	if err != nil {
		return nil, sdkErrors.Wrap(sdkErrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

func queryContractState(ctx sdk.Context, bech, queryMethod string, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	contractAddr, err := sdk.AccAddressFromBech32(bech)
	if err != nil {
		return nil, sdkErrors.Wrap(sdkErrors.ErrInvalidAddress, bech)
	}

	var resultData []types.Model
	switch queryMethod {
	case QueryMethodContractStateAll:
		for iter := keeper.GetContractState(ctx, contractAddr); iter.Valid(); iter.Next() {
			resultData = append(resultData, types.Model{
				Key:   string(iter.Key()),
				Value: string(iter.Value()),
			})
		}
		if resultData == nil {
			resultData = make([]types.Model, 0)
		}
	case QueryMethodContractStateRaw:
		resultData = keeper.QueryRaw(ctx, contractAddr, req.Data)
	case QueryMethodContractStateSmart:
		return keeper.QuerySmart(ctx, contractAddr, req.Data)
	default:
		return nil, sdkErrors.Wrap(sdkErrors.ErrUnknownRequest, queryMethod)
	}
	bz, err := json.MarshalIndent(resultData, "", "  ")
	if err != nil {
		return nil, sdkErrors.Wrap(sdkErrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

type GetCodeResponse struct {
	Code []byte `json:"code" yaml:"code"`
}

func queryCode(ctx sdk.Context, codeIDstr string, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	codeID, err := strconv.ParseUint(codeIDstr, 10, 64)
	if err != nil {
		return nil, sdkErrors.Wrap(sdkErrors.ErrUnknownRequest, "invalid codeID: "+err.Error())
	}

	code, err := keeper.GetByteCode(ctx, codeID)
	if err != nil {
		return nil, sdkErrors.Wrap(err, "loading wasm code")
	}

	bz, err := json.MarshalIndent(GetCodeResponse{code}, "", "  ")
	if err != nil {
		return nil, sdkErrors.Wrap(sdkErrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}

type ListCodeResponse struct {
	ID       uint64         `json:"id"`
	Creator  sdk.AccAddress `json:"creator"`
	CodeHash cmn.HexBytes   `json:"code_hash"`
}

func queryCodeList(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, error) {
	var info []ListCodeResponse

	var i uint64
	for true {
		i++
		res := keeper.GetCodeInfo(ctx, i)
		if res == nil {
			break
		}
		info = append(info, ListCodeResponse{
			ID:       i,
			Creator:  res.Creator,
			CodeHash: res.CodeHash,
		})
	}

	bz, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, sdkErrors.Wrap(sdkErrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}
