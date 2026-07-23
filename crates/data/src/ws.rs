//! WebSocket price feed. Authenticates, subscribes to per-trade or per-second
//! aggregate topics, and forwards each price tick onto an mpsc channel. Mirrors
//! `massive_client/ws.go`.

use anyhow::{Context, Result};
use futures_util::{SinkExt, StreamExt};
use serde_json::Value;
use tickerwall_proto::PriceUpdate;
use tokio::sync::mpsc;
use tokio_tungstenite::tungstenite::Message;
use tokio_util::sync::CancellationToken;

use crate::MarketClient;

impl MarketClient {
    /// Connect, authenticate, subscribe to `symbols`, and forward live prices to
    /// `tx` until `cancel` fires, the receiver is dropped, or the socket closes.
    /// The caller is responsible for reconnect/retry.
    pub async fn listen_for_price_updates(
        &self,
        symbols: &[String],
        tx: mpsc::Sender<PriceUpdate>,
        cancel: CancellationToken,
    ) -> Result<()> {
        let (ws, _) = tokio_tungstenite::connect_async(&self.ws_url)
            .await
            .with_context(|| format!("websocket connect: {}", self.ws_url))?;
        let (mut write, mut read) = ws.split();

        // Authenticate immediately; subscribe once the server acks auth_success.
        let auth = serde_json::json!({ "action": "auth", "params": self.api_key }).to_string();
        write
            .send(Message::Text(auth.into()))
            .await
            .context("send auth")?;

        // "A" = per-second aggregates, "T" = individual trades.
        let prefix = if self.per_tick_updates { "T" } else { "A" };
        let params = symbols
            .iter()
            .map(|s| format!("{prefix}.{s}"))
            .collect::<Vec<_>>()
            .join(",");
        let subscribe =
            serde_json::json!({ "action": "subscribe", "params": params }).to_string();

        loop {
            tokio::select! {
                _ = cancel.cancelled() => {
                    let _ = write.send(Message::Close(None)).await;
                    return Ok(());
                }
                msg = read.next() => {
                    let msg = match msg {
                        Some(m) => m.context("websocket read")?,
                        None => return Ok(()), // stream ended
                    };
                    match msg {
                        Message::Text(text) => {
                            for event in parse_events(text.as_str()) {
                                if is_auth_success(&event) {
                                    write
                                        .send(Message::Text(subscribe.clone().into()))
                                        .await
                                        .context("send subscribe")?;
                                    continue;
                                }
                                if let Some(update) = price_update_from_event(&event) {
                                    if tx.send(update).await.is_err() {
                                        return Ok(()); // receiver gone
                                    }
                                }
                            }
                        }
                        Message::Ping(payload) => {
                            let _ = write.send(Message::Pong(payload)).await;
                        }
                        Message::Close(_) => return Ok(()),
                        _ => {}
                    }
                }
            }
        }
    }
}

/// Split a server frame into individual events. Frames are usually a JSON array
/// of event objects, but tolerate a single object too.
fn parse_events(text: &str) -> Vec<Value> {
    match serde_json::from_str::<Value>(text) {
        Ok(Value::Array(events)) => events,
        Ok(other) => vec![other],
        Err(_) => Vec::new(),
    }
}

/// True if this is the control message telling us authentication succeeded.
fn is_auth_success(event: &Value) -> bool {
    event.get("ev").and_then(Value::as_str) == Some("status")
        && event.get("status").and_then(Value::as_str) == Some("auth_success")
}

/// Extract a price update from a data event, if this event carries a price.
/// `A`/`AM` (aggregates) use close `c`; `T` (trade) uses price `p`.
fn price_update_from_event(event: &Value) -> Option<PriceUpdate> {
    let ev = event.get("ev").and_then(Value::as_str)?;
    let symbol = event.get("sym").and_then(Value::as_str)?.to_string();
    let price = match ev {
        "A" | "AM" => event.get("c").and_then(Value::as_f64)?,
        "T" => event.get("p").and_then(Value::as_f64)?,
        _ => return None,
    };
    Some(PriceUpdate { symbol, price })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_second_aggregate() {
        let events = parse_events(r#"[{"ev":"A","sym":"AAPL","c":211.34,"v":1000}]"#);
        let update = price_update_from_event(&events[0]).unwrap();
        assert_eq!(update.symbol, "AAPL");
        assert_eq!(update.price, 211.34);
    }

    #[test]
    fn parses_trade() {
        let events = parse_events(r#"[{"ev":"T","sym":"NVDA","p":123.45,"s":10}]"#);
        let update = price_update_from_event(&events[0]).unwrap();
        assert_eq!(update.symbol, "NVDA");
        assert_eq!(update.price, 123.45);
    }

    #[test]
    fn recognizes_auth_success_and_ignores_status_events() {
        let events = parse_events(r#"[{"ev":"status","status":"auth_success","message":"ok"}]"#);
        assert!(is_auth_success(&events[0]));
        assert!(price_update_from_event(&events[0]).is_none());
    }

    #[test]
    fn handles_multiple_events_in_one_frame() {
        let events = parse_events(
            r#"[{"ev":"status","status":"connected"},{"ev":"A","sym":"MSFT","c":400.0}]"#,
        );
        assert_eq!(events.len(), 2);
        assert!(!is_auth_success(&events[0]));
        assert_eq!(price_update_from_event(&events[1]).unwrap().price, 400.0);
    }
}
