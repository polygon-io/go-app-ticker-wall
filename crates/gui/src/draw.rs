//! Drawing routines: background, ticker boxes, the mini price graph, and the
//! system/status panel. Ports `gui/tickers.go` and `gui/system.go` onto femtovg.

use femtovg::{Align, Baseline, Canvas, Color, Paint, Path, Renderer};
use tickerwall_proto::{PresentationSettings, Rgba, Screen, Ticker};

use crate::fonts::Fonts;
use crate::layout::{self, VisibleTicker};

// Ticker box geometry (from the Go constants).
const TICKER_BOX_HEIGHT: f32 = 240.0;
const TICKER_BOX_MARGIN: f32 = 30.0;
const TICKER_BOX_PADDING: f32 = 50.0;
const TICKER_BOX_BORDER_RADIUS: f32 = 8.0;
const UPPER_ROW_FONT_SIZE: f32 = 96.0;
const BOTTOM_ROW_FONT_SIZE: f32 = 58.0;
const MAX_COMPANY_NAME_CHARS: usize = 14;

// Mini graph.
const GRAPH_SIZE: f32 = 180.0;
const GRAPH_VIEWPORT_PERCENTAGE: f32 = 0.04;

pub fn color_of(c: &Option<Rgba>, fallback: Color) -> Color {
    match c {
        Some(r) => Color::rgba(r.red as u8, r.green as u8, r.blue as u8, r.alpha as u8),
        None => fallback,
    }
}

/// Render every visible ticker for this screen.
#[allow(clippy::too_many_arguments)]
pub fn render_tickers<T: Renderer>(
    canvas: &mut Canvas<T>,
    fonts: &Fonts,
    settings: &PresentationSettings,
    screen: &Screen,
    tickers: &[Ticker],
    global_offset: f32,
    screen_offset: f32,
    window_width: f32,
) {
    let visible = layout::visible_tickers(
        global_offset,
        screen_offset,
        window_width,
        tickers.len(),
        settings.ticker_box_width,
    );
    for VisibleTicker { index, x } in visible {
        if let Some(ticker) = tickers.get(index) {
            render_ticker(canvas, fonts, settings, screen, ticker, x);
        }
    }
}

fn render_ticker_bg<T: Renderer>(
    canvas: &mut Canvas<T>,
    settings: &PresentationSettings,
    screen: &Screen,
    left_offset: f32,
) {
    let top = (screen.height as f32 / 2.0) - (TICKER_BOX_HEIGHT / 2.0);
    let left = left_offset + (TICKER_BOX_MARGIN / 2.0);
    let box_width = settings.ticker_box_width as f32 - TICKER_BOX_MARGIN;

    let mut path = Path::new();
    path.rounded_rect(left, top, box_width, TICKER_BOX_HEIGHT, TICKER_BOX_BORDER_RADIUS);
    let bg = color_of(&settings.ticker_box_bg_color, Color::rgb(20, 20, 20));
    canvas.fill_path(&path, &Paint::color(bg));
}

fn render_ticker<T: Renderer>(
    canvas: &mut Canvas<T>,
    fonts: &Fonts,
    settings: &PresentationSettings,
    screen: &Screen,
    ticker: &Ticker,
    ticker_offset: f32,
) {
    render_ticker_bg(canvas, settings, screen, ticker_offset);

    let offset_left = ticker_offset + (TICKER_BOX_MARGIN / 2.0) + TICKER_BOX_PADDING;
    let offset_top = (screen.height as f32 / 2.0) - (TICKER_BOX_HEIGHT / 2.0);
    let offset_right =
        (ticker_offset + settings.ticker_box_width as f32 - TICKER_BOX_MARGIN) - TICKER_BOX_PADDING;
    let upper_row_top = offset_top + (TICKER_BOX_HEIGHT * 0.33);
    let lower_row_top = offset_top + (TICKER_BOX_HEIGHT * 0.66);

    let font_color = color_of(&settings.font_color, Color::white());

    // Symbol (upper-left, bold).
    let mut top_paint = Paint::color(font_color);
    top_paint.set_font(&[fonts.bold]);
    top_paint.set_font_size(UPPER_ROW_FONT_SIZE);
    top_paint.set_text_baseline(Baseline::Middle);
    top_paint.set_text_align(Align::Left);
    let _ = canvas.fill_text(offset_left, upper_row_top, &ticker.symbol, &top_paint);

    // Price (upper-right, bold).
    top_paint.set_text_align(Align::Right);
    let price = format!("{:.2}", ticker.price);
    let _ = canvas.fill_text(offset_right, upper_row_top, &price, &top_paint);

    // Company name (lower-left, light, truncated).
    let mut name = ticker.company_name.clone();
    if name.chars().count() >= MAX_COMPANY_NAME_CHARS {
        let truncated: String = name.chars().take(MAX_COMPANY_NAME_CHARS - 3).collect();
        name = format!("{truncated}...");
    }
    let mut bottom_paint = Paint::color(font_color);
    bottom_paint.set_font(&[fonts.light]);
    bottom_paint.set_font_size(BOTTOM_ROW_FONT_SIZE);
    bottom_paint.set_text_baseline(Baseline::Middle);
    bottom_paint.set_text_align(Align::Left);
    let _ = canvas.fill_text(offset_left, lower_row_top, &name, &bottom_paint);

    // Directional color for change + graph.
    let directional = if ticker.price_change_percentage < 0.0 {
        color_of(&settings.down_color, Color::rgb(255, 51, 51))
    } else {
        color_of(&settings.up_color, Color::rgb(51, 255, 51))
    };

    // Change: +diff (+pct%) lower-right, directional color.
    let price_diff = ticker.price - ticker.previous_close_price;
    let change = format!("{:+.2} ({:+.2}%)", price_diff, ticker.price_change_percentage);
    let mut change_paint = Paint::color(directional);
    change_paint.set_font(&[fonts.light]);
    change_paint.set_font_size(BOTTOM_ROW_FONT_SIZE);
    change_paint.set_text_baseline(Baseline::Middle);
    change_paint.set_text_align(Align::Right);
    let _ = canvas.fill_text(offset_right, lower_row_top, &change, &change_paint);

    // Mini graph.
    let graph_top = (screen.height as f32 / 2.0) - (GRAPH_SIZE / 2.0);
    draw_graph(canvas, ticker, offset_left + 400.0, graph_top, GRAPH_SIZE, GRAPH_SIZE, directional);
}

#[allow(clippy::too_many_arguments)]
fn draw_graph<T: Renderer>(
    canvas: &mut Canvas<T>,
    ticker: &Ticker,
    x: f32,
    y: f32,
    w: f32,
    h: f32,
    color: Color,
) {
    let points = ticker.aggs.len();
    if points < 2 {
        return;
    }

    let dx = w / (points as f32 - 1.0);
    let mut sx = vec![0.0f32; points];
    let mut sy = vec![0.0f32; points];

    let mut min = f32::INFINITY;
    let mut max = f32::NEG_INFINITY;
    for (i, agg) in ticker.aggs.iter().enumerate() {
        let p = agg.price as f32;
        min = min.min(p);
        max = max.max(p);
        sy[i] = p;
        sx[i] = x + i as f32 * dx;
    }

    let mid_range = (min + max) / 2.0;
    if mid_range == 0.0 {
        return;
    }

    // Normalize into a fraction of viewport movement, tracking the largest swing.
    let mut abs_max = 0.0f32;
    for v in sy.iter_mut() {
        *v = (*v - mid_range) / mid_range;
        abs_max = abs_max.max(v.abs());
    }
    // Squish if it exceeds the allowed viewport range.
    if abs_max > GRAPH_VIEWPORT_PERCENTAGE {
        for v in sy.iter_mut() {
            *v = (*v / abs_max) * GRAPH_VIEWPORT_PERCENTAGE;
        }
    }
    // Map the normalized value into pixels.
    let middle = h / 2.0;
    let base_multiplier = middle / GRAPH_VIEWPORT_PERCENTAGE;
    for v in sy.iter_mut() {
        *v = (y + h) - ((base_multiplier * *v) + middle);
    }

    let mut line = Path::new();
    line.move_to(sx[0], sy[0]);
    for i in 1..points {
        line.line_to(sx[i], sy[i]);
    }
    let mut stroke = Paint::color(color);
    stroke.set_line_width(4.0);
    canvas.stroke_path(&line, &stroke);

    let mut dot = Path::new();
    dot.circle(sx[points - 1], sy[points - 1], 6.0);
    canvas.fill_path(&dot, &Paint::color(color));
}

/// Small FPS readout pinned to the top-left corner. Shown only when the
/// `show_fps` presentation setting is enabled. Sizes itself off the screen
/// height so it stays legible from a 300px dev window up to a 1080p wall.
pub fn fps_meter<T: Renderer>(
    canvas: &mut Canvas<T>,
    fonts: &Fonts,
    screen_height: f32,
    fps: f32,
) {
    let font_size = (screen_height * 0.06).clamp(20.0, 64.0);
    let pad = font_size * 0.4;
    let text = format!("{fps:.0} FPS");
    let box_w = font_size * 5.0;
    let box_h = font_size + pad * 2.0;

    let mut path = Path::new();
    path.rounded_rect(pad, pad, box_w, box_h, 6.0);
    canvas.fill_path(&path, &Paint::color(Color::rgba(0, 0, 0, 160)));

    let mut paint = Paint::color(Color::rgb(0, 255, 0));
    paint.set_font(&[fonts.bold]);
    paint.set_font_size(font_size);
    paint.set_text_align(Align::Left);
    paint.set_text_baseline(Baseline::Middle);
    let _ = canvas.fill_text(pad * 2.0, pad + box_h / 2.0, &text, &paint);
}

/// Centered red panel shown when the client is not connected to the leader.
pub fn system_panel<T: Renderer>(
    canvas: &mut Canvas<T>,
    fonts: &Fonts,
    window_width: f32,
    screen: &Screen,
    message: &str,
) {
    let panel_height = 200.0f32;
    let padding = 20.0f32;
    let from_top = (screen.height as f32 / 2.0) - (panel_height / 2.0);

    let mut path = Path::new();
    path.rounded_rect(padding, from_top, window_width - (padding * 2.0), panel_height, 5.0);
    canvas.fill_path(&path, &Paint::color(Color::rgba(255, 0, 0, 222)));

    let mut text = Paint::color(Color::rgba(255, 255, 255, 255));
    text.set_font(&[fonts.bold]);
    text.set_font_size(32.0);
    text.set_text_align(Align::Center);
    text.set_text_baseline(Baseline::Middle);
    let _ = canvas.fill_text(
        screen.width as f32 / 2.0,
        screen.height as f32 / 2.0,
        message,
        &text,
    );
}
