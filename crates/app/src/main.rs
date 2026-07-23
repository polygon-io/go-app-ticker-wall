//! `tickerwall` CLI entry point. Phase 3 wires up `server` and `describe`; the
//! `gui`, `update`, and `announce` subcommands plus full config precedence land
//! in phase 6.

mod config;

use std::sync::Arc;

use anyhow::Result;
use clap::{Args, Parser, Subcommand};
use config::FileConfig;
use tickerwall_proto::{PresentationSettings, Rgba};
use tokio_util::sync::CancellationToken;

#[derive(Parser)]
#[command(name = "tickerwall", version, about = "Massive.com Ticker Wall")]
struct Cli {
    /// Enable verbose (debug) logging.
    #[arg(short, long, global = true)]
    debug: bool,

    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    /// Start the leader: pull market data and serve it to GUIs over gRPC.
    Server(ServerArgs),
    /// Start a GUI window (one screen of the wall).
    Gui(GuiArgs),
    /// Update presentation settings of a running cluster (partial: only the flags
    /// you pass are changed).
    Update(UpdateArgs),
    /// Announce a message across the wall.
    Announce(AnnounceArgs),
    /// Describe a running cluster (screens + tickers).
    Describe(ClientArgs),
}

#[derive(Args)]
struct UpdateArgs {
    #[arg(short = 'l', long, default_value = "http://localhost:6886")]
    leader: String,
    #[arg(short = 's', long)]
    scroll_speed: Option<i32>,
    #[arg(short = 'w', long)]
    ticker_box_width: Option<i32>,
    #[arg(long)]
    animation_duration: Option<i32>,
    #[arg(long)]
    per_tick_updates: Option<bool>,
    #[arg(long)]
    show_fps: Option<bool>,
    #[arg(long)]
    show_logos: Option<bool>,
    /// RGBA as "r,g,b,a" (e.g. 255,255,255,255).
    #[arg(long)]
    up_color: Option<String>,
    #[arg(long)]
    down_color: Option<String>,
    #[arg(long)]
    bg_color: Option<String>,
    #[arg(long)]
    font_color: Option<String>,
    #[arg(long)]
    ticker_bg_color: Option<String>,
}

#[derive(Args)]
struct AnnounceArgs {
    /// The message to display.
    message: String,
    #[arg(short = 'l', long, default_value = "http://localhost:6886")]
    leader: String,
    /// info | danger | success
    #[arg(short = 't', long, default_value = "info")]
    r#type: String,
    /// elastic | bounce | ease | back
    #[arg(short = 'n', long, default_value = "elastic")]
    animation: String,
    /// How long the message stays up, in milliseconds.
    #[arg(short = 'i', long, default_value_t = 2000)]
    lifespan: i64,
}

#[derive(Args)]
struct GuiArgs {
    /// Leader gRPC address.
    #[arg(short = 'l', long, default_value = "http://localhost:6886")]
    leader: String,

    /// Window height in pixels.
    #[arg(short = 'y', long, default_value_t = 300)]
    screen_height: i32,

    /// Window width in pixels.
    #[arg(short = 'x', long, default_value_t = 1600)]
    screen_width: i32,

    /// Index of this screen in the wall (used for left-to-right ordering).
    #[arg(short = 'i', long, default_value_t = 1)]
    screen_index: i32,

    /// Borderless-fullscreen on the current monitor (for kiosk deployment).
    #[arg(short = 'f', long, default_value_t = false)]
    fullscreen: bool,
}

#[derive(Args)]
struct ServerArgs {
    /// Massive.com API key.
    #[arg(short = 'a', long, env = "TW_API_KEY")]
    api_key: String,

    /// Comma-separated ticker symbols to display.
    #[arg(short = 't', long, default_value = "AAPL,AMD,NVDA,SBUX,META,HOOD")]
    tickers: String,

    /// Port the gRPC server binds to.
    #[arg(short = 'g', long, default_value_t = 6886)]
    grpc_port: u16,

    /// Scroll speed (inverted: 1 is fastest).
    #[arg(short = 's', long, default_value_t = 8)]
    scroll_speed: i32,

    /// Ticker box width in pixels.
    #[arg(short = 'w', long, default_value_t = 1200)]
    ticker_box_width: i32,

    /// Notification animation duration in milliseconds.
    #[arg(long, default_value_t = 500)]
    animation_duration: i32,

    /// Update on every trade (true) vs once per second (false).
    #[arg(long, default_value_t = true)]
    per_tick_updates: bool,

    /// Show an FPS meter on each screen.
    #[arg(long, default_value_t = false)]
    show_fps: bool,
}

#[derive(Args)]
struct ClientArgs {
    /// Leader gRPC address.
    #[arg(short = 'l', long, default_value = "http://localhost:6886")]
    leader: String,
}

fn main() -> Result<()> {
    let cli = Cli::parse();
    init_tracing(cli.debug);

    match cli.command {
        // The GUI owns the main thread (OpenGL) and builds its own tokio runtime,
        // so it must NOT run inside an outer runtime.
        Command::Gui(args) => tickerwall_gui::run(tickerwall_gui::Config {
            leader: args.leader,
            screen_width: args.screen_width,
            screen_height: args.screen_height,
            screen_index: args.screen_index,
            fullscreen: args.fullscreen,
        }),
        Command::Server(args) => block_on(run_server(args)),
        Command::Update(args) => block_on(run_update(args)),
        Command::Announce(args) => block_on(run_announce(args)),
        Command::Describe(args) => block_on(run_describe(args)),
    }
}

/// Run an async task to completion on a fresh tokio runtime.
fn block_on<F: std::future::Future<Output = Result<()>>>(fut: F) -> Result<()> {
    tokio::runtime::Runtime::new()?.block_on(fut)
}

fn init_tracing(debug: bool) {
    let default = if debug { "debug" } else { "info" };
    let filter = tracing_subscriber::EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new(default));
    tracing_subscriber::fmt().with_env_filter(filter).init();
}

async fn run_server(args: ServerArgs) -> Result<()> {
    let mut settings = default_settings();
    settings.scroll_speed = args.scroll_speed;
    settings.ticker_box_width = args.ticker_box_width;
    settings.animation_duration_ms = args.animation_duration;
    settings.per_tick_updates = args.per_tick_updates;
    settings.show_fps = args.show_fps;

    let tickers = args
        .tickers
        .split(',')
        .map(|s| s.trim().to_uppercase())
        .filter(|s| !s.is_empty())
        .collect();

    let config = tickerwall_leader::Config {
        api_key: args.api_key,
        tickers,
        settings,
    };

    let leader = Arc::new(tickerwall_leader::Leader::new(config));
    leader.load_initial_data().await?;

    let cancel = CancellationToken::new();
    leader.spawn_background(cancel.clone());

    {
        let cancel = cancel.clone();
        tokio::spawn(async move {
            let _ = tokio::signal::ctrl_c().await;
            tracing::info!("shutting down…");
            cancel.cancel();
        });
    }

    let addr = format!("0.0.0.0:{}", args.grpc_port).parse()?;
    tickerwall_leader::serve_grpc(leader, addr, cancel).await
}

async fn run_describe(args: ClientArgs) -> Result<()> {
    use tickerwall_proto::leader_client::LeaderClient;
    use tickerwall_proto::Empty;

    let mut client = LeaderClient::connect(args.leader).await?;
    let snapshot = client.get_snapshot(Empty {}).await?.into_inner();
    print_snapshot(&snapshot);
    Ok(())
}

async fn run_update(args: UpdateArgs) -> Result<()> {
    use tickerwall_proto::leader_client::LeaderClient;
    use tickerwall_proto::SettingsPatch;

    // Only fields the user passed become `Some`, so the leader merges them and
    // leaves everything else untouched (fixes the Go "reset to defaults" bug).
    let patch = SettingsPatch {
        scroll_speed: args.scroll_speed,
        ticker_box_width: args.ticker_box_width,
        animation_duration_ms: args.animation_duration,
        per_tick_updates: args.per_tick_updates,
        show_fps: args.show_fps,
        show_logos: args.show_logos,
        up_color: parse_color_opt(&args.up_color)?,
        down_color: parse_color_opt(&args.down_color)?,
        bg_color: parse_color_opt(&args.bg_color)?,
        font_color: parse_color_opt(&args.font_color)?,
        ticker_box_bg_color: parse_color_opt(&args.ticker_bg_color)?,
    };

    let mut client = LeaderClient::connect(args.leader).await?;
    client.update_settings(patch).await?;
    tracing::info!("settings updated");
    Ok(())
}

async fn run_announce(args: AnnounceArgs) -> Result<()> {
    use tickerwall_proto::leader_client::LeaderClient;
    use tickerwall_proto::{AnnounceRequest, AnnouncementAnimation, AnnouncementType};

    let r#type = match args.r#type.as_str() {
        "danger" => AnnouncementType::Danger,
        "success" => AnnouncementType::Success,
        _ => AnnouncementType::Info,
    } as i32;
    let animation = match args.animation.as_str() {
        "bounce" => AnnouncementAnimation::Bounce,
        "ease" => AnnouncementAnimation::Ease,
        "back" => AnnouncementAnimation::Back,
        _ => AnnouncementAnimation::Elastic,
    } as i32;

    let mut client = LeaderClient::connect(args.leader).await?;
    client
        .announce(AnnounceRequest {
            message: args.message,
            r#type,
            lifespan_ms: args.lifespan,
            animation,
        })
        .await?;
    tracing::info!("announcement sent");
    Ok(())
}

/// Parse an optional "r,g,b,a" string into an `Rgba`.
fn parse_color_opt(s: &Option<String>) -> Result<Option<Rgba>> {
    match s {
        Some(s) => Ok(Some(parse_color(s)?)),
        None => Ok(None),
    }
}

fn parse_color(s: &str) -> Result<Rgba> {
    let parts: Vec<&str> = s.split(',').collect();
    if parts.len() != 4 {
        anyhow::bail!("color must be 'r,g,b,a' (4 values), got: {s}");
    }
    let n = |i: usize| -> Result<i32> {
        parts[i]
            .trim()
            .parse::<i32>()
            .map_err(|e| anyhow::anyhow!("invalid color component '{}': {e}", parts[i]))
    };
    Ok(rgba(n(0)?, n(1)?, n(2)?, n(3)?))
}

fn print_snapshot(snapshot: &tickerwall_proto::Snapshot) {
    let cluster = match &snapshot.cluster {
        Some(c) => c,
        None => {
            println!("(no cluster data)");
            return;
        }
    };
    if let Some(s) = &cluster.settings {
        println!("Global Viewport Size: {} px", cluster.global_viewport_size());
        println!("Animation Duration: {} ms", s.animation_duration_ms);
        println!("Scroll Speed: {}", s.scroll_speed);
        println!("Ticker Box Width: {} px", s.ticker_box_width);
        println!("Per Tick Updates: {}", s.per_tick_updates);
    }
    println!("Screen Count: {}", cluster.number_of_screens());
    println!("Screen Details:");
    for screen in &cluster.screens {
        println!(" ------------ ");
        println!(" Screen ID: {}", screen.uuid);
        println!(" - Width {} px", screen.width);
        println!(" - Height {} px", screen.height);
        println!(" - Index {}", screen.index);
    }
    println!(" ------------ ");
    println!("Ticker count: {}", snapshot.tickers.len());
    println!("Tickers:");
    for t in &snapshot.tickers {
        println!(" -  {}  [  {}  ]", t.symbol, t.company_name);
    }
}

/// Default presentation settings (mirror the Go CLI defaults).
fn default_settings() -> PresentationSettings {
    PresentationSettings {
        ticker_box_width: 1200,
        scroll_speed: 8,
        up_color: Some(rgba(51, 255, 51, 255)),
        down_color: Some(rgba(255, 51, 51, 255)),
        bg_color: Some(rgba(1, 1, 1, 255)),
        font_color: Some(rgba(255, 255, 255, 255)),
        ticker_box_bg_color: Some(rgba(20, 20, 20, 255)),
        show_logos: false,
        show_fps: false,
        animation_duration_ms: 500,
        per_tick_updates: true,
    }
}

fn rgba(red: i32, green: i32, blue: i32, alpha: i32) -> Rgba {
    Rgba {
        red,
        green,
        blue,
        alpha,
    }
}
