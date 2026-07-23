//! Market-data client for Massive.com (Polygon). REST for ticker details, prices,
//! and minute aggregates; a WebSocket feed for live prices. Returns
//! `tickerwall-proto` domain types directly. Replaces the Go `massive_client`.

mod ws;

use anyhow::{Context, Result};
use chrono::{Datelike, NaiveDate, TimeZone, Utc, Weekday};
use chrono_tz::America::New_York;
use serde::Deserialize;
use tickerwall_proto::{Agg, Mover, Ticker};

/// Which side of the top-movers snapshot to fetch.
#[derive(Clone, Copy, Debug)]
pub enum MoverDirection {
    Gainers,
    Losers,
}

impl MoverDirection {
    fn path(self) -> &'static str {
        match self {
            MoverDirection::Gainers => "gainers",
            MoverDirection::Losers => "losers",
        }
    }
}

/// Default REST base host (mirrors `massive-com/client-go`).
pub const DEFAULT_REST_BASE: &str = "https://api.massive.com";
/// Default real-time stocks WebSocket endpoint.
pub const DEFAULT_WS_URL: &str = "wss://socket.massive.com/stocks";

/// Client for the Massive.com REST + WebSocket APIs.
#[derive(Clone)]
pub struct MarketClient {
    http: reqwest::Client,
    rest_base: String,
    ws_url: String,
    api_key: String,
    /// When true, subscribe to per-trade updates; otherwise per-second aggregates.
    per_tick_updates: bool,
}

impl MarketClient {
    pub fn new(api_key: impl Into<String>, per_tick_updates: bool) -> Self {
        Self {
            http: reqwest::Client::new(),
            rest_base: DEFAULT_REST_BASE.to_string(),
            ws_url: DEFAULT_WS_URL.to_string(),
            api_key: api_key.into(),
            per_tick_updates,
        }
    }

    /// Override the REST/WS hosts (useful for tests or alternate feeds).
    pub fn with_hosts(mut self, rest_base: impl Into<String>, ws_url: impl Into<String>) -> Self {
        self.rest_base = rest_base.into();
        self.ws_url = ws_url.into();
        self
    }

    async fn get_json<T: for<'de> Deserialize<'de>>(&self, url: &str) -> Result<T> {
        let resp = self
            .http
            .get(url)
            .bearer_auth(&self.api_key)
            .send()
            .await
            .with_context(|| format!("request failed: {url}"))?
            .error_for_status()
            .with_context(|| format!("non-success status: {url}"))?;
        resp.json::<T>()
            .await
            .with_context(|| format!("decode failed: {url}"))
    }

    /// Assemble a full `Ticker` from previous close, last trade, and company details.
    pub async fn load_ticker_data(&self, symbol: &str) -> Result<Ticker> {
        let previous_close_price = self.get_yesterdays_close(symbol).await?;
        let price = self.get_last_trade_price(symbol).await?;
        let details = self.get_ticker_details(symbol).await?;

        Ok(Ticker {
            symbol: symbol.to_string(),
            company_name: details.company_name,
            outstanding_shares: details.outstanding_shares,
            price,
            previous_close_price,
            ..Default::default()
        })
    }

    /// Latest weighted price for a ticker (`GET /v2/last/trade/{ticker}`).
    pub async fn get_last_trade_price(&self, symbol: &str) -> Result<f64> {
        let url = format!("{}/v2/last/trade/{symbol}", self.rest_base);
        let resp: LastTradeResponse = self.get_json(&url).await?;
        Ok(resp.results.price)
    }

    /// Company name + shares outstanding (`GET /v3/reference/tickers/{ticker}`).
    pub async fn get_ticker_details(&self, symbol: &str) -> Result<TickerDetails> {
        let url = format!("{}/v3/reference/tickers/{symbol}", self.rest_base);
        let resp: TickerDetailsResponse = self.get_json(&url).await?;
        Ok(TickerDetails {
            company_name: resp.results.name,
            outstanding_shares: resp.results.weighted_shares_outstanding,
        })
    }

    /// Previous trading day's close (`GET /v2/aggs/ticker/{ticker}/prev`). Accounts
    /// for weekends/holidays server-side; returns 0.0 if the ticker has no history.
    pub async fn get_yesterdays_close(&self, symbol: &str) -> Result<f64> {
        let url = format!("{}/v2/aggs/ticker/{symbol}/prev", self.rest_base);
        let resp: AggsResponse = self.get_json(&url).await?;
        Ok(resp.results.first().map(|a| a.close).unwrap_or(0.0))
    }

    /// Intraday aggregates for the sparkline, `range_size`-minute bars from 09:00 to
    /// 16:30 ET (09:00 start includes significant pre-market). Mirrors
    /// `massive_client.GetTickerTodayAggs`.
    pub async fn get_today_aggs(
        &self,
        day: NaiveDate,
        symbol: &str,
        range_size: i32,
    ) -> Result<Vec<Agg>> {
        let open = New_York
            .with_ymd_and_hms(day.year(), day.month(), day.day(), 9, 0, 0)
            .single()
            .context("invalid market open time")?;
        let close = New_York
            .with_ymd_and_hms(day.year(), day.month(), day.day(), 16, 30, 0)
            .single()
            .context("invalid market close time")?;

        let from_ms = open.timestamp_millis();
        let to_ms = close.timestamp_millis();
        let limit = (close - open).num_minutes().max(1);

        let url = format!(
            "{}/v2/aggs/ticker/{symbol}/range/{range_size}/minute/{from_ms}/{to_ms}\
             ?adjusted=true&sort=asc&limit={limit}",
            self.rest_base
        );
        let resp: AggsResponse = self.get_json(&url).await?;

        Ok(resp
            .results
            .into_iter()
            .map(|a| Agg {
                price: a.close,
                volume: a.volume as i32,
                timestamp: a.timestamp,
            })
            .collect())
    }

    /// Top market movers (gainers or losers) for US stocks. Returns them in the
    /// API's order (biggest move first). `GET /v2/snapshot/locale/us/markets/
    /// stocks/{direction}`.
    pub async fn get_market_movers(&self, direction: MoverDirection) -> Result<Vec<Mover>> {
        let url = format!(
            "{}/v2/snapshot/locale/us/markets/stocks/{}",
            self.rest_base,
            direction.path()
        );
        let resp: MoversResponse = self.get_json(&url).await?;
        Ok(resp
            .tickers
            .into_iter()
            .map(|t| {
                // Prefer the last trade price; fall back to today's close.
                let price = if t.last_trade.p != 0.0 { t.last_trade.p } else { t.day.c };
                Mover {
                    symbol: t.ticker,
                    price,
                    todays_change: t.todays_change,
                    todays_change_percentage: t.todays_change_perc,
                }
            })
            .collect())
    }
}

/// Company metadata returned by [`MarketClient::get_ticker_details`].
pub struct TickerDetails {
    pub company_name: String,
    pub outstanding_shares: i64,
}

/// Most recent completed weekday at/behind the given date (walks back over
/// weekends). Holidays are handled server-side by the aggregates endpoint.
pub fn current_or_previous_weekday(mut day: NaiveDate) -> NaiveDate {
    loop {
        match day.weekday() {
            Weekday::Sat | Weekday::Sun => {
                day = day.pred_opt().expect("date underflow");
            }
            _ => return day,
        }
    }
}

/// The current trading day (ET), walking back over weekends.
pub fn latest_trading_day() -> NaiveDate {
    let now_et = Utc::now().with_timezone(&New_York);
    current_or_previous_weekday(now_et.date_naive())
}

// ---- REST response shapes (short JSON tags as returned by the API) --------

#[derive(Deserialize)]
struct AggsResponse {
    #[serde(default)]
    results: Vec<AggResult>,
}

#[derive(Deserialize)]
struct AggResult {
    #[serde(default, rename = "c")]
    close: f64,
    #[serde(default, rename = "v")]
    volume: f64,
    #[serde(default, rename = "t")]
    timestamp: i64,
}

#[derive(Deserialize)]
struct LastTradeResponse {
    results: LastTradeResult,
}

#[derive(Deserialize)]
struct LastTradeResult {
    #[serde(default, rename = "p")]
    price: f64,
}

#[derive(Deserialize)]
struct MoversResponse {
    #[serde(default)]
    tickers: Vec<MoverResult>,
}

#[derive(Deserialize)]
struct MoverResult {
    #[serde(default)]
    ticker: String,
    #[serde(default, rename = "todaysChange")]
    todays_change: f64,
    #[serde(default, rename = "todaysChangePerc")]
    todays_change_perc: f64,
    #[serde(default, rename = "lastTrade")]
    last_trade: MoverTrade,
    #[serde(default)]
    day: MoverDay,
}

#[derive(Deserialize, Default)]
struct MoverTrade {
    #[serde(default)]
    p: f64,
}

#[derive(Deserialize, Default)]
struct MoverDay {
    #[serde(default)]
    c: f64,
}

#[derive(Deserialize)]
struct TickerDetailsResponse {
    results: TickerDetailsResult,
}

#[derive(Deserialize)]
struct TickerDetailsResult {
    #[serde(default)]
    name: String,
    #[serde(default)]
    weighted_shares_outstanding: i64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn weekend_walks_back_to_friday() {
        // 2026-07-25 is a Saturday; 2026-07-26 a Sunday. Both -> Fri 2026-07-24.
        let sat = NaiveDate::from_ymd_opt(2026, 7, 25).unwrap();
        let sun = NaiveDate::from_ymd_opt(2026, 7, 26).unwrap();
        let fri = NaiveDate::from_ymd_opt(2026, 7, 24).unwrap();
        assert_eq!(current_or_previous_weekday(sat), fri);
        assert_eq!(current_or_previous_weekday(sun), fri);
        // A weekday is returned unchanged.
        assert_eq!(current_or_previous_weekday(fri), fri);
    }
}
