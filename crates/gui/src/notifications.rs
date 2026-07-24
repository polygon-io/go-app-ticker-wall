//! Full-screen announcement overlays with eased slide in/out. Ports
//! `gui/notifications/`. The banner spans the whole cluster so its text lands in
//! the true center across all screens.

use std::time::{SystemTime, UNIX_EPOCH};

use femtovg::{Align, Baseline, Canvas, Color, Paint, Path, Renderer};
use tickerwall_proto::{Announcement, PresentationSettings, Screen, ScreenCluster};

use crate::ease;
use crate::fonts::Fonts;

fn now_ms() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as i64
}

type EaseFn = fn(f32) -> f32;

struct Notification {
    announcement: Announcement,
    completed: bool,
    anim_in: EaseFn,
    anim_out: EaseFn,
}

impl Notification {
    fn new(announcement: Announcement) -> Self {
        // Match the Go pairing of intro/outro easings per animation type.
        let (anim_out, anim_in): (EaseFn, EaseFn) = match announcement.animation {
            // Bounce: bouncy on the way in, but a single springy wind-up on the way
            // out. The Go original used `in_elastic` here, whose multiple sine
            // oscillations made the banner visibly jitter up/down as it left;
            // `in_back` winds up once and slides off cleanly.
            1 => (ease::in_back, ease::out_bounce), // Bounce
            2 => (ease::in_quint, ease::out_quint), // Ease
            3 => (ease::in_back, ease::out_back),   // Back
            _ => (ease::in_elastic, ease::out_elastic), // Elastic (default; oscillation is expected here)
        };
        Self {
            announcement,
            completed: false,
            anim_in,
            anim_out,
        }
    }

    /// Whether to draw this frame; flips `completed` once fully expired.
    fn should_render(&mut self, animation_duration_ms: i32) -> bool {
        let now = now_ms();
        let show_at = self.announcement.show_at_timestamp_ms;
        let end = show_at + self.announcement.lifespan_ms + animation_duration_ms as i64;
        if now < show_at {
            false
        } else if now > end {
            self.completed = true;
            false
        } else {
            true
        }
    }

    fn render<T: Renderer>(
        &self,
        canvas: &mut Canvas<T>,
        fonts: &Fonts,
        cluster: &ScreenCluster,
        screen: &Screen,
        animation_duration_ms: i32,
    ) {
        let now = now_ms();
        let anim_dur = animation_duration_ms.max(1) as f64;
        let h = screen.height as f64;

        // Rest the text at the true vertical center of the (full-height) banner.
        // The Go original hardcoded 140, which only looked centered on its default
        // ~300px screens; deriving it from height keeps it centered at any size.
        // Baseline::Middle means this y is the text's vertical midpoint.
        let text_top_end = h / 2.0;
        // Start off the top of the screen so it slides down into place.
        let text_top_start = -(h / 2.0) - 160.0;
        let bg_bottom_start = 0.0;
        let bg_bottom_end = h;

        let mut text_top = text_top_end;
        let mut bg_bottom = bg_bottom_end;

        let show_at = self.announcement.show_at_timestamp_ms;
        let lifespan = self.announcement.lifespan_ms;

        if now - show_at < anim_dur as i64 {
            // Enter animation.
            let perc = (now - show_at) as f64 / anim_dur;
            let inp = (self.anim_in)(perc as f32) as f64;
            bg_bottom = bg_bottom_start - (bg_bottom_start - bg_bottom_end) * inp;
            text_top = text_top_start - (text_top_start - text_top_end) * inp;
        } else if now > show_at + lifespan {
            // Exit animation.
            let perc = (now - (show_at + lifespan)) as f64 / anim_dur;
            let outp = (self.anim_out)(perc as f32) as f64;
            bg_bottom = bg_bottom_end - (bg_bottom_end - bg_bottom_start) * outp;
            text_top = text_top_end - (text_top_end - text_top_start) * outp;
        }
        let bg_top = bg_bottom - h;

        let screen_offset = cluster.screen_global_offset(&screen.uuid);
        let left = -screen_offset;
        let cluster_width = cluster.global_viewport_size() as f32;

        let mut path = Path::new();
        path.rect(left, bg_top as f32, cluster_width, bg_bottom as f32);
        let bg = match self.announcement.r#type {
            1 => Color::rgba(255, 122, 122, 222), // danger
            2 => Color::rgba(122, 255, 122, 222), // success
            _ => Color::rgba(122, 122, 255, 222), // info
        };
        canvas.fill_path(&path, &Paint::color(bg));

        let mut text = Paint::color(Color::rgba(255, 255, 255, 255));
        text.set_font(&[fonts.bold]);
        text.set_font_size(96.0);
        text.set_text_align(Align::Center);
        text.set_text_baseline(Baseline::Middle);
        let middle = (cluster_width / 2.0) - screen_offset;
        let _ = canvas.fill_text(middle, text_top as f32, &self.announcement.message, &text);
    }
}

/// Owns the active announcement overlays.
#[derive(Default)]
pub struct Notifications {
    items: Vec<Notification>,
}

impl Notifications {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn add(&mut self, announcement: Announcement) {
        self.items.push(Notification::new(announcement));
    }

    /// Update, draw, and garbage-collect the active notifications.
    pub fn render<T: Renderer>(
        &mut self,
        canvas: &mut Canvas<T>,
        fonts: &Fonts,
        settings: &PresentationSettings,
        cluster: &ScreenCluster,
        screen: &Screen,
    ) {
        let anim_dur = settings.animation_duration_ms;
        for n in &mut self.items {
            if n.should_render(anim_dur) {
                n.render(canvas, fonts, cluster, screen, anim_dur);
            }
        }
        self.items.retain(|n| !n.completed);
    }
}
