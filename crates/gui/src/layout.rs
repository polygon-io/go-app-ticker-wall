//! Scroll / layout math — the load-bearing logic that makes N screens render one
//! continuous tape. Pure functions (no rendering) so they can be unit-tested.
//!
//! Ported from the Go `generateGlobalOffset` (gui.go) and
//! `DetermineTickersForRender` / `TickerOffset` (layout.go), with the acknowledged
//! coverage bug fixed: when a screen is wider than the whole tape, we keep wrapping
//! and drawing repeated copies to fill it instead of emitting `nil` tickers.

use std::time::{SystemTime, UNIX_EPOCH};

/// Wall-clock nanoseconds. Wall clock (not a per-process instant) is what keeps
/// every screen — even on different machines — scrolling in agreement.
pub fn now_nanos() -> u128 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos()
}

/// The global scroll offset (in pixels) at `now_nanos`, reduced into `[0, tape)`
/// where `tape = num_tickers * ticker_box_width`. Higher `scroll_speed` scrolls
/// slower (it is inverted, matching the Go behavior: 1 is fastest).
pub fn global_offset(now_nanos: u128, scroll_speed: i32, ticker_box_width: i32, num_tickers: usize) -> f32 {
    let tape = num_tickers as f64 * ticker_box_width as f64;
    if tape <= 0.0 {
        return 0.0;
    }
    // time.Millisecond == 1_000_000 ns; divisor = scroll_speed * 1ms.
    let speed = scroll_speed.max(1) as f64;
    let raw = now_nanos as f64 / (speed * 1_000_000.0);
    let reduced = raw - (raw / tape).floor() * tape;
    reduced as f32
}

/// A ticker that should be drawn this frame: its index into the sorted ticker
/// slice, and the x pixel offset of its box's left edge on this screen.
#[derive(Debug, Clone, Copy, PartialEq)]
pub struct VisibleTicker {
    pub index: usize,
    pub x: f32,
}

/// Determine which tickers are visible on a screen and where to draw them.
///
/// * `global_offset` — from [`global_offset`].
/// * `screen_offset` — this screen's left edge in tape coordinates
///   (`ScreenCluster::screen_global_offset`).
/// * `screen_width` — this screen's width in pixels.
///
/// Returns entries left-to-right. If the screen is wider than the tape, tickers
/// wrap and repeat (each repeat gets its own x), so the screen is always filled.
pub fn visible_tickers(
    global_offset: f32,
    screen_offset: f32,
    screen_width: f32,
    num_tickers: usize,
    ticker_box_width: i32,
) -> Vec<VisibleTicker> {
    let mut out = Vec::new();
    if num_tickers == 0 || ticker_box_width <= 0 {
        return out;
    }
    let box_w = ticker_box_width as f32;
    let n = num_tickers as i64;
    let tape = n as f32 * box_w;

    // This screen's window into the tape, reduced into [0, tape).
    let mut localized = (global_offset + screen_offset) % tape;
    if localized < 0.0 {
        localized += tape;
    }

    // Leftmost (possibly partially visible) box index.
    let first = (localized / box_w).floor() as i64;

    // Safety cap: at most every box once, plus a full extra wrap for wide screens.
    let max_boxes = (num_tickers * 2 + 4) as usize;
    let mut i = first;
    loop {
        let x = (i as f32 * box_w) - localized;
        if x >= screen_width {
            break;
        }
        let index = ((i % n) + n) % n; // wrap into [0, n)
        out.push(VisibleTicker {
            index: index as usize,
            x,
        });
        i += 1;
        if out.len() >= max_boxes {
            break;
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn global_offset_is_bounded_and_zero_safe() {
        // No tickers -> no division, offset 0.
        assert_eq!(global_offset(123_456_789, 8, 1100, 0), 0.0);
        // Always within [0, tape).
        let tape = 6.0 * 1100.0;
        for t in [0u128, 1_000_000, 9_999_999_999, 123_456_789_012_345] {
            let o = global_offset(t, 8, 1100, 6);
            assert!(o >= 0.0 && o < tape, "offset {o} out of range for t={t}");
        }
    }

    #[test]
    fn visible_from_left_edge() {
        // 6 tickers of width 1000, screen 2500 wide, aligned at 0.
        let v = visible_tickers(0.0, 0.0, 2500.0, 6, 1000);
        // boxes at x = 0, 1000, 2000 are visible (2500 wide -> 3 boxes).
        assert_eq!(v.len(), 3);
        assert_eq!(v[0], VisibleTicker { index: 0, x: 0.0 });
        assert_eq!(v[1], VisibleTicker { index: 1, x: 1000.0 });
        assert_eq!(v[2], VisibleTicker { index: 2, x: 2000.0 });
    }

    #[test]
    fn second_screen_offset_shows_later_tickers() {
        // Screen 2 sits 2000px into the tape.
        let v = visible_tickers(0.0, 2000.0, 2000.0, 6, 1000);
        assert_eq!(v[0].index, 2);
        assert_eq!(v[1].index, 3);
    }

    #[test]
    fn wraps_around_the_end_of_the_tape() {
        // Localized near the end of a 6*1000 tape should wrap back to index 0.
        let v = visible_tickers(5500.0, 0.0, 1000.0, 6, 1000);
        // localized = 5500 -> first box index 5 at x=-500, next index 0 at x=500.
        assert_eq!(v[0].index, 5);
        assert_eq!(v[1].index, 0);
    }

    #[test]
    fn screen_wider_than_tape_repeats_tickers_no_gaps() {
        // Tape is 3*500 = 1500 wide; screen is 4000 wide. Must fill with repeats,
        // never emit gaps (this is the Go "nil ticker" bug, fixed).
        let v = visible_tickers(0.0, 0.0, 4000.0, 3, 500);
        // 4000/500 = 8 boxes fill the screen.
        assert_eq!(v.len(), 8);
        // Indices cycle 0,1,2,0,1,2,0,1 and x increments by 500 with no gaps.
        let expected_idx = [0, 1, 2, 0, 1, 2, 0, 1];
        for (k, vt) in v.iter().enumerate() {
            assert_eq!(vt.index, expected_idx[k]);
            assert_eq!(vt.x, k as f32 * 500.0);
        }
    }
}
