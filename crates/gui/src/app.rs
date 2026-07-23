//! winit `ApplicationHandler` + glutin OpenGL context + the per-frame render loop.
//! The GUI main thread owns rendering; a background tokio task (the client) keeps
//! shared state fresh, which this loop reads every frame.

use std::num::NonZeroU32;
use std::sync::Arc;

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

pub struct App {
    client: Arc<ClusterClient>,
    notifications: Notifications,
    state: Option<RenderState>,
    fullscreen: bool,
}

impl App {
    pub fn new(client: Arc<ClusterClient>, fullscreen: bool) -> Self {
        Self {
            client,
            notifications: Notifications::new(),
            state: None,
            fullscreen,
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

        let screen = self.client.screen();
        let settings = self.client.settings();
        let cluster = self.client.cluster();

        // Background (config color if we have settings, else black).
        let bg = settings
            .as_ref()
            .map(|s| draw::color_of(&s.bg_color, Color::black()))
            .unwrap_or_else(Color::black);
        state.canvas.clear_rect(0, 0, pw, ph, bg);

        if let (Some(settings), Some(cluster)) = (settings.as_ref(), cluster.as_ref()) {
            let tickers = self.client.tickers();
            let global = layout::global_offset(
                layout::now_nanos(),
                settings.scroll_speed,
                settings.ticker_box_width,
                tickers.len(),
            );
            let screen_offset = cluster.screen_global_offset(&screen.uuid);

            draw::render_tickers(
                &mut state.canvas,
                &state.fonts,
                settings,
                &screen,
                &tickers,
                global,
                screen_offset,
                screen.width as f32,
            );

            // Drain and render announcements.
            for announcement in self.client.drain_announcements() {
                self.notifications.add(announcement);
            }
            self.notifications
                .render(&mut state.canvas, &state.fonts, settings, cluster, &screen);
        }

        // Overlay the system panel when not connected.
        if self.client.status() != GrpcStatus::Connected {
            let message = match self.client.status() {
                GrpcStatus::Reconnecting => "Reconnecting to Leader..",
                GrpcStatus::Disconnected => "Disconnected from Leader..",
                GrpcStatus::Connected => "",
            };
            state.canvas.set_size(pw, ph, dpi);
            draw::system_panel(&mut state.canvas, &state.fonts, screen.width as f32, &screen, message);
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
