package app

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/argument"
	"github.com/TruStory/truchain/x/backing"
	"github.com/TruStory/truchain/x/category"
	"github.com/TruStory/truchain/x/challenge"
	"github.com/TruStory/truchain/x/expiration"
	clientParams "github.com/TruStory/truchain/x/params"
	"github.com/TruStory/truchain/x/stake"
	"github.com/TruStory/truchain/x/story"
	"github.com/TruStory/truchain/x/truapi"
	trubank "github.com/TruStory/truchain/x/trubank"
	"github.com/TruStory/truchain/x/users"
	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tmlibs/cli"
)

// default home directories for expected binaries
var (
	DefaultCLIHome  = os.ExpandEnv("$HOME/.trucli")
	DefaultNodeHome = os.ExpandEnv("$HOME/.truchaind")
)

// TruChain implements an extended ABCI application. It contains a BaseApp,
// a codec for serialization, KVStore keys for multistore state management, and
// various mappers and keepers to manage getting, setting, and serializing the
// integral app types.
type TruChain struct {
	*bam.BaseApp
	codec *codec.Codec

	// keys to access the multistore
	keyAccount    *sdk.KVStoreKey
	keyArgument   *sdk.KVStoreKey
	keyBacking    *sdk.KVStoreKey
	keyCategory   *sdk.KVStoreKey
	keyChallenge  *sdk.KVStoreKey
	keyExpiration *sdk.KVStoreKey
	keyFee        *sdk.KVStoreKey
	keyIBC        *sdk.KVStoreKey
	keyMain       *sdk.KVStoreKey
	keyStake      *sdk.KVStoreKey
	keyStory      *sdk.KVStoreKey
	keyStoryQueue *sdk.KVStoreKey
	keyTruBank    *sdk.KVStoreKey
	keyParams     *sdk.KVStoreKey
	tkeyParams    *sdk.TransientStoreKey

	// manage getting and setting accounts
	accountKeeper       auth.AccountKeeper
	feeCollectionKeeper auth.FeeCollectionKeeper
	bankKeeper          bank.Keeper
	ibcMapper           ibc.Mapper
	paramsKeeper        params.Keeper

	// access truchain multistore
	argumentKeeper     argument.Keeper
	backingKeeper      backing.Keeper
	categoryKeeper     category.Keeper
	challengeKeeper    challenge.Keeper
	clientParamsKeeper clientParams.Keeper
	expirationKeeper   expiration.Keeper
	storyKeeper        story.Keeper
	stakeKeeper        stake.Keeper
	truBankKeeper      trubank.Keeper

	// state to run api
	blockCtx     *sdk.Context
	blockHeader  abci.Header
	api          *truapi.TruAPI
	apiStarted   bool
	registrarKey secp256k1.PrivKeySecp256k1
}

// NewTruChain returns a reference to a new TruChain. Internally,
// a codec is created along with all the necessary keys.
// In addition, all necessary mappers and keepers are created, routes
// registered, and finally the stores being mounted along with any necessary
// chain initialization.
func NewTruChain(logger log.Logger, db dbm.DB, loadLatest bool, options ...func(*bam.BaseApp)) *TruChain {
	// create and register app-level codec for TXs and accounts
	codec := MakeCodec()

	loadEnvVars()

	// create your application type
	var app = &TruChain{
		BaseApp: bam.NewBaseApp(types.AppName, logger, db, auth.DefaultTxDecoder(codec), options...),
		codec:   codec,

		keyParams:  sdk.NewKVStoreKey("params"),
		tkeyParams: sdk.NewTransientStoreKey("transient_params"),

		keyMain:       sdk.NewKVStoreKey("main"),
		keyAccount:    sdk.NewKVStoreKey("acc"),
		keyIBC:        sdk.NewKVStoreKey("ibc"),
		keyArgument:   sdk.NewKVStoreKey(argument.StoreKey),
		keyStory:      sdk.NewKVStoreKey(story.StoreKey),
		keyStoryQueue: sdk.NewKVStoreKey(story.QueueStoreKey),
		keyCategory:   sdk.NewKVStoreKey(category.StoreKey),
		keyBacking:    sdk.NewKVStoreKey(backing.StoreKey),
		keyChallenge:  sdk.NewKVStoreKey(challenge.StoreKey),
		keyExpiration: sdk.NewKVStoreKey(expiration.StoreKey),
		keyFee:        sdk.NewKVStoreKey("fee_collection"),
		keyStake:      sdk.NewKVStoreKey(stake.StoreKey),
		keyTruBank:    sdk.NewKVStoreKey(trubank.StoreKey),
		api:           nil,
		apiStarted:    false,
		blockCtx:      nil,
		blockHeader:   abci.Header{},
		registrarKey:  loadRegistrarKey(),
	}

	// The ParamsKeeper handles parameter storage for the application
	app.paramsKeeper = params.NewKeeper(app.codec, app.keyParams, app.tkeyParams)

	// The AccountKeeper handles address -> account lookups
	app.accountKeeper = auth.NewAccountKeeper(
		app.codec,
		app.keyAccount,
		app.paramsKeeper.Subspace(auth.DefaultParamspace),
		auth.ProtoBaseAccount,
	)

	app.bankKeeper = bank.NewBaseKeeper(
		app.accountKeeper,
		app.paramsKeeper.Subspace(bank.DefaultParamspace),
		bank.DefaultCodespace,
	)

	app.ibcMapper = ibc.NewMapper(app.codec, app.keyIBC, ibc.DefaultCodespace)
	app.feeCollectionKeeper = auth.NewFeeCollectionKeeper(app.codec, app.keyFee)

	// wire up keepers
	app.categoryKeeper = category.NewKeeper(
		app.keyCategory,
		codec,
	)

	app.storyKeeper = story.NewKeeper(
		app.keyStory,
		app.keyStoryQueue,
		app.categoryKeeper,
		app.paramsKeeper.Subspace(story.StoreKey),
		app.codec,
	)

	app.argumentKeeper = argument.NewKeeper(
		app.keyArgument,
		app.storyKeeper,
		app.paramsKeeper.Subspace(argument.StoreKey),
		app.codec)

	app.truBankKeeper = trubank.NewKeeper(
		app.keyTruBank,
		app.bankKeeper,
		app.categoryKeeper,
		app.codec)

	app.stakeKeeper = stake.NewKeeper(
		app.storyKeeper,
		app.truBankKeeper,
		app.paramsKeeper.Subspace(stake.StoreKey),
	)

	app.backingKeeper = backing.NewKeeper(
		app.keyBacking,
		app.argumentKeeper,
		app.stakeKeeper,
		app.storyKeeper,
		app.bankKeeper,
		app.truBankKeeper,
		app.categoryKeeper,
		codec,
	)

	app.challengeKeeper = challenge.NewKeeper(
		app.keyChallenge,
		app.argumentKeeper,
		app.stakeKeeper,
		app.backingKeeper,
		app.truBankKeeper,
		app.bankKeeper,
		app.storyKeeper,
		app.paramsKeeper.Subspace(challenge.StoreKey),
		codec,
	)

	app.expirationKeeper = expiration.NewKeeper(
		app.keyExpiration,
		app.keyStoryQueue,
		app.stakeKeeper,
		app.storyKeeper,
		app.backingKeeper,
		app.challengeKeeper,
		app.paramsKeeper.Subspace(expiration.StoreKey),
		codec,
	)

	app.clientParamsKeeper = clientParams.NewKeeper(
		app.argumentKeeper,
		app.backingKeeper,
		app.challengeKeeper,
		app.expirationKeeper,
		app.stakeKeeper,
		app.storyKeeper,
	)

	// The AnteHandler handles signature verification and transaction pre-processing
	// TODO [shanev]: see https://github.com/TruStory/truchain/issues/364
	// Add this back after fixing issues with signature verification
	// app.SetAnteHandler(auth.NewAnteHandler(app.accountKeeper, app.feeCollectionKeeper))

	// The app.Router is the main transaction router where each module registers its routes
	app.Router().
		AddRoute("bank", bank.NewHandler(app.bankKeeper)).
		AddRoute("ibc", ibc.NewHandler(app.ibcMapper, app.bankKeeper)).
		AddRoute("story", story.NewHandler(app.storyKeeper)).
		AddRoute("category", category.NewHandler(app.categoryKeeper)).
		AddRoute("backing", backing.NewHandler(app.backingKeeper)).
		AddRoute("challenge", challenge.NewHandler(app.challengeKeeper)).
		AddRoute("users", users.NewHandler(app.accountKeeper))

	// The app.QueryRouter is the main query router where each module registers its routes
	app.QueryRouter().
		AddRoute(argument.QueryPath, argument.NewQuerier(app.argumentKeeper)).
		AddRoute(story.QueryPath, story.NewQuerier(app.storyKeeper)).
		AddRoute(category.QueryPath, category.NewQuerier(app.categoryKeeper)).
		AddRoute(users.QueryPath, users.NewQuerier(codec, app.accountKeeper)).
		AddRoute(backing.QueryPath, backing.NewQuerier(app.backingKeeper)).
		AddRoute(challenge.QueryPath, challenge.NewQuerier(app.challengeKeeper)).
		AddRoute(clientParams.QueryPath, clientParams.NewQuerier(app.clientParamsKeeper)).
		AddRoute(trubank.QueryPath, trubank.NewQuerier(app.truBankKeeper))

	// The initChainer handles translating the genesis.json file into initial state for the network
	app.SetInitChainer(app.initChainer)

	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// mount the multistore and load the latest state
	app.MountStores(
		app.keyAccount,
		app.keyParams,
		app.keyBacking,
		app.keyCategory,
		app.keyChallenge,
		app.keyExpiration,
		app.keyFee,
		app.keyIBC,
		app.keyMain,
		app.keyStory,
		app.keyStoryQueue,
		app.keyTruBank,
		app.keyArgument,
	)

	app.MountStoresTransient(app.tkeyParams)

	if loadLatest {
		err := app.LoadLatestVersion(app.keyMain)
		if err != nil {
			cmn.Exit(err.Error())
		}
	}

	// build HTTP api
	app.api = app.makeAPI()

	return app
}

func loadEnvVars() {
	rootdir := viper.GetString(cli.HomeFlag)
	if rootdir == "" {
		rootdir = DefaultNodeHome
	}

	envPath := filepath.Join(rootdir, ".env")
	err := godotenv.Load(envPath)
	if err != nil {
		panic("Error loading .env file")
	}
}

// MakeCodec creates a new codec codec and registers all the necessary types
// with the codec.
func MakeCodec() *codec.Codec {
	cdc := codec.New()

	auth.RegisterCodec(cdc)
	bank.RegisterCodec(cdc)
	staking.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	ibc.RegisterCodec(cdc)

	// register msg types
	story.RegisterAmino(cdc)
	backing.RegisterAmino(cdc)
	category.RegisterAmino(cdc)
	challenge.RegisterAmino(cdc)
	users.RegisterAmino(cdc)

	// register other types
	cdc.RegisterConcrete(&types.AppAccount{}, "types/AppAccount", nil)

	codec.RegisterCrypto(cdc)

	return cdc
}

// BeginBlocker reflects logic to run before any TXs application are processed
// by the application.
func (app *TruChain) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	app.blockCtx = &ctx
	app.blockHeader = req.Header

	if !(app.apiStarted) {
		go app.startAPI()
		app.apiStarted = true
	}

	if app.apiStarted == false && ctx.BlockHeight() > int64(1) {
		panic("API server not started.")
	}

	return abci.ResponseBeginBlock{}
}

// EndBlocker reflects logic to run after all TXs are processed by the
// application.
func (app *TruChain) EndBlocker(ctx sdk.Context, _ abci.RequestEndBlock) abci.ResponseEndBlock {
	tags := app.expirationKeeper.EndBlock(ctx)

	return abci.ResponseEndBlock{Tags: tags}
}

// LoadHeight loads the app at a particular height
func (app *TruChain) LoadHeight(height int64) error {
	return app.LoadVersion(height, app.keyMain)
}

func loadRegistrarKey() secp256k1.PrivKeySecp256k1 {
	rootdir := viper.GetString(cli.HomeFlag)
	if rootdir == "" {
		rootdir = DefaultNodeHome
	}

	keypath := filepath.Join(rootdir, "registrar.key")
	fileBytes, err := ioutil.ReadFile(keypath)

	if err != nil {
		panic(err)
	}

	keyBytes, err := hex.DecodeString(string(fileBytes))

	if err != nil {
		panic(err)
	}

	if len(keyBytes) != 32 {
		panic("Invalid registrar key: " + string(fileBytes))
	}

	key := secp256k1.PrivKeySecp256k1{}

	copy(key[:], keyBytes)

	return key
}
