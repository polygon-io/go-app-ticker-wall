//! winit `ApplicationHandler` + glutin OpenGL context + the per-frame render loop.
//! The GUI main thread owns rendering; a background tokio task (the client) keeps
//! shared state fresh, which this loop reads every frame.

use std::num::NonZeroU32;
use std::sync::Arc;
use std::time::Instant;

use femtovg::{renderer::OpenGl, Canvas, Color};
use glutin::config::{ConfigTemplateBuilder, GlConfig};
use glutin::context::{
    ContextAttributesBuilder, NotCurrentGlContext, PossiblyCurrentContext,
};
use glutin::display::{GetGlDisplay, GlDisplay};
use glutin::surface::{GlSurface, Surface, SwapInterval, WindowSurface};
use glutin_winit::{DisplayBuilder, GlWindow};
use raw_window_handle::HasWindowHandle;
use tickerwall_client::{ClusterClient, GrpcStatus};
use tracing::error;
use winit::application::ApplicationHandler;
use winit::event::WindowEvent;
use winit::event_loop::ActiveEventLoop;
use winit::window::{Window, WindowId};

use crate::draw;
use crate::fonts::Fonts;
use crate::layout;
use crate::notifications::Notifications;

/// Everything created once the event loop is `resumed` (needs an active display).
struct RenderState {
    window: Window,
    surface: Surface<WindowSurface>,
    context: PossiblyCurrentContext,
    canvas: Canvas<OpenGl>,
    fonts: Fonts,
    last_logical_size: (i32, i32),
}

/// Rolling frames-per-second counter. Averages over ~half a second so the
/// on-screen readout is stable instead of flickering every frame.
struct FpsCounter {
    last: Option<Instant>,
    accum_secs: f32,
    accum_frames: u32,
    display: f32,
}

impl FpsCounter {
    fn new() -> Self {
        Self { last: None, accum_secs: 0.0, accum_frames: 0, display: 0.0 }
    }

    /// Record a frame boundary and return the current averaged FPS.
    fn tick(&mut self) -> f32 {
        let now = Instant::now();
        if let Some(last) = self.last {
            self.accum_secs += now.duration_since(last).as_secs_f32();
            self.accum_frames += 1;
            if self.accum_secs >= 0.5 {
                self.display = self.accum_frames as f32 / self.accum_secs;
                self.accum_secs = 0.0;
                self.accum_frames = 0;
            }
        }
        self.last = Some(now);
        self.display
    }
}

pub struct App {
    client: Arc<ClusterClient>,
    notifications: Notifications,
    state: Option<RenderState>,
    fullscreen: bool,
    fps: FpsCounter,
}

impl App {
    pub fn new(client: Arc<ClusterClient>, fullscreen: bool) -> Self {
        Self {
            client,
            notifications: Notifications::new(),
            state: None,
            fullscreen,
            fps: FpsCounter::new(),
        }
    }

    fn init(&mut self, event_loop: &ActiveEventLoop) {
        let screen = self.client.screen();
        let mut attrs = Window::default_attributes()
            .with_title(format!("Massive Ticker Wall ( INDEX: {} )", screen.index))
            .with_inner_size(winit::dpi::LogicalSize::new(
                screen.width.max(1) as f64,
                screen.height.max(1) as f64,
            ));
        if self.fullscreen {
            // Borderless fullscreen on the current monitor (kiosk).
            attrs = attrs.with_fullscreen(Some(winit::window::Fullscreen::Borderless(None)));
        }

        let template = ConfigTemplateBuilder::new().with_alpha_size(8);
        let display_builder = DisplayBuilder::new().with_window_attributes(Some(attrs));
        let (window, gl_config) = display_builder
            .build(event_loop, template, |configs| {
                configs
                    .reduce(|a, b| if b.num_samples() > a.num_samples() { b } else { a })
                    .expect("no GL config")
            })
            .expect("failed to build GL display");

        let window = window.expect("no window created");
        let raw = window
            .window_handle()
            .expect("window handle")
            .as_raw();
        let gl_display = gl_config.display();

        let context_attributes = ContextAttributesBuilder::new().build(Some(raw));
        let not_current = unsafe {
            gl_display
                .create_context(&gl_config, &context_attributes)
                .expect("create GL context")
        };

        let surface_attrs = window
            .build_surface_attributes(Default::default())
            .expect("surface attributes");
        let surface = unsafe {
            gl_display
                .create_window_surface(&gl_config, &surface_attrs)
                .expect("create window surface")
        };
        let context = not_current.make_current(&surface).expect("make current");

        // vsync so we don't spin at thousands of fps.
        let _ = surface.set_swap_interval(
            &context,
            SwapInterval::Wait(NonZeroU32::new(1).unwrap()),
        );

        let renderer =
            unsafe { OpenGl::new_from_function_cstr(|s| gl_display.get_proc_address(s)) }
                .expect("create femtovg renderer");
        let mut canvas = Canvas::new(renderer).expect("create canvas");
        let fonts = Fonts::load(&mut canvas).expect("load fonts");

        self.state = Some(RenderState {
            window,
            surface,
            context,
            canvas,
            fonts,
            last_logical_size: (screen.width, screen.height),
        });
    }

    fn render(&mut self) {
        let Some(state) = self.state.as_mut() else {
            return;
        };

        let physical = state.window.inner_size();
        let dpi = state.window.scale_factor() as f32;
        let (pw, ph) = (physical.width.max(1), physical.height.max(1));
        let logical_w = (pw as f32 / dpi).round() as i32;
        let logical_h = (ph as f32 / dpi).round() as i32;

        // Report size changes to the leader (debounced by "did it change").
        if (logical_w, logical_h) != state.last_logical_size {
            self.client.request_resize(logical_w, logical_h);
            state.last_logical_size = (logical_w, logical_h);
        }

        state.canvas.set_size(pw, ph, dpi);

        // One lock acquisition for all shared state this frame.
        let frame = self.client.render_snapshot();

        // Always measure frame rate (cheap) so the meter is accurate the moment
        // `show_fps` is toggled on.
        let fps = self.fps.tick();

        // Feed any newly-arrived announcements into the manager.
        for announcement in frame.new_announcements {
            self.notifications.add(announcement);
        }

        let settings = frame.cluster.as_ref().and_then(|c| c.settings.as_ref());

        // Background (config color if we have settings, else black). Clear the
        // whole framebuffer in physical pixels, before the DPI transform below.
        let bg = settings
            .map(|s| draw::color_of(&s.bg_color, Color::black()))
            .unwrap_or_else(Color::black);
        state.canvas.clear_rect(0, 0, pw, ph, bg);

        // femtovg draws in physical pixels; our whole layout is in logical
        // (density-independent) units. Scale by the device pixel ratio so logical
        // coordinates fill the physical surface. Without this, everything renders
        // at 1/dpi scale in the top-left on any HiDPI (e.g. Retina) display.
        state.canvas.reset_transform();
        state.canvas.scale(dpi, dpi);

        if let (Some(settings), Some(cluster)) = (settings, frame.cluster.as_ref()) {
            let global = layout::global_offset(
                layout::now_nanos(),
                settings.scroll_speed,
                settings.ticker_box_width,
                frame.tickers.len(),
            );
            let screen_offset = cluster.screen_global_offset(&frame.screen.uuid);

            // Center the primary tape in the space above the movers strip (or the
            // whole window when the strip is hidden), not the whole window.
            let showing_movers = settings.show_movers && !frame.movers.is_empty();
            let content_height = if showing_movers {
                frame.screen.height as f32 - draw::movers_tape_height(frame.screen.height as f32)
            } else {
                frame.screen.height as f32
            };

            draw::render_tickers(
                &mut state.canvas,
                &state.fonts,
                settings,
                content_height,
                &frame.tickers,
                global,
                screen_offset,
                frame.screen.width as f32,
            );

            // Secondary gainers/losers tape at the bottom, on its own speed.
            if showing_movers {
                let movers_global = layout::global_offset(
                    layout::now_nanos(),
                    settings.movers_scroll_speed,
                    draw::MOVERS_BOX_WIDTH,
                    frame.movers.len(),
                );
                draw::render_movers_tape(
                    &mut state.canvas,
                    &state.fonts,
                    settings,
                    &frame.screen,
                    cluster,
                    &frame.movers,
                    movers_global,
                    screen_offset,
                );
            }

            self.notifications
                .render(&mut state.canvas, &state.fonts, settings, cluster, &frame.screen);

            // FPS meter on top of everything (incl. any notification banner).
            if settings.show_fps {
                draw::fps_meter(&mut state.canvas, &state.fonts, frame.screen.height as f32, fps);
            }
        }

        // Overlay the system panel when not connected.
        if frame.status != GrpcStatus::Connected {
            let message = match frame.status {
                GrpcStatus::Reconnecting => "Reconnecting to Leader..",
                GrpcStatus::Disconnected => "Disconnected from Leader..",
                GrpcStatus::Connected => "",
            };
            draw::system_panel(
                &mut state.canvas,
                &state.fonts,
                frame.screen.width as f32,
                &frame.screen,
                message,
            );
        }

        state.canvas.flush_to_output(());
        let _ = state.surface.swap_buffers(&state.context);
    }
}

impl ApplicationHandler for App {
    fn resumed(&mut self, event_loop: &ActiveEventLoop) {
        if self.state.is_none() {
            self.init(event_loop);
        }
    }

    fn window_event(&mut self, event_loop: &ActiveEventLoop, _id: WindowId, event: WindowEvent) {
        match event {
            WindowEvent::CloseRequested => event_loop.exit(),
            WindowEvent::Resized(size) => {
                if let Some(state) = &self.state {
                    if let (Some(w), Some(h)) =
                        (NonZeroU32::new(size.width), NonZeroU32::new(size.height))
                    {
                        state.surface.resize(&state.context, w, h);
                    }
                }
            }
            WindowEvent::RedrawRequested => self.render(),
            _ => {}
        }
    }

    fn about_to_wait(&mut self, _event_loop: &ActiveEventLoop) {
        if let Some(state) = &self.state {
            state.window.request_redraw();
        }
    }
}

/// Log a client error without crashing the render loop.
pub(crate) fn log_client_error(err: anyhow::Error) {
    error!(error = %err, "cluster client exited with error");
}
