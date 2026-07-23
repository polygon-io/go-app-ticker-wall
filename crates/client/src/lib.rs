//! GUI-side cluster client: connects to the leader over gRPC, joins the cluster,
//! and keeps a synchronized local copy of state behind a lock the render loop
//! reads each frame. Replaces the Go `client` package.

use std::sync::Arc;
use std::time::Duration;

use parking_lot::{Mutex, RwLock};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use tickerwall_proto::leader_client::LeaderClient;
use tickerwall_proto::{
    Announcement, Empty, JoinRequest, PresentationSettings, PriceUpdate, Screen, ScreenCluster,
    Ticker, Update, UpdateKind,
};

use tonic::transport::Channel;

/// How long to wait between (re)connection / retry attempts.
const RETRY_DELAY: Duration = Duration::from_secs(1);

/// Configuration for a [`ClusterClient`].
pub struct Config {
    /// Leader address, including scheme (e.g. `"http://localhost:6886"`).
    pub leader: String,
    pub screen_width: i32,
    pub screen_height: i32,
    pub screen_index: i32,
}

/// Current state of the gRPC connection to the leader.
#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum GrpcStatus {
    Connected,
    Reconnecting,
    Disconnected,
}

/// Everything the render loop needs for a single frame, captured in one lock
/// acquisition (see [`ClusterClient::render_snapshot`]).
pub struct RenderSnapshot {
    pub screen: Screen,
    pub cluster: Option<ScreenCluster>,
    pub tickers: Vec<Ticker>,
    pub status: GrpcStatus,
    /// Announcements received since the previous frame (drained from the queue).
    pub new_announcements: Vec<Announcement>,
}

/// Mutable state kept in sync with the leader. Guarded by a single lock so the
/// render loop can cheaply snapshot it each frame.
struct Inner {
    screen: Screen,
    tickers: Vec<Ticker>,
    cluster: Option<ScreenCluster>,
    status: GrpcStatus,
    announcements: Vec<Announcement>,
}

/// Keeps the local client state in sync with the leader.
pub struct ClusterClient {
    leader: String,
    inner: RwLock<Inner>,
    /// Sender used by [`ClusterClient::request_resize`] to hand screen updates
    /// off to the async resize task spawned in [`ClusterClient::run`].
    resize_tx: Mutex<Option<mpsc::UnboundedSender<Screen>>>,
}

impl ClusterClient {
    /// Create a client. Generates a random screen uuid and builds the local
    /// [`Screen`] from `config`. Does not connect; call [`ClusterClient::run`].
    pub fn new(config: Config) -> Arc<Self> {
        let screen = Screen {
            uuid: Uuid::new_v4().to_string(),
            width: config.screen_width,
            height: config.screen_height,
            index: config.screen_index,
        };

        Arc::new(Self {
            leader: config.leader,
            inner: RwLock::new(Inner {
                screen,
                tickers: Vec::new(),
                cluster: None,
                status: GrpcStatus::Disconnected,
                announcements: Vec::new(),
            }),
            resize_tx: Mutex::new(None),
        })
    }

    /// Connect (retry until success or cancel), join the cluster, process the
    /// update stream, and auto-reconnect. Returns when `cancel` fires.
    pub async fn run(self: Arc<Self>, cancel: CancellationToken) -> anyhow::Result<()> {
        // Wire up the resize channel + background pusher task.
        let (tx, rx) = mpsc::unbounded_channel::<Screen>();
        *self.resize_tx.lock() = Some(tx);
        let resize_task = tokio::spawn(resize_loop(self.leader.clone(), rx, cancel.clone()));

        self.set_status(GrpcStatus::Disconnected);

        let result = self.clone().stream_loop(&cancel).await;

        // Tear down the resize task: drop the sender so the channel closes, then
        // wait for it to observe cancellation and exit.
        self.resize_tx.lock().take();
        let _ = resize_task.await;

        result
    }

    /// Main connect -> seed -> join -> stream loop with automatic reconnect.
    async fn stream_loop(self: Arc<Self>, cancel: &CancellationToken) -> anyhow::Result<()> {
        loop {
            if cancel.is_cancelled() {
                return Ok(());
            }

            // Connect, retrying until success or cancellation.
            let mut client = tokio::select! {
                _ = cancel.cancelled() => return Ok(()),
                c = self.connect_retry() => c,
            };

            // A session runs until the stream ends/errors or we are cancelled.
            match self.session(&mut client, cancel).await {
                Ok(()) => return Ok(()), // only returns Ok when cancelled
                Err(err) => {
                    tracing::warn!(error = %err, "cluster session ended; reconnecting");
                    self.set_status(GrpcStatus::Reconnecting);
                    tokio::select! {
                        _ = cancel.cancelled() => return Ok(()),
                        _ = tokio::time::sleep(RETRY_DELAY) => {}
                    }
                }
            }
        }
    }

    /// Connect to the leader, retrying with a short delay until it succeeds.
    async fn connect_retry(&self) -> LeaderClient<Channel> {
        loop {
            match LeaderClient::connect(self.leader.clone()).await {
                Ok(client) => return client,
                Err(err) => {
                    tracing::warn!(error = %err, "could not connect to leader; retrying");
                    self.set_status(GrpcStatus::Reconnecting);
                    tokio::time::sleep(RETRY_DELAY).await;
                }
            }
        }
    }

    /// Seed state from a snapshot, join the cluster, and process updates until
    /// the stream ends/errors (returns `Err`) or `cancel` fires (returns `Ok`).
    async fn session(
        &self,
        client: &mut LeaderClient<Channel>,
        cancel: &CancellationToken,
    ) -> anyhow::Result<()> {
        // Seed tickers + cluster from the one-shot snapshot.
        let snapshot = client.get_snapshot(Empty {}).await?.into_inner();
        {
            let mut guard = self.inner.write();
            guard.cluster = snapshot.cluster;
            let mut tickers = snapshot.tickers;
            tickerwall_proto::sort_and_tag_tickers(&mut tickers);
            guard.tickers = tickers;
        }

        // Open the update stream.
        let join = JoinRequest {
            screen: Some(self.screen()),
        };
        let mut stream = client.join_cluster(join).await?.into_inner();

        self.set_status(GrpcStatus::Connected);

        loop {
            tokio::select! {
                _ = cancel.cancelled() => return Ok(()),
                message = stream.message() => match message {
                    Ok(Some(update)) => self.apply_update(update),
                    Ok(None) => anyhow::bail!("update stream closed by leader"),
                    Err(status) => anyhow::bail!("update stream error: {status}"),
                },
            }
        }
    }

    // ---- state mutation (unit-testable without a server) ------------------

    /// Apply a single streamed [`Update`] to local state.
    fn apply_update(&self, update: Update) {
        let Some(kind) = update.kind else {
            tracing::warn!("received empty update");
            return;
        };

        match kind {
            UpdateKind::Cluster(cluster) => {
                self.inner.write().cluster = Some(cluster);
            }
            UpdateKind::TickerUpserted(ticker) => self.upsert_ticker(ticker),
            UpdateKind::TickerRemoved(symbol) => self.remove_ticker(&symbol),
            UpdateKind::Price(price) => self.apply_price(price),
            UpdateKind::Announcement(announcement) => {
                self.inner.write().announcements.push(announcement);
            }
            UpdateKind::Settings(settings) => {
                let mut guard = self.inner.write();
                match guard.cluster.as_mut() {
                    Some(cluster) => cluster.settings = Some(settings),
                    None => {
                        guard.cluster = Some(ScreenCluster {
                            settings: Some(settings),
                            screens: Vec::new(),
                        })
                    }
                }
            }
        }
    }

    /// Insert or replace a ticker by symbol, re-tag indices, then recompute its
    /// derived price fields.
    fn upsert_ticker(&self, ticker: Ticker) {
        let symbol = ticker.symbol.clone();
        let price = ticker.price;

        {
            let mut guard = self.inner.write();
            if let Some(existing) = guard.tickers.iter_mut().find(|t| t.symbol == symbol) {
                *existing = ticker;
            } else {
                guard.tickers.push(ticker);
            }
            tickerwall_proto::sort_and_tag_tickers(&mut guard.tickers);
        }

        // Recompute market cap / change percentage from the ticker's own price.
        self.apply_price(PriceUpdate { symbol, price });
    }

    /// Remove a ticker by symbol and re-tag indices.
    fn remove_ticker(&self, symbol: &str) {
        let mut guard = self.inner.write();
        guard.tickers.retain(|t| t.symbol != symbol);
        tickerwall_proto::sort_and_tag_tickers(&mut guard.tickers);
    }

    /// Apply a price update: set price and recompute market cap + change
    /// percentage. Guards against a zero previous close.
    fn apply_price(&self, update: PriceUpdate) {
        let mut guard = self.inner.write();
        if let Some(ticker) = guard.tickers.iter_mut().find(|t| t.symbol == update.symbol) {
            ticker.price = update.price;
            ticker.market_cap = ticker.outstanding_shares as f64 * update.price;
            if ticker.previous_close_price != 0.0 {
                ticker.price_change_percentage =
                    ((ticker.price / ticker.previous_close_price) - 1.0) * 100.0;
            }
        }
    }

    fn set_status(&self, status: GrpcStatus) {
        self.inner.write().status = status;
    }

    // ---- read accessors ---------------------------------------------------

    /// Capture all render-loop state under a single lock, draining any queued
    /// announcements. The render loop calls this once per frame instead of the
    /// separate `screen`/`settings`/`cluster`/`tickers`/`status`/
    /// `drain_announcements` accessors (~7 lock acquisitions → 1), which reduces
    /// contention with the price-update writer under a busy feed.
    pub fn render_snapshot(&self) -> RenderSnapshot {
        let mut inner = self.inner.write();
        RenderSnapshot {
            screen: inner.screen.clone(),
            cluster: inner.cluster.clone(),
            tickers: inner.tickers.clone(),
            status: inner.status,
            new_announcements: std::mem::take(&mut inner.announcements),
        }
    }

    pub fn tickers(&self) -> Vec<Ticker> {
        self.inner.read().tickers.clone()
    }

    pub fn settings(&self) -> Option<PresentationSettings> {
        self.inner
            .read()
            .cluster
            .as_ref()
            .and_then(|c| c.settings)
    }

    pub fn cluster(&self) -> Option<ScreenCluster> {
        self.inner.read().cluster.clone()
    }

    pub fn screen(&self) -> Screen {
        self.inner.read().screen.clone()
    }

    pub fn status(&self) -> GrpcStatus {
        self.inner.read().status
    }

    /// Drain any announcements received since the last call.
    pub fn drain_announcements(&self) -> Vec<Announcement> {
        std::mem::take(&mut self.inner.write().announcements)
    }

    /// Update the local screen size and asynchronously push it to the leader.
    /// Non-blocking: safe to call from the GUI main thread.
    pub fn request_resize(&self, width: i32, height: i32) {
        let screen = {
            let mut guard = self.inner.write();
            guard.screen.width = width;
            guard.screen.height = height;
            guard.screen.clone()
        };

        if let Some(tx) = self.resize_tx.lock().as_ref() {
            // Unbounded send never blocks; only fails if the task has stopped.
            let _ = tx.send(screen);
        }
    }
}

/// Background task that pushes screen-resize updates to the leader. Owns its own
/// connection so a resize during a reconnect still eventually lands.
async fn resize_loop(
    leader: String,
    mut rx: mpsc::UnboundedReceiver<Screen>,
    cancel: CancellationToken,
) {
    let mut client: Option<LeaderClient<Channel>> = None;

    loop {
        let screen = tokio::select! {
            _ = cancel.cancelled() => break,
            msg = rx.recv() => match msg {
                Some(screen) => screen,
                None => break, // sender dropped
            },
        };

        // Lazily (re)connect.
        if client.is_none() {
            match LeaderClient::connect(leader.clone()).await {
                Ok(connected) => client = Some(connected),
                Err(err) => {
                    tracing::warn!(error = %err, "resize: could not connect to leader");
                    continue;
                }
            }
        }

        if let Some(inner) = client.as_mut() {
            if let Err(status) = inner.update_screen(screen).await {
                tracing::warn!(error = %status, "resize: update_screen failed");
                // Force a reconnect on the next resize.
                client = None;
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_client() -> Arc<ClusterClient> {
        ClusterClient::new(Config {
            leader: "http://localhost:6886".to_string(),
            screen_width: 1920,
            screen_height: 1080,
            screen_index: 0,
        })
    }

    fn ticker(symbol: &str, price: f64) -> Ticker {
        Ticker {
            symbol: symbol.to_string(),
            price,
            ..Default::default()
        }
    }

    fn upsert(t: Ticker) -> Update {
        Update {
            kind: Some(UpdateKind::TickerUpserted(t)),
        }
    }

    #[test]
    fn upsert_inserts_dedupes_and_tags() {
        let client = test_client();

        client.apply_update(upsert(ticker("NVDA", 100.0)));
        client.apply_update(upsert(ticker("AAPL", 200.0)));
        client.apply_update(upsert(ticker("MSFT", 300.0)));

        let tickers = client.tickers();
        assert_eq!(tickers.len(), 3);
        // Sorted ascending by symbol with re-tagged indices.
        assert_eq!(tickers[0].symbol, "AAPL");
        assert_eq!(tickers[0].index, 0);
        assert_eq!(tickers[1].symbol, "MSFT");
        assert_eq!(tickers[1].index, 1);
        assert_eq!(tickers[2].symbol, "NVDA");
        assert_eq!(tickers[2].index, 2);

        // Upserting an existing symbol replaces rather than duplicates.
        client.apply_update(upsert(ticker("AAPL", 250.0)));
        let tickers = client.tickers();
        assert_eq!(tickers.len(), 3);
        let aapl = tickers.iter().find(|t| t.symbol == "AAPL").unwrap();
        assert_eq!(aapl.price, 250.0);
    }

    #[test]
    fn price_update_recomputes_derived_fields() {
        let client = test_client();

        client.apply_update(upsert(Ticker {
            symbol: "AAPL".into(),
            outstanding_shares: 10,
            previous_close_price: 100.0,
            ..Default::default()
        }));

        client.apply_price(PriceUpdate {
            symbol: "AAPL".into(),
            price: 110.0,
        });

        let tickers = client.tickers();
        let aapl = &tickers[0];
        assert_eq!(aapl.price, 110.0);
        assert_eq!(aapl.market_cap, 10.0 * 110.0);
        // ((110/100) - 1) * 100 == 10%
        assert!((aapl.price_change_percentage - 10.0).abs() < 1e-9);
    }

    #[test]
    fn price_update_does_not_panic_on_zero_previous_close() {
        let client = test_client();

        client.apply_update(upsert(Ticker {
            symbol: "AAPL".into(),
            outstanding_shares: 5,
            previous_close_price: 0.0,
            ..Default::default()
        }));

        client.apply_price(PriceUpdate {
            symbol: "AAPL".into(),
            price: 42.0,
        });

        let tickers = client.tickers();
        let aapl = &tickers[0];
        assert_eq!(aapl.price, 42.0);
        assert_eq!(aapl.market_cap, 5.0 * 42.0);
        // Percentage left at its default when previous close is zero.
        assert_eq!(aapl.price_change_percentage, 0.0);
    }

    #[test]
    fn removal_removes_and_retags() {
        let client = test_client();
        client.apply_update(upsert(ticker("AAPL", 1.0)));
        client.apply_update(upsert(ticker("MSFT", 2.0)));
        client.apply_update(upsert(ticker("NVDA", 3.0)));

        client.apply_update(Update {
            kind: Some(UpdateKind::TickerRemoved("MSFT".to_string())),
        });

        let tickers = client.tickers();
        assert_eq!(tickers.len(), 2);
        assert_eq!(tickers[0].symbol, "AAPL");
        assert_eq!(tickers[0].index, 0);
        assert_eq!(tickers[1].symbol, "NVDA");
        assert_eq!(tickers[1].index, 1);
    }

    #[test]
    fn announcements_queue_and_drain() {
        let client = test_client();
        assert!(client.drain_announcements().is_empty());

        client.apply_update(Update {
            kind: Some(UpdateKind::Announcement(Announcement {
                message: "hello".into(),
                ..Default::default()
            })),
        });
        client.apply_update(Update {
            kind: Some(UpdateKind::Announcement(Announcement {
                message: "world".into(),
                ..Default::default()
            })),
        });

        let drained = client.drain_announcements();
        assert_eq!(drained.len(), 2);
        assert_eq!(drained[0].message, "hello");
        assert_eq!(drained[1].message, "world");
        // Draining empties the queue.
        assert!(client.drain_announcements().is_empty());
    }

    #[test]
    fn settings_update_replaces_cluster_settings() {
        let client = test_client();

        // Seed a cluster first.
        client.apply_update(Update {
            kind: Some(UpdateKind::Cluster(ScreenCluster {
                settings: Some(PresentationSettings {
                    scroll_speed: 1,
                    ..Default::default()
                }),
                screens: Vec::new(),
            })),
        });

        client.apply_update(Update {
            kind: Some(UpdateKind::Settings(PresentationSettings {
                scroll_speed: 9,
                ticker_box_width: 1234,
                ..Default::default()
            })),
        });

        let settings = client.settings().expect("settings present");
        assert_eq!(settings.scroll_speed, 9);
        assert_eq!(settings.ticker_box_width, 1234);
        // Cluster still present, screens untouched.
        assert_eq!(client.cluster().unwrap().screens.len(), 0);
    }

    #[test]
    fn request_resize_updates_local_screen() {
        let client = test_client();
        // No resize task running (run() not called): must not panic.
        client.request_resize(800, 600);
        let screen = client.screen();
        assert_eq!(screen.width, 800);
        assert_eq!(screen.height, 600);
    }
}
