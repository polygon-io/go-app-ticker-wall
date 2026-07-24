//! tonic `Leader` service implementation. Each mutation updates state under the
//! lock, then broadcasts the resulting change to all streaming clients.

use std::pin::Pin;
use std::sync::Arc;

use futures::Stream;
use parking_lot::Mutex;
use tokio::sync::broadcast;
use tonic::{Request, Response, Status};
use tracing::{info, warn};

use tickerwall_proto::{
    leader_server, AnnounceRequest, Announcement, Empty, JoinRequest, MarketMovers,
    PresentationSettings, Screen, SettingsPatch, Snapshot, TickerSymbol, Update,
    UpdateKind as Kind,
};

use crate::{now_ms, Leader, LeaderState, AGG_RANGE_MINUTES, ANNOUNCEMENT_LEAD_MS};

/// Removes a screen from the cluster (and announces the change) when a client's
/// stream is dropped — i.e. when the GUI disconnects.
struct ScreenGuard {
    state: Arc<Mutex<LeaderState>>,
    tx: broadcast::Sender<Update>,
    uuid: String,
}

impl Drop for ScreenGuard {
    fn drop(&mut self) {
        let cluster = {
            let mut st = self.state.lock();
            st.remove_screen(&self.uuid);
            st.cluster()
        };
        let _ = self.tx.send(Update {
            kind: Some(Kind::Cluster(cluster)),
        });
        info!(uuid = %self.uuid, "screen left cluster");
    }
}

#[tonic::async_trait]
impl leader_server::Leader for Leader {
    type JoinClusterStream = Pin<Box<dyn Stream<Item = Result<Update, Status>> + Send>>;

    async fn join_cluster(
        &self,
        request: Request<JoinRequest>,
    ) -> Result<Response<Self::JoinClusterStream>, Status> {
        let screen = request
            .into_inner()
            .screen
            .ok_or_else(|| Status::invalid_argument("join request missing screen"))?;
        let uuid = screen.uuid.clone();
        info!(uuid = %uuid, index = screen.index, width = screen.width, height = screen.height, "screen joined cluster");

        // Subscribe BEFORE broadcasting so this client sees the join update too.
        let rx = self.tx.subscribe();
        let cluster = {
            let mut st = self.state.lock();
            st.add_screen(screen);
            st.cluster()
        };
        self.broadcast(Update {
            kind: Some(Kind::Cluster(cluster)),
        });

        let guard = ScreenGuard {
            state: self.state.clone(),
            tx: self.tx.clone(),
            uuid,
        };

        // The guard lives inside the stream generator; when the client
        // disconnects, tonic drops the stream, dropping the guard and removing
        // the screen.
        let stream = async_stream::stream! {
            let _guard = guard;
            let mut rx = rx;
            loop {
                match rx.recv().await {
                    Ok(update) => yield Ok(update),
                    Err(broadcast::error::RecvError::Lagged(n)) => {
                        warn!(skipped = n, "client lagged; skipping ahead");
                    }
                    Err(broadcast::error::RecvError::Closed) => break,
                }
            }
        };

        Ok(Response::new(Box::pin(stream)))
    }

    async fn get_snapshot(&self, _request: Request<Empty>) -> Result<Response<Snapshot>, Status> {
        let st = self.state.lock();
        Ok(Response::new(Snapshot {
            cluster: Some(st.cluster()),
            tickers: st.tickers.clone(),
            movers: Some(MarketMovers {
                movers: st.movers.clone(),
            }),
        }))
    }

    async fn update_settings(
        &self,
        request: Request<SettingsPatch>,
    ) -> Result<Response<PresentationSettings>, Status> {
        let patch = request.into_inner();
        let settings = {
            let mut st = self.state.lock();
            st.settings.apply_patch(&patch);
            st.settings
        };
        self.broadcast(Update {
            kind: Some(Kind::Settings(settings)),
        });
        Ok(Response::new(settings))
    }

    async fn announce(
        &self,
        request: Request<AnnounceRequest>,
    ) -> Result<Response<Announcement>, Status> {
        let req = request.into_inner();
        let announcement = Announcement {
            message: req.message,
            r#type: req.r#type,
            show_at_timestamp_ms: now_ms() + ANNOUNCEMENT_LEAD_MS,
            lifespan_ms: req.lifespan_ms,
            animation: req.animation,
        };
        self.broadcast(Update {
            kind: Some(Kind::Announcement(announcement.clone())),
        });
        Ok(Response::new(announcement))
    }

    async fn update_screen(&self, request: Request<Screen>) -> Result<Response<Screen>, Status> {
        let screen = request.into_inner();
        let cluster = {
            let mut st = self.state.lock();
            if !st.update_screen(screen.clone()) {
                return Err(Status::not_found("unknown screen uuid"));
            }
            st.cluster()
        };
        self.broadcast(Update {
            kind: Some(Kind::Cluster(cluster)),
        });
        Ok(Response::new(screen))
    }

    async fn add_ticker(&self, request: Request<TickerSymbol>) -> Result<Response<Empty>, Status> {
        let symbol = request.into_inner().symbol.to_uppercase();
        if symbol.is_empty() {
            return Err(Status::invalid_argument("empty ticker symbol"));
        }
        // Already present? Nothing to do.
        if self.state.lock().tickers.iter().any(|t| t.symbol == symbol) {
            return Ok(Response::new(Empty {}));
        }

        // Fetch data before taking the lock.
        let mut ticker = self
            .data
            .load_ticker_data(&symbol)
            .await
            .map_err(|e| Status::internal(format!("load ticker {symbol}: {e}")))?;
        if let Ok(aggs) = self
            .data
            .get_today_aggs(
                tickerwall_data::latest_trading_day(),
                &symbol,
                AGG_RANGE_MINUTES,
            )
            .await
        {
            ticker.aggs = aggs;
        }

        let stored = {
            let mut st = self.state.lock();
            st.upsert_ticker(ticker);
            st.tickers.iter().find(|t| t.symbol == symbol).cloned()
        };
        // Re-subscribe the price feed to include the new symbol.
        self.resubscribe.notify_one();

        if let Some(stored) = stored {
            self.broadcast(Update {
                kind: Some(Kind::TickerUpserted(stored)),
            });
        }
        info!(symbol = %symbol, "ticker added");
        Ok(Response::new(Empty {}))
    }

    async fn remove_ticker(
        &self,
        request: Request<TickerSymbol>,
    ) -> Result<Response<Empty>, Status> {
        let symbol = request.into_inner().symbol.to_uppercase();
        let removed = self.state.lock().remove_ticker(&symbol);
        if !removed {
            return Err(Status::not_found("unknown ticker symbol"));
        }
        self.resubscribe.notify_one();
        self.broadcast(Update {
            kind: Some(Kind::TickerRemoved(symbol.clone())),
        });
        info!(symbol = %symbol, "ticker removed");
        Ok(Response::new(Empty {}))
    }
}
