# Ticker wall dev shortcuts — run with `just <recipe>` (https://github.com/casey/just).
# The leader needs a Massive.com API key in TW_API_KEY.

# List available recipes.
default:
    @just --list

# Build the workspace (debug).
build:
    cargo build

# Build the optimized release binary (target/release/tickerwall).
release:
    cargo build --release

# Run the test suite.
test:
    cargo test --workspace

# Format check + clippy — matches CI.
lint:
    cargo fmt --all --check
    cargo clippy --workspace --all-targets -- -D warnings

# Auto-format the code.
fmt:
    cargo fmt --all

# Run the leader. Extra flags pass through, e.g. `just server --tickers AAPL,NVDA`.
server *ARGS:
    cargo run --bin tickerwall -- server {{ARGS}}

# Run one GUI screen at the given index (default 10).
gui INDEX="10":
    cargo run --bin tickerwall -- gui --screen-index {{INDEX}}

# Build, then run the leader + two GUI screens together (needs TW_API_KEY).
run:
    #!/usr/bin/env bash
    set -euo pipefail
    cargo build
    ./target/debug/tickerwall server &
    leader=$!
    trap 'kill $leader 2>/dev/null || true' EXIT
    sleep 4
    ./target/debug/tickerwall gui --screen-index 10 &
    ./target/debug/tickerwall gui --screen-index 20 &
    wait
