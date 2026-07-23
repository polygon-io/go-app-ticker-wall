//! Generated protobuf/gRPC types for the ticker wall, plus small domain helpers
//! that live close to the wire types (cluster geometry, settings merge, ticker
//! ordering). These mirror the hand-written helpers from the Go `models` package.

/// Generated code for the `tickerwall.v1` package.
pub mod pb {
    tonic::include_proto!("tickerwall.v1");
}

// Re-export the common types at the crate root for ergonomics.
pub use pb::{
    leader_client, leader_server, update::Kind as UpdateKind, Agg, AnnounceRequest, Announcement,
    AnnouncementAnimation, AnnouncementType, Empty, JoinRequest, PresentationSettings, PriceUpdate,
    Rgba, Screen, ScreenCluster, SettingsPatch, Snapshot, Ticker, TickerSymbol, Update,
};

impl ScreenCluster {
    /// Sum of the widths of every screen ordered before `uuid`. This is where a
    /// given screen's left edge sits within the full tape. Screens are expected
    /// to be sorted by `index` (the leader guarantees this).
    pub fn screen_global_offset(&self, uuid: &str) -> f32 {
        let mut offset = 0.0;
        for screen in &self.screens {
            if screen.uuid == uuid {
                break;
            }
            offset += screen.width as f32;
        }
        offset
    }

    /// Total pixel width of the whole cluster (all screens combined).
    pub fn global_viewport_size(&self) -> i32 {
        self.screens.iter().map(|s| s.width).sum()
    }

    pub fn number_of_screens(&self) -> usize {
        self.screens.len()
    }
}

impl PresentationSettings {
    /// Merge a partial patch into these settings. Only fields present on the
    /// patch are applied; unset fields are left untouched. This is the single
    /// settings-merge path (the Go original had two divergent ones).
    pub fn apply_patch(&mut self, patch: &SettingsPatch) {
        if let Some(v) = patch.ticker_box_width {
            self.ticker_box_width = v;
        }
        if let Some(v) = patch.scroll_speed {
            self.scroll_speed = v;
        }
        if patch.up_color.is_some() {
            self.up_color = patch.up_color;
        }
        if patch.down_color.is_some() {
            self.down_color = patch.down_color;
        }
        if patch.bg_color.is_some() {
            self.bg_color = patch.bg_color;
        }
        if patch.font_color.is_some() {
            self.font_color = patch.font_color;
        }
        if patch.ticker_box_bg_color.is_some() {
            self.ticker_box_bg_color = patch.ticker_box_bg_color;
        }
        if let Some(v) = patch.show_logos {
            self.show_logos = v;
        }
        if let Some(v) = patch.show_fps {
            self.show_fps = v;
        }
        if let Some(v) = patch.animation_duration_ms {
            self.animation_duration_ms = v;
        }
        if let Some(v) = patch.per_tick_updates {
            self.per_tick_updates = v;
        }
    }
}

/// Sort tickers by symbol ascending and (re)assign each ticker's `index` to its
/// position. Every screen runs this identically, which is what keeps the tape
/// ordering consistent across the cluster.
pub fn sort_and_tag_tickers(tickers: &mut [Ticker]) {
    tickers.sort_by(|a, b| a.symbol.cmp(&b.symbol));
    for (i, ticker) in tickers.iter_mut().enumerate() {
        ticker.index = i as i32;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn screen(uuid: &str, index: i32, width: i32) -> Screen {
        Screen {
            uuid: uuid.to_string(),
            width,
            height: 300,
            index,
            ..Default::default()
        }
    }

    #[test]
    fn screen_global_offset_sums_preceding_widths() {
        let cluster = ScreenCluster {
            settings: None,
            screens: vec![screen("a", 10, 1920), screen("b", 20, 1080), screen("c", 30, 800)],
        };
        assert_eq!(cluster.screen_global_offset("a"), 0.0);
        assert_eq!(cluster.screen_global_offset("b"), 1920.0);
        assert_eq!(cluster.screen_global_offset("c"), 3000.0);
        assert_eq!(cluster.global_viewport_size(), 3800);
    }

    #[test]
    fn apply_patch_only_touches_set_fields() {
        let mut settings = PresentationSettings {
            ticker_box_width: 1100,
            scroll_speed: 8,
            animation_duration_ms: 500,
            per_tick_updates: true,
            ..Default::default()
        };
        let patch = SettingsPatch {
            scroll_speed: Some(5),
            ..Default::default()
        };
        settings.apply_patch(&patch);
        // Changed:
        assert_eq!(settings.scroll_speed, 5);
        // Untouched (the Go CLI bug would have reset these to defaults):
        assert_eq!(settings.ticker_box_width, 1100);
        assert_eq!(settings.animation_duration_ms, 500);
        assert!(settings.per_tick_updates);
    }

    #[test]
    fn sort_and_tag_orders_by_symbol() {
        let mut tickers = vec![
            Ticker { symbol: "NVDA".into(), ..Default::default() },
            Ticker { symbol: "AAPL".into(), ..Default::default() },
            Ticker { symbol: "MSFT".into(), ..Default::default() },
        ];
        sort_and_tag_tickers(&mut tickers);
        assert_eq!(tickers[0].symbol, "AAPL");
        assert_eq!(tickers[0].index, 0);
        assert_eq!(tickers[1].symbol, "MSFT");
        assert_eq!(tickers[1].index, 1);
        assert_eq!(tickers[2].symbol, "NVDA");
        assert_eq!(tickers[2].index, 2);
    }
}
