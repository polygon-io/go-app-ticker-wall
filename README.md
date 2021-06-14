# Ticker Wall

For linux you will need: `libgl1-mesa-dev` and `xorg-dev` packages.

To run:
`go run cmd/client/*.go`

To run second screen(or more):
`SCREEN_INDEX=2 go run cmd/client/*.go`

### APIs

There is a RESTful HTTP API which you can interact with to update the ticker wall in real-time.

Updating presentation data:

```
POST /v1/presentation
{
    "tickerBoxWidth": 1400,
    "scrollSpeed": 20
}
```

To display an announcement

```
POST /v1/announcement
{
    "message": "testing",
    "lifespanMS": 5000, // How long the message stays visable ( ms ).
    "announcementType":"info", // info|danger|success
    "animation": "ease" // ease|bounce|back|elastic(Default)
}
```

This will slow down the scroll speed, and increase the width of each ticker symbol.

### TODO / Wish List

These are not in order of priority.

- CLI for interacting with cluster.
- - Ability to add tickers while app is running.
- - All the current REST endpoint interactions.
- WebSocket reconnect logic ( we really need our go client library ).
- gRPC reconnect logic to leader.
- Potentially make charts instead of logos.
- Run inside docker container.
- Some kind of build process. tests?

- Maybe v2.0? - Instead of 2 separate binaries, use raft to establish the leader. 1 binary.
