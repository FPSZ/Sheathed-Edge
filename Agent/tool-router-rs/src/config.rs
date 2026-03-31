use std::fs;

use anyhow::{Context, Result};

use crate::models::Config;

pub fn load_config(path: &str) -> Result<Config> {
    let data = fs::read_to_string(path).with_context(|| format!("read config {path}"))?;
    let mut cfg: Config = serde_json::from_str(&data).context("parse tool-router config")?;
    cfg.registry_path = normalize_runtime_path(&cfg.registry_path);
    cfg.user_settings_path = normalize_runtime_path(&cfg.user_settings_path);
    cfg.logs.tool_log_dir = normalize_runtime_path(&cfg.logs.tool_log_dir);
    cfg.allowed_paths = cfg
        .allowed_paths
        .iter()
        .map(|p| normalize_runtime_path(p))
        .collect();
    cfg.mcp.servers_path = normalize_runtime_path(&cfg.mcp.servers_path);
    cfg.mcp.tool_cache_path = normalize_runtime_path(&cfg.mcp.tool_cache_path);
    cfg.mcp.process_log_dir = normalize_runtime_path(&cfg.mcp.process_log_dir);
    if cfg.mcp.bridge_host.trim().is_empty() {
        cfg.mcp.bridge_host = "127.0.0.1".into();
    }
    if cfg.mcp.bridge_port_start == 0 {
        cfg.mcp.bridge_port_start = 8092;
    }
    if cfg.mcp.bridge_port_end < cfg.mcp.bridge_port_start {
        cfg.mcp.bridge_port_end = cfg.mcp.bridge_port_start.saturating_add(32);
    }
    Ok(cfg)
}

pub fn normalize_runtime_path(path: &str) -> String {
    if cfg!(target_os = "linux") && path.len() >= 3 && path.as_bytes()[1] == b':' {
        let drive = path[0..1].to_ascii_lowercase();
        let rest = path[3..].replace('\\', "/");
        return format!("/mnt/{drive}/{rest}");
    }
    path.replace('\\', "/")
}
