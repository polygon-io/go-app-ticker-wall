//! The authoritative cluster state, data-refresh loops, broadcast bus, and the
//! tonic `Leader` service. Replaces the Go `leader` + `server` packages.

mod service;
mod state;

pub use state::{Config, LeaderState};

use std::net::SocketAddr;
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use parking_lot::Mutex;
use tickerwall_data::{latest_trading_day, MarketClient, MoverDirection};
use tickerwall_proto::{MarketMovers, PriceUpdate, Ticker, Update, UpdateKind as Kind};
use tokio::sync::{broadcast, mpsc, Notify};
use tokio_util::sync::CancellationToken;
use tracing::{error, info, warn};

/// Broadcast bus capacity. Slow clients that fall this far behind get a `Lagged`
/// error and skip ahead rather than blocking the bus.
const BROADCAST_CAPACITY: usize = 1024;
/// Aggregate bar size, in minutes. The Go original used 10, giving only ~45
/// points across a full session — visibly jagged on the full-bleed graph. At 2
/// minutes we get ~225 points: smooth without being noisy (1 minute was too busy).
pub(crate) const AGG_RANGE_MINUTES: i32 = 2;
/// How often intraday aggregates are refreshed.
pub const TICKER_AGGS_REFRESH_INTERVAL: Duration = Duration::from_secs(60);
/// How often company details / prices are refreshed (a true 5 minutes; the Go
/// original said "5min" in a comment but actually used 500s).
pub const TICKER_DETAILS_REFRESH_INTERVAL: Duration = Duration::from_secs(300);
/// How often the top gainers/losers tape is refreshed (they shift through the day).
pub const MOVERS_REFRESH_INTERVAL: Duration = Duration::from_secs(60);
/// How many gainers and how many losers to show on the secondary tape (each).
pub(crate) const MOVERS_PER_DIRECTION: usize = 10;
/// Lead time added to an announcement's display timestamp so every screen shows
/// it in sync.
pub(crate) const ANNOUNCEMENT_LEAD_MS: i64 = 200;
/// Backoff before reconnecting the price feed after an error/close.
const PRICE_FEED_RETRY: Duration = Duration::from_secs(2);

/// The ticker wall leader.
pub struct Leader {
    state: Arc<Mutex<LeaderState>>,
    tx: broadcast::Sender<Update>,
    data: MarketClient,
    /// Notified when the ticker set changes so the price feed re-subscribes.
    resubscribe: Arc<Notify>,
}

impl Leader {
    pub fn new(config: Config) -> Self {
        let data = MarketClient::new(config.api_key, config.settings.per_tick_updates);
        let tickers = config
            .tickers
            .into_iter()
            .map(|symbol| Ticker {
                symbol,
                ..Default::default()
            })
            .collect();
        let state = LeaderState {
            settings: config.settings,
            tickers,
            screens: Vec::new(),
            movers: Vec::new(),
        };
        let (tx, _) = broadcast::channel(BROADCAST_CAPACITY);
        Self {
            state: Arc::new(Mutex::new(state)),
            tx,
            data,
            resubscribe: Arc::new(Notify::new()),
        }
    }

    /// Send an update to every connected client. An error only means no clients
    /// are currently subscribed, which is fine.
    fn broadcast(&self, update: Update) {
        let _ = self.tx.send(update);
    }

    fn symbols(&self) -> Vec<String> {
        self.state.lock().symbols()
    }

    /// Synchronously load details + aggregates before accepting clients so the
    /// first screen to join sees populated data.
    pub async fn load_initial_data(&self) -> anyhow::Result<()> {
        info!("loading initial ticker data…");
        self.refresh_details(true).await?;
        self.refresh_aggs().await?;
        // Movers are best-effort: a failure here shouldn't block startup.
        if let Err(e) = self.refresh_movers().await {
            warn!(error = %e, "initial movers load failed");
        }
        info!("initial ticker data loaded");
        Ok(())
    }

    /// Fetch top gainers + losers and broadcast them as the secondary tape.
    async fn refresh_movers(&self) -> anyhow::Result<()> {
        let gainers = self.data.get_market_movers(MoverDirection::Gainers).await?;
        let losers = self.data.get_market_movers(MoverDirection::Losers).await?;

        let mut movers = Vec::with_capacity(MOVERS_PER_DIRECTION * 2);
        movers.extend(gainers.into_iter().take(MOVERS_PER_DIRECTION));
        movers.extend(losers.into_iter().take(MOVERS_PER_DIRECTION));

        self.state.lock().movers = movers.clone();
        self.broadcast(Update {
            kind: Some(Kind::Movers(MarketMovers { movers })),
        });
        Ok(())
    }

    async fn refresh_details(&self, first_run: bool) -> anyhow::Result<()> {
        for symbol in self.symbols() {
            let details = self.data.load_ticker_data(&symbol).await?;
            let updated = {
                let mut st = self.state.lock();
                let Some(t) = st.tickers.iter_mut().find(|t| t.symbol == symbol) else {
                    continue;
                };
                t.company_name = details.company_name;
                t.previous_close_price = details.previous_close_price;
                t.outstanding_shares = details.outstanding_shares;
                if first_run {
                    t.price = details.price;
                }
                t.clone()
            };
            self.broadcast(upsert(updated));
        }
        Ok(())
    }

    async fn refresh_aggs(&self) -> anyhow::Result<()> {
        let day = latest_trading_day();
        for symbol in self.symbols() {
            let aggs = self.data.get_today_aggs(day, &symbol, AGG_RANGE_MINUTES).await?;
            let updated = {
                let mut st = self.state.lock();
                let Some(t) = st.tickers.iter_mut().find(|t| t.symbol == symbol) else {
                    continue;
                };
                // Diff by content, not slice length (the Go version only compared
                // len, so same-count value changes were dropped).
                if t.aggs == aggs {
                    continue;
                }
                t.aggs = aggs;
                t.clone()
            };
            self.broadcast(upsert(updated));
        }
        Ok(())
    }

    /// Spawn the background loops (price feed, aggregate + detail refresh).
    pub fn spawn_background(self: &Arc<Self>, cancel: CancellationToken) {
        {
            let this = self.clone();
            let cancel = cancel.clone();
            tokio::spawn(async move { this.price_feed_loop(cancel).await });
        }
        {
            let this = self.clone();
            let cancel = cancel.clone();
            tokio::spawn(async move {
                this.refresh_loop(cancel, TICKER_AGGS_REFRESH_INTERVAL, false).await
            });
        }
        {
            let this = self.clone();
            let cancel = cancel.clone();
            tokio::spawn(async move {
                this.refresh_loop(cancel, TICKER_DETAILS_REFRESH_INTERVAL, true).await
            });
        }
        {
            let this = self.clone();
            tokio::spawn(async move { this.movers_loop(cancel).await });
        }
    }

    /// Periodically refresh the gainers/losers tape. Errors are logged, not fatal.
    async fn movers_loop(self: Arc<Self>, cancel: CancellationToken) {
        let mut interval = tokio::time::interval(MOVERS_REFRESH_INTERVAL);
        interval.tick().await; // consume the immediate first tick (loaded at startup)
        loop {
            tokio::select! {
                _ = cancel.cancelled() => break,
                _ = interval.tick() => {
                    if let Err(e) = self.refresh_movers().await {
                        error!(error = %e, "periodic movers refresh failed");
                    }
                }
            }
        }
    }

    /// Periodic refresh loop. `details` selects details-vs-aggregates. A failed
    /// refresh is logged but never terminates the loop (one bad API call
    /// shouldn't take the leader down).
    async fn refresh_loop(self: Arc<Self>, cancel: CancellationToken, period: Duration, details: bool) {
        let mut interval = tokio::time::interval(period);
        interval.tick().await; // consume the immediate first tick (already loaded at startup)
        loop {
            tokio::select! {
                _ = cancel.cancelled() => break,
                _ = interval.tick() => {
                    let result = if details {
                        self.refresh_details(false).await
                    } else {
                        self.refresh_aggs().await
                    };
                    if let Err(e) = result {
                        error!(error = %e, details, "periodic refresh failed");
                    }
                }
            }
        }
    }

    /// Consume the live price feed and forward each tick onto the broadcast bus.
    /// Reconnects on error and re-subscribes when the ticker set changes.
    async fn price_feed_loop(self: Arc<Self>, cancel: CancellationToken) {
        let (tx, mut rx) = mpsc::channel::<PriceUpdate>(10_000);

        // Forwarder: price updates -> broadcast bus.
        {
            let this = self.clone();
            let cancel = cancel.clone();
            tokio::spawn(async move {
                loop {
                    tokio::select! {
                        _ = cancel.cancelled() => break,
                        maybe = rx.recv() => match maybe {
                            Some(update) => this.broadcast(Update { kind: Some(Kind::Price(update)) }),
                            None => break,
                        },
                    }
                }
            });
        }

        // Producer: (re)connect the websocket for the current symbol set.
        while !cancel.is_cancelled() {
            let symbols = self.symbols();
            if symbols.is_empty() {
                tokio::select! {
                    _ = cancel.cancelled() => break,
                    _ = self.resubscribe.notified() => continue,
                }
            }

            let child = cancel.child_token();
            let mut listener = {
                let data = self.data.clone();
                let tx = tx.clone();
                let child = child.clone();
                tokio::spawn(async move { data.listen_for_price_updates(&symbols, tx, child).await })
            };

            tokio::select! {
                _ = cancel.cancelled() => {
                    child.cancel();
                    let _ = (&mut listener).await;
                    break;
                }
                _ = self.resubscribe.notified() => {
                    // Ticker set changed; drop this connection and reconnect.
                    child.cancel();
                    let _ = (&mut listener).await;
                }
                res = &mut listener => {
                    if let Ok(Err(e)) = res {
                        warn!(error = %e, "price feed error; reconnecting");
                    }
                    if cancel.is_cancelled() {
                        break;
                    }
                    tokio::time::sleep(PRICE_FEED_RETRY).await;
                }
            }
        }
    }
}

/// Serve the gRPC `Leader` service until `cancel` fires.
pub async fn serve_grpc(
    leader: Arc<Leader>,
    addr: SocketAddr,
    cancel: CancellationToken,
) -> anyhow::Result<()> {
    use tickerwall_proto::leader_server::LeaderServer;

    info!(%addr, "gRPC leader listening");
    tonic::transport::Server::builder()
        .add_service(LeaderServer::from_arc(leader))
        .serve_with_shutdown(addr, async move { cancel.cancelled().await })
        .await?;
    Ok(())
}

/// Convenience: wrap a ticker in an "upserted" update.
fn upsert(ticker: Ticker) -> Update {
    Update {
        kind: Some(Kind::TickerUpserted(ticker)),
    }
}

pub(crate) fn now_ms() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as i64
}

// Re-export for the app binary to build its config.
pub use tickerwall_proto::PresentationSettings as Settings;
