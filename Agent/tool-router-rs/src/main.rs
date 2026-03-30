#![allow(dead_code)]
#![recursion_limit = "512"]

mod api;
mod config;
mod executor;
mod logging;
mod mcp;
mod models;
mod policy;
mod registry;
mod ssh;

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
        workspace_root: std::env::current_dir()?
            .to_string_lossy()
            .replace('\\', "/"),
        ssh_hosts_path: std::path::Path::new(&cli.config)
            .parent()
            .unwrap_or(std::path::Path::new("."))
            .join("ssh-hosts.json")
            .to_string_lossy()
            .replace('\\', "/"),
        mcp_servers_path: if config.mcp.servers_path.trim().is_empty() {
            std::path::Path::new(&cli.config)
                .parent()
                .unwrap_or(std::path::Path::new("."))
                .join("mcp-servers.json")
                .to_string_lossy()
                .replace('\\', "/")
        } else {
            config.mcp.servers_path.clone()
        },
        mcp_tool_cache_path: if config.mcp.tool_cache_path.trim().is_empty() {
            std::path::Path::new(&cli.config)
                .parent()
                .unwrap_or(std::path::Path::new("."))
                .join("mcp-tool-cache.json")
                .to_string_lossy()
                .replace('\\', "/")
        } else {
            config.mcp.tool_cache_path.clone()
        },
        mcp_runtime: std::sync::Arc::new(tokio::sync::Mutex::new(
            models::McpRuntimeState::default(),
        )),
        resources_count,
        prompts_count,
    };

    let app = Router::new()
        .route("/healthz", get(api::healthz))
        .route("/openapi.json", get(api::openapi_spec))
        .route("/api/tools/terminal", post(api::openapi_terminal))
        .route("/internal/ssh/test", post(api::test_ssh_host))
        .route("/internal/mcp/validate", post(api::validate_mcp_server))
        .route(
            "/internal/mcp/discover-tools",
            post(api::discover_mcp_tools),
        )
        .route("/internal/mcp/runtime", get(api::mcp_runtime))
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
