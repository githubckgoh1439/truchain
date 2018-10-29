package story

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// query endpoints supported by the truchain Querier
const (
	QueryStoriesByCategory           = "category/:id/stories"
	QueryChallengedStoriesByCategory = "category/:id/challenged_stories"
	QueryStoryFeedByCategory         = "category/:id/story_feed"
)

// QueryCategoryStoriesParams are params for stories by category queries
type QueryCategoryStoriesParams struct {
	CategoryID int64
}

// NewQuerier returns a function that handles queries on the KVStore
func NewQuerier(k ReadKeeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QueryStoriesByCategory:
			return queryStoriesWithCategory(ctx, req, k)
		case QueryChallengedStoriesByCategory:
			return queryChallengedStoriesWithCategory(ctx, req, k)
		case QueryStoryFeedByCategory:
			return queryStoryFeed(ctx, req, k)
		default:
			return nil, sdk.ErrUnknownRequest("Unknown truchain query endpoint")
		}
	}
}

// ============================================================================

func queryStoriesWithCategory(
	ctx sdk.Context,
	req abci.RequestQuery,
	k ReadKeeper) (res []byte, err sdk.Error) {

	// get query params
	params, err := unmarshalQueryParams(k, req)
	if err != nil {
		return
	}

	// fetch stories
	stories, err := k.GetStoriesWithCategory(ctx, params.CategoryID)
	if err != nil {
		return
	}

	// return stories JSON bytes
	return marshalStories(k, stories)
}

func queryChallengedStoriesWithCategory(
	ctx sdk.Context,
	req abci.RequestQuery,
	k ReadKeeper) (res []byte, err sdk.Error) {

	// get query params
	params, err := unmarshalQueryParams(k, req)
	if err != nil {
		return
	}

	// fetch challenged stories for category
	stories, err := k.GetChallengedStoriesWithCategory(ctx, params.CategoryID)
	if err != nil {
		return
	}

	// return stories JSON bytes
	return marshalStories(k, stories)
}

func queryStoryFeed(
	ctx sdk.Context,
	req abci.RequestQuery,
	k ReadKeeper) (res []byte, err sdk.Error) {

	// get query params
	params, err := unmarshalQueryParams(k, req)
	if err != nil {
		return
	}

	// fetch stories
	stories, err := k.GetFeedWithCategory(ctx, params.CategoryID)
	if err != nil {
		return
	}

	// return stories JSON bytes
	return marshalStories(k, stories)
}

// unmarshal query params into struct
func unmarshalQueryParams(
	k ReadKeeper,
	req abci.RequestQuery) (params QueryCategoryStoriesParams, sdkErr sdk.Error) {
	err := k.GetCodec().UnmarshalJSON(req.Data, &params)
	if err != nil {
		return params,
			sdk.ErrUnknownRequest(fmt.Sprintf("Incorrectly formatted request data - %s", err.Error()))
	}
	return
}

// marshal stories into pretty JSON bytes
func marshalStories(k ReadKeeper, stories []Story) (res []byte, sdkErr sdk.Error) {
	res, err := codec.MarshalJSONIndent(k.GetCodec(), stories)
	if err != nil {
		panic("Could not marshal result to JSON")
	}
	return
}