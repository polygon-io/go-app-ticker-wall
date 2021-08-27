# Ticker Wall

The Polygon.io ticker wall is an open source, cross platform, scalable ticker tape. It is meant to be scalable across many machines to eliminate the need for expensive specialty hardware for achieving a scrolling ticker tape. It is cross platform compatible, so it runs on mac, windows or linux ( only mac and linux tested so far ). There are APIs which allow you to control most settings of the display while it's running. It also has APIs for displaying announcements with different animation choices.

We use it at the Polygon.io office, but we also wanted it to be general enough to suite a broad group of needs, so most interactions and settings are configurable via the API or CLI.

# Getting Started

There are 2 components to a ticker wall cluster. 1x Leader and N number of GUIs. The leader can also be run on the same system as a GUI, and there is no minimum for the number of GUIs. You can start with 1 screen, then continue to add more and it will dynamically adjust in real-time.

## To run the Leader with default settings

This must be run in parallel with GUIs.

`LEADER_API_KEY=myPolygonApiKey go run ./cmd/leader`

## To run the GUI with default settings

`LEADER=localhost:6886 go run ./cmd/gui`

## To run a second GUI with default settings

`LEADER=localhost:6886 SCREEN_INDEX=2 go run ./cmd/gui`

# Prerequisites

On linux it requires X11. So you will need: `libgl1-mesa-dev` and `xorg-dev` packages.

# Configuration ( ENV Variables )

## Leader

- `LEADER_TICKER_LIST` Is a comma separated list of ticker symbols we want on the ticker tape. Default: `AAPL,AMD,NVDA,SBUX,FB,HOOD`
- `LEADER_API_KEY` Is your Polygon.io API Key which is used to stream the data. Currently it requires a real-time data subscription.
- `LEADER_PRESENTATION_TICKER_BOX_WIDTH` Is the pixel width of an individual tickers box. Default: `1100`
- `LEADER_PRESENTATION_SCROLL_SPEED` Sets how fast the ticker tape moves. This is inverse, so lower=faster. Default: `15`
- `LEADER_PRESENTATION_ANIMATION_DURATION` Sets how fast animation durations are in milliseconds. Default: `500`
- `LEADER_PRESENTATION_PER_TICK_UPDATES` If true sets the display to update for every trade executed. False updates once per second max. Default: `500`
- `LEADER_PRESENTATION_UP_COLOR` RGBA Value of the color which a stock is 'up'. Default (green): `red:51,green:255,blue:51,alpha:255`
- `LEADER_PRESENTATION_DOWN_COLOR` RGBA Value of the color which a stock is 'down'. Default (red): `red:255,green:51,blue:51,alpha:255`
- `LEADER_PRESENTATION_FONT_COLOR` RGBA Value of the normal text. Default (white): `red:255,green:255,blue:255,alpha:255`
- `LEADER_PRESENTATION_BG_COLOR` RGBA Value of the background. Default (black): `red:1,green:1,blue:1,alpha:255`
- `LEADER_PRESENTATION_TICKER_BOX_BG_COLOR` RGBA Value of the background for a ticker box. Default (dark gray): `red:20,green:20,blue:20,alpha:255`

## GUI

- `LEADER` Is the hostname and gRPC port of the leader endpoint. Default: `localhost:6886`
- `SCREEN_WIDTH` Is the width of the GUI. Default: `1920`
- `SCREEN_HEIGHT` Is the height of the GUI. Default: `300`
- `SCREEN_INDEX` Is the index of the GUI. Eg: First screen `1` Second screen `2` and so on. Default: `1`

# APIs

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

# TODO / Wish List

These are not in order of priority.

- CLI for interacting with cluster.
  - Ability to add tickers while app is running.
  - All the current REST endpoint interactions.
- WebSocket reconnect logic ( we really need our go client library ).
- gRPC reconnect logic to leader.
- Potentially make charts instead of logos.
- Run inside docker container.
- Some kind of build process. tests?

- Maybe v2.0? - Instead of 2 separate binaries, use raft to establish the leader. 1 binary.

### Data

- Ticker Data
  - Pricing Information
  - Company Details
- Presentation Data
  - Local Presentation Data ( screen width, height, index )
  - Cluster Presentation Data ( other screen details )
    - Variables
      - Background Color
      - Up Color
      - Down Color

### Presentation

- Rendering
  - Global Ticker Tape
  - Special Messages / Alerts
  - Non OK state ( cannot connect to leader, etc )
