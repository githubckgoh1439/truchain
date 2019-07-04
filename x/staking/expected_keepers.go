package staking

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankexported "github.com/TruStory/truchain/x/bank/exported"
	"github.com/TruStory/truchain/x/claim"
)

type AccountKeeper interface {
	IsJailed(ctx sdk.Context, address sdk.AccAddress) (bool, sdk.Error)
	UnJail(ctx sdk.Context, address sdk.AccAddress) sdk.Error
}

type ClaimKeeper interface {
	Claim(ctx sdk.Context, id uint64) (claim claim.Claim, ok bool)
}

// BankKeeper is the expected bank keeper interface for this module
type BankKeeper interface {
	AddCoin(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coin,
		referenceID uint64, txType bankexported.TransactionType) (sdk.Coins, sdk.Error)
	GetCoins(ctx sdk.Context, address sdk.AccAddress) sdk.Coins
	SubtractCoin(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coin,
		referenceID uint64, txType TransactionType) (sdk.Coins, sdk.Error)
	TransactionsByAddress(ctx sdk.Context, address sdk.AccAddress, filterSetters ...bankexported.Filter) []bankexported.Transaction
}