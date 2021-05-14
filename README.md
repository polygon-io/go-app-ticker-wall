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
    "message": "Anonymous message...",
    "lifespanms": 5000
}
```

This will slow down the scroll speed, and increase the width of each ticker symbol.
