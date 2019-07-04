package bank

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// NewQuerier creates a new querier
func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, sdk.Error) {
		switch path[0] {
		case QueryTransactionsByAddress:
			return queryTransactionsByAddress(ctx, req, keeper)
		default:
			return nil, sdk.ErrUnknownRequest("Unknown bank query endpoint")
		}
	}
}

func queryTransactionsByAddress(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QueryTransactionsByAddressParams
	err := keeper.codec.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, ErrInvalidQueryParams(err)
	}
	sortOrder := SortAsc
	if params.SortOrder.Valid() {
		sortOrder = params.SortOrder
	}
	transactions := keeper.TransactionsByAddress(ctx,
		params.Address,
		FilterByTransactionType(params.Types...),
		SortOrder(sortOrder),
		Limit(params.Limit),
		Offset(params.Offset),
	)
	return keeper.codec.MustMarshalJSON(transactions), nil
}
