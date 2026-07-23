//! The leader's authoritative in-memory state and its configuration. Kept behind
//! a `Mutex`; critical sections are short and never hold the lock across `.await`.

use tickerwall_proto::{
    sort_and_tag_tickers, Mover, PresentationSettings, Screen, ScreenCluster, Ticker,
};

/// Configuration used to build a [`crate::Leader`].
pub struct Config {
    pub api_key: String,
    /// Initial ticker symbols to display.
    pub tickers: Vec<String>,
    /// Starting presentation settings (defaults come from the CLI).
    pub settings: PresentationSettings,
}

/// Mutable cluster state: settings, the ticker list, and the connected screens.
pub struct LeaderState {
    pub settings: PresentationSettings,
    pub tickers: Vec<Ticker>,
    /// Currently-connected screens, always sorted ascending by `index`.
    pub screens: Vec<Screen>,
    /// Latest top gainers/losers for the secondary tape.
    pub movers: Vec<Mover>,
}

impl LeaderState {
    /// Build a `ScreenCluster` snapshot from the current settings + screens.
    pub fn cluster(&self) -> ScreenCluster {
        ScreenCluster {
            settings: Some(self.settings.clone()),
            screens: self.screens.clone(),
        }
    }

    /// Add (or replace, by UUID) a screen and keep the list sorted by index.
    pub fn add_screen(&mut self, screen: Screen) {
        match self.screens.iter_mut().find(|s| s.uuid == screen.uuid) {
            Some(existing) => *existing = screen,
            None => self.screens.push(screen),
        }
        self.sort_screens();
    }

    /// Remove a screen by UUID. Returns whether one was removed.
    pub fn remove_screen(&mut self, uuid: &str) -> bool {
        let before = self.screens.len();
        self.screens.retain(|s| s.uuid != uuid);
        self.sort_screens();
        self.screens.len() != before
    }

    /// Update an existing screen's geometry. Returns false if the UUID is unknown.
    pub fn update_screen(&mut self, screen: Screen) -> bool {
        match self.screens.iter_mut().find(|s| s.uuid == screen.uuid) {
            Some(existing) => {
                *existing = screen;
                self.sort_screens();
                true
            }
            None => false,
        }
    }

    fn sort_screens(&mut self) {
        self.screens.sort_by_key(|s| s.index);
    }

    /// Insert or replace a ticker (by symbol), then re-sort and re-tag indices.
    pub fn upsert_ticker(&mut self, ticker: Ticker) {
        match self.tickers.iter_mut().find(|t| t.symbol == ticker.symbol) {
            Some(existing) => *existing = ticker,
            None => self.tickers.push(ticker),
        }
        sort_and_tag_tickers(&mut self.tickers);
    }

    /// Remove a ticker by symbol. Returns whether one was removed.
    pub fn remove_ticker(&mut self, symbol: &str) -> bool {
        let before = self.tickers.len();
        self.tickers.retain(|t| t.symbol != symbol);
        sort_and_tag_tickers(&mut self.tickers);
        self.tickers.len() != before
    }

    pub fn symbols(&self) -> Vec<String> {
        self.tickers.iter().map(|t| t.symbol.clone()).collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn ticker(symbol: &str) -> Ticker {
        Ticker {
            symbol: symbol.to_string(),
            ..Default::default()
        }
    }

    fn state() -> LeaderState {
        LeaderState {
            settings: PresentationSettings::default(),
            tickers: Vec::new(),
            screens: Vec::new(),
            movers: Vec::new(),
        }
    }

    #[test]
    fn screens_stay_sorted_by_index() {
        let mut st = state();
        st.add_screen(Screen { uuid: "b".into(), index: 20, width: 100, height: 10 });
        st.add_screen(Screen { uuid: "a".into(), index: 10, width: 100, height: 10 });
        st.add_screen(Screen { uuid: "c".into(), index: 30, width: 100, height: 10 });
        assert_eq!(
            st.screens.iter().map(|s| s.uuid.as_str()).collect::<Vec<_>>(),
            ["a", "b", "c"]
        );
        assert!(st.remove_screen("b"));
        assert_eq!(
            st.screens.iter().map(|s| s.uuid.as_str()).collect::<Vec<_>>(),
            ["a", "c"]
        );
        assert!(!st.remove_screen("missing"));
    }

    #[test]
    fn upsert_dedupes_and_retags() {
        let mut st = state();
        st.upsert_ticker(ticker("NVDA"));
        st.upsert_ticker(ticker("AAPL"));
        st.upsert_ticker(ticker("AAPL")); // dedupe by symbol
        assert_eq!(st.tickers.len(), 2);
        // sorted + tagged
        assert_eq!(st.tickers[0].symbol, "AAPL");
        assert_eq!(st.tickers[0].index, 0);
        assert_eq!(st.tickers[1].symbol, "NVDA");
        assert_eq!(st.tickers[1].index, 1);

        assert!(st.remove_ticker("AAPL"));
        assert_eq!(st.tickers.len(), 1);
        assert_eq!(st.tickers[0].index, 0);
    }
}
