//! Embedded Roboto fonts, registered into the femtovg canvas. Reuses the same
//! TTF assets as the Go app (repo-root `fonts/`), embedded at compile time.

use anyhow::{anyhow, Result};
use femtovg::{Canvas, FontId, Renderer};

/// Handles to the font weights used by the renderer (light for body text, bold
/// for symbols/prices/announcements — matching the Go app's usage).
#[derive(Clone, Copy)]
pub struct Fonts {
    pub light: FontId,
    pub bold: FontId,
}

const LIGHT: &[u8] = include_bytes!("../../../fonts/Roboto-Light.ttf");
const BOLD: &[u8] = include_bytes!("../../../fonts/Roboto-Bold.ttf");

impl Fonts {
    pub fn load<T: Renderer>(canvas: &mut Canvas<T>) -> Result<Self> {
        let light = canvas
            .add_font_mem(LIGHT)
            .map_err(|e| anyhow!("load light font: {e:?}"))?;
        let bold = canvas
            .add_font_mem(BOLD)
            .map_err(|e| anyhow!("load bold font: {e:?}"))?;
        Ok(Self { light, bold })
    }
}
