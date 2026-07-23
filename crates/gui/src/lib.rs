//! The rendering layer: a winit + glutin window driving a femtovg render loop
//! that draws the scrolling tape, mini price graphs, notifications, and the
//! system panel. Replaces the Go `gui`, `gui/notifications`, and `fonts` packages.

mod app;
mod draw;
mod ease;
mod fonts;
mod layout;
mod notifications;

use std::sync::Arc;

use anyhow::Result;
use tickerwall_client::{ClusterClient, Config as ClientConfig};
use tokio_util::sync::CancellationToken;
use winit::event_loop::{ControlFlow, EventLoop};

/// GUI configuration.
pub struct Config {
    pub leader: String,
    pub screen_width: i32,
    pub screen_height: i32,
    pub screen_index: i32,
    /// Borderless-fullscreen on the current monitor (for kiosk deployment).
    pub fullscreen: bool,
}

/// Run the GUI. Spins up a tokio runtime for the cluster client, then runs the
/// winit event loop on the calling (main) thread. Returns when the window closes.
pub fn run(config: Config) -> Result<()> {
    let runtime = tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()?;

    let fullscreen = config.fullscreen;
    let client = ClusterClient::new(ClientConfig {
        leader: config.leader,
        screen_width: config.screen_width,
        screen_height: config.screen_height,
        screen_index: config.screen_index,
    });

    let cancel = CancellationToken::new();

    // Background: keep local state synced with the leader.
    {
        let client = Arc::clone(&client);
        let cancel = cancel.clone();
        runtime.spawn(async move {
            if let Err(e) = client.run(cancel).await {
                app::log_client_error(e);
            }
        });
    }

    // Foreground: the render loop (must own the main thread for OpenGL).
    let event_loop = EventLoop::new()?;
    event_loop.set_control_flow(ControlFlow::Poll);
    let mut application = app::App::new(client, fullscreen);
    let result = event_loop.run_app(&mut application);

    // Tear down the background client.
    cancel.cancel();
    // Keep the runtime alive briefly for a graceful shutdown, then drop it.
    runtime.shutdown_timeout(std::time::Duration::from_millis(200));

    result.map_err(Into::into)
}
