//! Live smoke tests against the real Massive.com API. Ignored by default so CI /
//! `cargo test` stay offline. Run manually with a key in the environment:
//!
//!   TW_API_KEY=xxx cargo test -p tickerwall-data --test live_smoke -- --ignored --nocapture
//!
//! REST works any time; live WS prices only flow during market hours, so the WS
//! test just verifies the connect/auth/subscribe handshake succeeds.

use std::time::Duration;

use tickerwall_data::{latest_trading_day, MarketClient};

fn api_key() -> Option<String> {
    std::env::var("TW_API_KEY").ok().filter(|k| !k.is_empty())
}

#[tokio::test]
#[ignore = "hits the live API; requires TW_API_KEY"]
async fn rest_smoke() {
    let Some(key) = api_key() else {
        eprintln!("TW_API_KEY not set — skipping");
        return;
    };
    let client = MarketClient::new(key, false);

    let ticker = client
        .load_ticker_data("AAPL")
        .await
        .expect("load_ticker_data(AAPL)");
    eprintln!(
        "AAPL -> name={:?} price={} prev_close={} shares={}",
        ticker.company_name, ticker.price, ticker.previous_close_price, ticker.outstanding_shares
    );
    assert!(!ticker.company_name.is_empty(), "expected a company name");
    assert!(ticker.previous_close_price > 0.0, "expected a prev close");

    let aggs = client
        .get_today_aggs(latest_trading_day(), "AAPL", 10)
        .await
        .expect("get_today_aggs(AAPL)");
    eprintln!(
        "AAPL -> {} aggregate bars for {}",
        aggs.len(),
        latest_trading_day()
    );
}

#[tokio::test]
#[ignore = "hits the live API; requires TW_API_KEY"]
async fn ws_handshake_smoke() {
    let Some(key) = api_key() else {
        eprintln!("TW_API_KEY not set — skipping");
        return;
    };
    let client = MarketClient::new(key, false);
    let (tx, mut rx) = tokio::sync::mpsc::channel(100);
    let cancel = tokio_util::sync::CancellationToken::new();

    let listener = {
        let client = client.clone();
        let cancel = cancel.clone();
        tokio::spawn(async move {
            client
                .listen_for_price_updates(&["AAPL".into(), "MSFT".into()], tx, cancel)
                .await
        })
    };

    // Either a live tick arrives (market open) or we time out (market closed).
    // Either way, a clean return from the listener means auth+subscribe worked.
    tokio::select! {
        Some(update) = rx.recv() => {
            eprintln!("live update: {} = {}", update.symbol, update.price);
        }
        _ = tokio::time::sleep(Duration::from_secs(6)) => {
            eprintln!("no live data in 6s (market likely closed) — handshake OK if no error below");
        }
    }

    cancel.cancel();
    listener
        .await
        .expect("listener task panicked")
        .expect("websocket listener returned an error");
}
