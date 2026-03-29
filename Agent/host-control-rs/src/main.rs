mod api;
mod config;
mod llama;
mod models;

use std::sync::Arc;

use anyhow::Result;
use axum::{
    Router,
    routing::{get, post},
};
use clap::Parser;
use models::{RuntimeState, SharedState};
use tokio::{net::TcpListener, sync::Mutex};

#[derive(Parser, Debug)]
struct Cli {
    #[arg(long, default_value = "../host-control.config.json")]
    config: String,
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    let cfg = config::load_config(&cli.config)?;
    let llama = config::load_llama_config(&cfg.llama_server_config_path)?;
    let profiles = config::load_profiles(&cfg.model_profiles_path)?;
    config::validate_default_profile(&profiles, &cfg.default_profile_id)?;

    let state: SharedState = Arc::new(RuntimeState {
        llama,
        profiles: Mutex::new(profiles),
        http_client: reqwest::Client::builder()
            .timeout(std::time::Duration::from_millis(cfg.timeout_ms))
            .build()?,
        active_profile_id: Mutex::new(cfg.default_profile_id.clone()),
        process: Mutex::new(None),
        config: cfg.clone(),
    });

    let app = Router::new()
        .route("/healthz", get(api::healthz))
        .route("/internal/host/llama/status", get(api::llama_status))
        .route("/internal/host/llama/start", post(api::llama_start))
        .route("/internal/host/llama/stop", post(api::llama_stop))
        .route("/internal/host/llama/restart", post(api::llama_restart))
        .route("/internal/host/llama/switch", post(api::llama_switch))
        .route("/internal/host/profiles/update", post(api::profile_update))
        .with_state(state);

    let listener = TcpListener::bind(format!("{}:{}", cfg.listen_host, cfg.listen_port)).await?;
    axum::serve(listener, app).await?;
    Ok(())
}
