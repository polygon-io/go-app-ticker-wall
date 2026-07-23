# Ticker Wall — Rust Rewrite

Rust reimplementation of the ticker wall (see `docs/ARCHITECTURE.md` for the Go original
this replaces, and the approved plan for the design). The Cargo workspace lives alongside the
Go module during the transition.

## Workspace layout

| Crate | Role |
|---|---|
| `crates/proto` | Redesigned protobuf schema + `tonic`/`prost` generated code, and domain helpers (cluster geometry, settings merge, ticker ordering). |
| `crates/data` | Massive.com (Polygon) REST + WebSocket market-data client. |
| `crates/leader` | Authoritative state, data-refresh loops, broadcast bus, tonic `Leader` service. |
| `crates/client` | GUI-side cluster client: join-stream, state sync, auto-reconnect. |
| `crates/gui` | `winit` + `glutin` + `femtovg` render loop (tape, graphs, notifications, system panel). |
| `crates/app` | The `tickerwall` binary: `server` / `gui` / `update` / `announce` / `describe`. |

## Build

```
cargo build --release
```

Linux build prerequisites (X11 + OpenGL dev headers), same as the Go app:

```
# Debian/Ubuntu
sudo apt-get install -y libgl1-mesa-dev xorg-dev
```

macOS needs no extra packages.

## Run (development, e.g. on macOS)

```
# 1. Leader (pulls data, serves gRPC on :6886)
tickerwall server -a <MASSIVE_API_KEY>          # or set TW_API_KEY

# 2. One or more GUI screens (each is one window)
tickerwall gui --screen-index 10
tickerwall gui --screen-index 20 --screen-width 1920

# 3. Live control
tickerwall update --scroll-speed 5              # partial: only this field changes
tickerwall update --bg-color 255,255,255,255
tickerwall announce "Big Success!" --type success --animation ease
tickerwall describe
```

Config precedence is **flags > environment**. Environment variables are the flag name
uppercased with a `TW_` prefix (e.g. `TW_API_KEY`). A config-file layer (the Go app's
`tickerwall.{yml,json,toml}`) is not yet implemented — tracked as a follow-up.

## Linux kiosk deployment (bare X, no window manager)

The deployed screens boot straight into the app with **no window manager**. As with the Go
build (GLFW required X11/Wayland), we run a bare X server and launch the GUI fullscreen — no
desktop environment or WM. `winit` + `glutin` render fine under a bare X server.

Minimal setup: log a kiosk user in on a TTY and start X with only the app as its client.

`~/.xinitrc` (the *only* X client — when it exits, X exits):

```sh
#!/bin/sh
# No window manager. Just the ticker wall, fullscreen.
exec tickerwall gui --leader http://<LEADER_HOST>:6886 --screen-index 10 \
    --screen-width 1920 --screen-height 1080
```

Auto-start X on login (e.g. append to `~/.bash_profile`), restarting if it ever exits:

```sh
if [ -z "$DISPLAY" ] && [ "$(tty)" = "/dev/tty1" ]; then
  while true; do startx; sleep 1; done
fi
```

Notes:
- The window is created at the requested size; on a dedicated display with no WM it fills the
  screen. (A future option can force borderless-fullscreen via winit's fullscreen mode.)
- `femtovg`'s OpenGL ES backend and surfaceless support keep a future **true DRM/KMS** path
  (no X server at all) open without changing the drawing code — `winit` has no KMS backend
  today, so that path would bypass winit. Not built now.
- Run the leader as a normal systemd service on one host; the screens only need the GUI.

## Tests

```
cargo test                    # unit tests across all crates (offline)

# Opt-in live tests against the real API (requires a key):
TW_API_KEY=xxx cargo test -p tickerwall-data --test live_smoke -- --ignored --nocapture
```

## Debt fixed vs. the Go original (see ARCHITECTURE.md §12)

- Partial settings updates merge instead of clobbering unset fields (proven: `update
  --scroll-speed` leaves other settings intact).
- `Update` is a real `oneof`; dynamic `add_ticker`/`remove_ticker` are wired end-to-end and
  actually emitted (with a live price-feed re-subscribe).
- Aggregates diff by content, not slice length.
- Single announcement-timestamp path; named refresh-interval constants.
- Layout math is unit-tested and the "screen wider than the tape" gap bug is fixed (tickers
  wrap and repeat to fill instead of leaving holes).
- No FPS-graph memory-leak workaround; dead logo code omitted.
