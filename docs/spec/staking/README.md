# Staking module specification

## Summary

The staking module contains all data types and state transitions needed to stake on Arguments. Arguments are how claims are supported or refuted. Arguments contain a text body that serves to *Back* or *Challenge* a specific claim with a specific amount of trustake.

Furthermore, Arguments can be staked with an *Upvote* designation. It further enhances the standing of a claim, making the case for it stronger.

## State

### Main Types

The `Stake` type stores all data necessarily for a staking action. It is designed to be embedded by every type that requires staking functionality. For example, it is embedded in `Argument` and `Upvote` types.

```go
type Stake struct {
    ID           int64
    ArgumentID   int64
    Type         StakeType
    Amount       sdk.Coin
    Creator      sdk.AccAddress
    CreatedTime  time.Time
    EndTime      time.Time
}

// stake type enum
type StakeType int
const (
    Backing StakeType = iota    // 0
    Challenge                   // 1
    Upvote                      // 2
)

// Params can be voted on by governance
type Params struct {
    Period                  time.Time       // default = 7 days
    ArgumentCreationStake   sdk.Coin        // default = 50 trustake
    UpvoteStake             sdk.Coin        // default = 10 trustake
    CreatorShare            sdk.Dec         // default = 50%
    InterestRate            sdk.Dec         // default = 25%
    StakeLimitPercent       sdk.Dec         // default = 66.7%
    StakeLimitDays          time.Duration   // default = 7 days
    UnjailUpvotes           int             // default = 1
}
```

An `Argument` contains all data for an argument that either supports (back) or refutes (challenge) a claim. It also embeds a `Stake` type for staking data storage.

```go
type Argument struct {
    *Stake

    ClaimID              int64
    Summary              string
    Body                 string
    TotalUpvoted         sdk.Coin
    SlashCount           int
    IsUnhelpful          bool
    UpdatedTime          time.Time
}
```

An `Upvote` wraps a `Stake` and represents an upvote.

```go
type Upvote struct {
    *Stake
}
```

### Associations

`ArgumentStakes` maintains an easily accessible list of all stakes for each argument.

```go
// "argument:id:XX:staketype:XX:creator:XX" -> nil
type ArgumentStakes app.UserList
```

### Queues

`ActiveStakes` maintains a queue of all currently active stakes, sorted by `EndTime`.

```go
type ActiveStakes Queue
```

## State Transitions
### Messages

`CreateArgumentMsg` creates an `Argument` in the module's key-value store. The only allowed values for `StakeType` are 0 (back), and 1 (challenge). 

`EndTime` is currently fixed at 7 days. 

The stake `Amount` is currently fixed at 50 trustake. In the future, this will be a value algorithmically determined based on various factors such as the current amount staked on the claim, total supply of trustake, and the health of the community associated with the claim.

Only users who have surpassed their jail time are allowed to create arguments. Each user may create one backing argument, and one challenge argument per claim.

```go
type CreateArgumentMsg struct {
    ClaimID       int64
    Summary       string
    Body          string
    StakeType     int             // back  or challenge
    Creator       sdk.AccAddress
}
```

An argument currently cannot be edited.

An argument's standing can be enhanced with an `UpvoteArgumentMsg` with some stake. The stake `Amount` is currently fixed at 10 trustake.

An argument creator cannot upvote their own argument.

Check if the upvote is on an argument by a user who is jailed. Check their jail status and unjail them. Un-jailing should reset the creator's `SlashCount` and `IsJailed` status.

```go
type UpvoteArgumentMsg struct {
    ArgumentID    int64
    Creator       sdk.AccAddress
}
```

If no actions has been taken on an `Argument`, allow the original creator of an argument to delete it.

```go
type DeleteArgumentMsg struct {
    ID            int64
    Creator       sdk.AccAddress
}
```

Staking via `CreateArgumentMsg` and `UpvoteArgumentMsg` should fail validation if the creator has already staked over 66% of their total trustake within a 7-day rolling period. 

## Block Triggers

### End Block

After each block is processed, check the `ActiveStakes` queue for expiring stakes. After a stake has ended, distribute rewards.

Rewards:
* argument creators get `CreatorShare` interest reward from each staker
* stakers keep (1 - `CreatorShare`) interest

This incentive structure heavily rewards argument creation as creators get 50% of the interest from multiple upvoters. Upvoting is a lightweight way to earn 50% interest. But to earn full interest and rewards, content creators are encouraged to write arguments.

Interest is calculated based on the time the stake was placed, using the annual `InterestRate` param.