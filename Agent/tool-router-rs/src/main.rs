#![allow(dead_code)]

mod api;
mod config;
mod executor;
mod logging;
mod models;
mod policy;
mod registry;

use anyhow::Result;
use axum::{
    Router,
    routing::{get, post},
};
use clap::Parser;
use models::AppState;
use tokio::net::TcpListener;
use tower_http::cors::{Any, CorsLayer};

#[derive(Parser, Debug)]
struct Cli {
    #[arg(long, default_value = "../tool-router.config.json")]
    config: String,
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    let config = config::load_config(&cli.config)?;
    let registry = registry::load_registry(&config.registry_path)?;
    let resources_count = registry.resources.len();
    let prompts_count = registry.prompts.len();
    let tools = registry::build_tool_map(registry)?;

    std::fs::create_dir_all(&config.logs.tool_log_dir)?;

    let state = AppState {
        config: config.clone(),
        tools: std::sync::Arc::new(tools),
        workspace_root: std::env::current_dir()?.to_string_lossy().replace('\\', "/"),
        resources_count,
        prompts_count,
    };

    let app = Router::new()
        .route("/healthz", get(api::healthz))
        .route("/openapi.json", get(api::openapi_spec))
        .route("/api/tools/terminal", post(api::openapi_terminal))
        .route("/internal/tools/resolve", post(api::resolve_tool))
        .route("/internal/tools/execute", post(api::execute_tool))
        .layer(
            CorsLayer::new()
                .allow_origin(Any)
                .allow_methods(Any)
                .allow_headers(Any),
        )
        .with_state(state);

    let listener =
        TcpListener::bind(format!("{}:{}", config.listen_host, config.listen_port)).await?;
    axum::serve(listener, app).await?;
    Ok(())
}
