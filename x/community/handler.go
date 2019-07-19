package community

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewHandler creates a new handler for all community messages
func NewHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case MsgNewCommunity:
			return handleMsgNewCommunity(ctx, k, msg)
		case MsgAddAdmin:
			return handleMsgAddAdmin(ctx, k, msg)
		case MsgRemoveAdmin:
			return handleMsgRemoveAdmin(ctx, k, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized community message type: %T", msg)
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleMsgNewCommunity(ctx sdk.Context, k Keeper, msg MsgNewCommunity) sdk.Result {
	if err := msg.ValidateBasic(); err != nil {
		return err.Result()
	}

	community, err := k.NewCommunity(ctx, msg.ID, msg.Name, msg.Description, msg.Creator)
	if err != nil {
		return err.Result()
	}

	res, jsonErr := json.Marshal(community)
	if jsonErr != nil {
		return sdk.ErrInternal(fmt.Sprintf("Marshal result error: %s", jsonErr)).Result()
	}

	return sdk.Result{
		Data: res,
	}
}

func handleMsgAddAdmin(ctx sdk.Context, k Keeper, msg MsgAddAdmin) sdk.Result {
	if err := msg.ValidateBasic(); err != nil {
		return err.Result()
	}

	err := k.AddAdmin(ctx, msg.Admin, msg.Creator)
	if err != nil {
		return err.Result()
	}

	res, jsonErr := json.Marshal(true)
	if jsonErr != nil {
		return sdk.ErrInternal(fmt.Sprintf("Marshal result error: %s", jsonErr)).Result()
	}

	return sdk.Result{
		Data: res,
	}
}

func handleMsgRemoveAdmin(ctx sdk.Context, k Keeper, msg MsgRemoveAdmin) sdk.Result {
	if err := msg.ValidateBasic(); err != nil {
		return err.Result()
	}

	err := k.RemoveAdmin(ctx, msg.Admin, msg.Remover)
	if err != nil {
		return err.Result()
	}

	res, jsonErr := json.Marshal(true)
	if jsonErr != nil {
		return sdk.ErrInternal(fmt.Sprintf("Marshal result error: %s", jsonErr)).Result()
	}

	return sdk.Result{
		Data: res,
	}
}
