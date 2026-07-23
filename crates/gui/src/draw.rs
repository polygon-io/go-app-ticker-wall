//! Drawing routines: background, ticker boxes, the mini price graph, and the
//! system/status panel. Ports `gui/tickers.go` and `gui/system.go` onto femtovg.

use femtovg::{Align, Baseline, Canvas, Color, Paint, Path, Renderer};
use tickerwall_proto::{Mover, PresentationSettings, Rgba, Screen, ScreenCluster, Ticker};

use crate::fonts::Fonts;
use crate::layout::{self, VisibleTicker};

// Secondary "market movers" tape (pinned to the bottom of each screen).
/// Pixel width of one mover entry on the secondary tape.
pub const MOVERS_BOX_WIDTH: i32 = 680;
/// Tape height as a fraction of screen height, clamped to a sane pixel range.
const MOVERS_TAPE_HEIGHT_FRAC: f32 = 0.20;
const MOVERS_TAPE_MIN_H: f32 = 70.0;
const MOVERS_TAPE_MAX_H: f32 = 220.0;

/// Height of the movers tape for a given screen height.
pub fn movers_tape_height(screen_height: f32) -> f32 {
    (screen_height * MOVERS_TAPE_HEIGHT_FRAC).clamp(MOVERS_TAPE_MIN_H, MOVERS_TAPE_MAX_H)
}

// Ticker box geometry (from the Go constants).
const TICKER_BOX_HEIGHT: f32 = 240.0;
const TICKER_BOX_MARGIN: f32 = 30.0;
const TICKER_BOX_PADDING: f32 = 50.0;
const TICKER_BOX_BORDER_RADIUS: f32 = 8.0;
const UPPER_ROW_FONT_SIZE: f32 = 96.0;
const BOTTOM_ROW_FONT_SIZE: f32 = 58.0;
const MAX_COMPANY_NAME_CHARS: usize = 14;

// Full-bleed graph: opacity of the translucent area fill under the price line,
// and how far (as a fraction of box height) the line is kept off the top/bottom
// edges so it never clips into the box border.
const GRAPH_FILL_ALPHA: f32 = 0.22;
const GRAPH_VERTICAL_PAD: f32 = 0.12;

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

    // Box interior geometry (matches render_ticker_bg).
    let box_top = (screen.height as f32 / 2.0) - (TICKER_BOX_HEIGHT / 2.0);
    let box_left = ticker_offset + (TICKER_BOX_MARGIN / 2.0);
    let box_width = settings.ticker_box_width as f32 - TICKER_BOX_MARGIN;

    // Directional color drives both the change text and the graph.
    let directional = if ticker.price_change_percentage < 0.0 {
        color_of(&settings.down_color, Color::rgb(255, 51, 51))
    } else {
        color_of(&settings.up_color, Color::rgb(51, 255, 51))
    };

    // Full-bleed price graph filling the whole box, behind the text.
    draw_graph(canvas, ticker, box_left, box_top, box_width, TICKER_BOX_HEIGHT, directional);

    // --- Text layer, drawn on top of the graph ---
    let offset_left = ticker_offset + (TICKER_BOX_MARGIN / 2.0) + TICKER_BOX_PADDING;
    let offset_right =
        (ticker_offset + settings.ticker_box_width as f32 - TICKER_BOX_MARGIN) - TICKER_BOX_PADDING;
    let upper_row_top = box_top + (TICKER_BOX_HEIGHT * 0.33);
    let lower_row_top = box_top + (TICKER_BOX_HEIGHT * 0.66);

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

    // Change: +diff (+pct%) lower-right, directional color.
    let price_diff = ticker.price - ticker.previous_close_price;
    let change = format!("{:+.2} ({:+.2}%)", price_diff, ticker.price_change_percentage);
    let mut change_paint = Paint::color(directional);
    change_paint.set_font(&[fonts.light]);
    change_paint.set_font_size(BOTTOM_ROW_FONT_SIZE);
    change_paint.set_text_baseline(Baseline::Middle);
    change_paint.set_text_align(Align::Right);
    let _ = canvas.fill_text(offset_right, lower_row_top, &change, &change_paint);
}

/// Full-bleed price graph filling the whole ticker box, drawn behind the text:
/// a translucent area fill under a solid price line, plus an endpoint dot. Prices
/// are normalized to the aggregate min/max so the line uses the full box height.
#[allow(clippy::too_many_arguments)]
fn draw_graph<T: Renderer>(
    canvas: &mut Canvas<T>,
    ticker: &Ticker,
    box_left: f32,
    box_top: f32,
    box_width: f32,
    box_height: f32,
    color: Color,
) {
    let points = ticker.aggs.len();
    if points < 2 {
        return;
    }

    // Inset by the corner radius so the fill never spills past the box's rounded
    // corners (avoids needing a clip/scissor).
    let inset = TICKER_BOX_BORDER_RADIUS;
    let x0 = box_left + inset;
    let y0 = box_top + inset;
    let w = box_width - inset * 2.0;
    let h = box_height - inset * 2.0;
    let bottom = y0 + h;

    let mut min = f32::INFINITY;
    let mut max = f32::NEG_INFINITY;
    for a in &ticker.aggs {
        let p = a.price as f32;
        min = min.min(p);
        max = max.max(p);
    }
    let range = (max - min).max(f32::EPSILON);

    // Keep the line off the very top/bottom edges.
    let v_pad = h * GRAPH_VERTICAL_PAD;
    let usable_h = h - v_pad * 2.0;
    let dx = w / (points as f32 - 1.0);
    let xy = |i: usize| -> (f32, f32) {
        let norm = (ticker.aggs[i].price as f32 - min) / range; // 0 at low, 1 at high
        (x0 + i as f32 * dx, y0 + v_pad + (1.0 - norm) * usable_h)
    };

    // Translucent area under the line (line points, then down to the baseline).
    let (sx, sy) = xy(0);
    let mut area = Path::new();
    area.move_to(sx, sy);
    for i in 1..points {
        let (x, y) = xy(i);
        area.line_to(x, y);
    }
    let (ex, _) = xy(points - 1);
    area.line_to(ex, bottom);
    area.line_to(x0, bottom);
    area.close();
    let mut fill_color = color;
    fill_color.a = GRAPH_FILL_ALPHA;
    canvas.fill_path(&area, &Paint::color(fill_color));

    // Solid price line on top of the fill.
    let mut line = Path::new();
    line.move_to(sx, sy);
    for i in 1..points {
        let (x, y) = xy(i);
        line.line_to(x, y);
    }
    let mut stroke = Paint::color(color);
    stroke.set_line_width(6.0);
    canvas.stroke_path(&line, &stroke);

    // Endpoint dot at the latest price.
    let (lx, ly) = xy(points - 1);
    let mut dot = Path::new();
    dot.circle(lx, ly, 8.0);
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

/// Render the secondary gainers/losers tape pinned to the bottom of the screen.
/// Like the primary tape it spans the whole cluster (so it's continuous across
/// screens) but scrolls at its own speed and shows a compact `SYMBOL price +x%`
/// entry per mover, colored green/red by direction.
#[allow(clippy::too_many_arguments)]
pub fn render_movers_tape<T: Renderer>(
    canvas: &mut Canvas<T>,
    fonts: &Fonts,
    settings: &PresentationSettings,
    screen: &Screen,
    _cluster: &ScreenCluster,
    movers: &[Mover],
    global_offset: f32,
    screen_offset: f32,
) {
    if movers.is_empty() {
        return;
    }
    let width = screen.width as f32;
    let tape_height = movers_tape_height(screen.height as f32);
    let strip_top = screen.height as f32 - tape_height;

    // Strip background + a subtle top divider.
    let mut bg = Path::new();
    bg.rect(0.0, strip_top, width, tape_height);
    canvas.fill_path(&bg, &Paint::color(Color::rgba(0, 0, 0, 190)));
    let mut divider = Path::new();
    divider.rect(0.0, strip_top, width, 2.0);
    canvas.fill_path(&divider, &Paint::color(Color::rgba(255, 255, 255, 40)));

    let visible = layout::visible_tickers(
        global_offset,
        screen_offset,
        width,
        movers.len(),
        MOVERS_BOX_WIDTH,
    );
    for VisibleTicker { index, x } in visible {
        if let Some(mover) = movers.get(index) {
            draw_mover(canvas, fonts, settings, mover, x, strip_top, tape_height);
        }
    }
}

fn draw_mover<T: Renderer>(
    canvas: &mut Canvas<T>,
    fonts: &Fonts,
    settings: &PresentationSettings,
    mover: &Mover,
    x: f32,
    strip_top: f32,
    tape_height: f32,
) {
    let mid_y = strip_top + tape_height / 2.0;
    let font_size = (tape_height * 0.42).clamp(20.0, 72.0);
    let pad = font_size * 0.9;
    let gap = font_size * 0.5;
    let box_right = x + MOVERS_BOX_WIDTH as f32;

    let dir_color = if mover.todays_change_percentage < 0.0 {
        color_of(&settings.down_color, Color::rgb(255, 51, 51))
    } else {
        color_of(&settings.up_color, Color::rgb(51, 255, 51))
    };
    let font_color = color_of(&settings.font_color, Color::white());

    // Symbol (bold, left).
    let mut sym = Paint::color(font_color);
    sym.set_font(&[fonts.bold]);
    sym.set_font_size(font_size);
    sym.set_text_baseline(Baseline::Middle);
    sym.set_text_align(Align::Left);
    let _ = canvas.fill_text(x + pad, mid_y, &mover.symbol, &sym);

    // Change % (bold, directional) anchored to the right edge of the box.
    let change = format!("{:+.2}%", mover.todays_change_percentage);
    let mut chg = Paint::color(dir_color);
    chg.set_font(&[fonts.bold]);
    chg.set_font_size(font_size);
    chg.set_text_baseline(Baseline::Middle);
    chg.set_text_align(Align::Right);
    let _ = canvas.fill_text(box_right - pad, mid_y, &change, &chg);

    // Price (light) right-aligned just left of the change %. Measuring the change
    // width means a long (3+ digit) percentage can never overlap the price.
    let change_w = canvas
        .measure_text(0.0, 0.0, &change, &chg)
        .map(|m| m.width())
        .unwrap_or(0.0);
    let price = format!("{:.2}", mover.price);
    let mut pr = Paint::color(font_color);
    pr.set_font(&[fonts.light]);
    pr.set_font_size(font_size * 0.85);
    pr.set_text_baseline(Baseline::Middle);
    pr.set_text_align(Align::Right);
    let _ = canvas.fill_text(box_right - pad - change_w - gap, mid_y, &price, &pr);
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
