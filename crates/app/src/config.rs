//! Optional config-file layer. Values here sit *below* environment variables and
//! CLI flags in precedence (flags > env > file > built-in defaults), mirroring the
//! Go app's viper setup. The file is named `tickerwall.{toml,yaml,yml,json}` and is
//! searched for in the current directory, then the user's home directory.

use std::path::PathBuf;

use serde::Deserialize;
use tracing::warn;

/// All fields optional: a config file only supplies values the user chose to set.
/// Field names match the CLI flag names (snake_case).
#[derive(Debug, Default, Deserialize)]
#[serde(default)]
pub struct FileConfig {
    // server
    pub api_key: Option<String>,
    pub tickers: Option<String>,
    pub grpc_port: Option<u16>,
    pub scroll_speed: Option<i32>,
    pub ticker_box_width: Option<i32>,
    pub animation_duration: Option<i32>,
    pub per_tick_updates: Option<bool>,
    pub show_fps: Option<bool>,
    pub show_logos: Option<bool>,
    pub up_color: Option<String>,
    pub down_color: Option<String>,
    pub bg_color: Option<String>,
    pub font_color: Option<String>,
    pub ticker_bg_color: Option<String>,
    // gui
    pub leader: Option<String>,
    pub screen_width: Option<i32>,
    pub screen_height: Option<i32>,
    pub screen_index: Option<i32>,
    pub fullscreen: Option<bool>,
}

const BASENAME: &str = "tickerwall";
const EXTENSIONS: &[&str] = &["toml", "yaml", "yml", "json"];

impl FileConfig {
    /// Load the first `tickerwall.*` config found in the cwd or home dir. Returns
    /// an empty config (with a warning) if none is found or one fails to parse.
    pub fn load() -> Self {
        let Some(path) = Self::find() else {
            return Self::default();
        };
        match Self::parse(&path) {
            Ok(cfg) => {
                tracing::info!(path = %path.display(), "loaded config file");
                cfg
            }
            Err(e) => {
                warn!(path = %path.display(), error = %e, "ignoring unreadable config file");
                Self::default()
            }
        }
    }

    fn search_dirs() -> Vec<PathBuf> {
        let mut dirs = vec![PathBuf::from(".")];
        if let Some(home) = std::env::var_os("HOME") {
            dirs.push(PathBuf::from(home));
        }
        dirs
    }

    fn find() -> Option<PathBuf> {
        for dir in Self::search_dirs() {
            for ext in EXTENSIONS {
                let candidate = dir.join(format!("{BASENAME}.{ext}"));
                if candidate.is_file() {
                    return Some(candidate);
                }
            }
        }
        None
    }

    fn parse(path: &PathBuf) -> anyhow::Result<Self> {
        let text = std::fs::read_to_string(path)?;
        let ext = path
            .extension()
            .and_then(|e| e.to_str())
            .unwrap_or("")
            .to_lowercase();
        let cfg = match ext.as_str() {
            "json" => serde_json::from_str(&text)?,
            "yaml" | "yml" => serde_yaml::from_str(&text)?,
            _ => toml::from_str(&text)?, // default to TOML
        };
        Ok(cfg)
    }
}
