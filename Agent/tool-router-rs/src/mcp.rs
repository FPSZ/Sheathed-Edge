use std::{
    collections::HashMap,
    fs,
    path::{Path, PathBuf},
    process::Stdio,
};

use anyhow::{Context, Result, anyhow};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use reqwest::{
    Client,
    header::{AUTHORIZATION, HeaderMap, HeaderName, HeaderValue},
};
use serde_json::{Value, json};
use tokio::{
    io::AsyncWriteExt,
    process::Command,
    time::{Duration, sleep},
};

use crate::{
    config::normalize_runtime_path,
    logging::now_ms,
    models::{
        AppState, McpBridgeProcess, McpDiscoverRequest, McpDiscoverResponse, McpDiscoveredTool,
        McpRuntimeEntry, McpRuntimeState, McpRuntimeStatusResponse, McpServerProfile,
        McpToolCacheEntry, McpToolCacheFile, McpValidateRequest, McpValidateResponse,
    },
};

pub async fn validate_server(
    state: &AppState,
    req: McpValidateRequest,
) -> Result<McpValidateResponse> {
    let server = sanitize_server(req.server)?;
    match server.kind.as_str() {
        "native_streamable_http" => {
            let client = http_client_for_server(&server)?;
            probe_native_streamable_http(&client, &server).await?;
            Ok(McpValidateResponse {
                ok: true,
                summary: format!("native MCP {} is reachable", server.label),
                effective_openwebui_type: "mcp".into(),
                effective_connection_url: server.url.clone(),
                error: None,
            })
        }
        "mcpo_sse" => {
            let client = http_client_for_server(&server)?;
            client
                .get(&server.url)
                .headers(auth_headers(&server)?)
                .send()
                .await
                .with_context(|| format!("connect to SSE MCP server {}", server.url))?;
            Ok(McpValidateResponse {
                ok: true,
                summary: format!("SSE MCP {} is reachable", server.label),
                effective_openwebui_type: "openapi".into(),
                effective_connection_url: runtime_connection_base(state, &server),
                error: None,
            })
        }
        "mcpo_stdio" => {
            if server.command.is_empty() {
                return Err(anyhow!("stdio MCP command is required"));
            }
            if server.workdir.trim().is_empty() {
                return Err(anyhow!("stdio MCP workdir is required"));
            }
            let workdir = Path::new(&server.workdir);
            if !workdir.exists() {
                return Err(anyhow!(
                    "stdio MCP workdir does not exist: {}",
                    server.workdir
                ));
            }
            Ok(McpValidateResponse {
                ok: true,
                summary: format!("stdio MCP {} configuration looks valid", server.label),
                effective_openwebui_type: "openapi".into(),
                effective_connection_url: runtime_connection_base(state, &server),
                error: None,
            })
        }
        other => Err(anyhow!("unsupported mcp kind: {other}")),
    }
}

pub async fn discover_tools(
    state: &AppState,
    req: McpDiscoverRequest,
) -> Result<McpDiscoverResponse> {
    let server_id = req.server_id.trim().to_string();
    if server_id.is_empty() {
        return Err(anyhow!("server_id is required"));
    }
    let servers = load_servers(state)?;
    let server = servers
        .into_iter()
        .find(|item| item.id.eq_ignore_ascii_case(&server_id))
        .ok_or_else(|| anyhow!("unknown mcp server: {}", server_id))?;

    let tools = match server.kind.as_str() {
        "native_streamable_http" => discover_native_tools(&server).await?,
        "mcpo_stdio" | "mcpo_sse" => {
            let runtime = sync_runtime_with_config(state).await?;
            let entry = runtime
                .servers
                .into_iter()
                .find(|item| item.server_id.eq_ignore_ascii_case(&server.id))
                .ok_or_else(|| anyhow!("missing runtime entry for {}", server.id))?;
            if entry.status != "running" {
                return Err(anyhow!(
                    "mcpo bridge for {} is not running: {}",
                    server.id,
                    entry.last_error
                ));
            }
            discover_openapi_tools(&entry.effective_connection_url, &server.id).await?
        }
        other => return Err(anyhow!("unsupported mcp kind: {other}")),
    };

    let now = now_iso_string();
    write_tool_cache(state, &server.id, &tools, &now, "")?;

    Ok(McpDiscoverResponse {
        ok: true,
        summary: format!("discovered {} tools for {}", tools.len(), server.label),
        server_id: server.id.clone(),
        tools,
        last_discovered_at: now,
        effective_openwebui_type: effective_openwebui_type(&server).into(),
        effective_connection_url: effective_connection_url(state, &server).await,
        error: None,
    })
}

pub async fn runtime_status(state: &AppState) -> Result<McpRuntimeStatusResponse> {
    sync_runtime_with_config(state).await
}

fn sanitize_server(mut server: McpServerProfile) -> Result<McpServerProfile> {
    server.id = server.id.trim().to_string();
    server.label = server.label.trim().to_string();
    server.kind = server.kind.trim().to_string();
    server.url = server.url.trim().to_string();
    server.workdir = normalize_runtime_path(server.workdir.trim());
    server.command = server
        .command
        .into_iter()
        .map(|item| item.trim().to_string())
        .filter(|item| !item.is_empty())
        .collect();
    server.disabled_tools = unique_trimmed(server.disabled_tools);
    server.plugin_scope = unique_trimmed(server.plugin_scope);
    server.headers = cleaned_map(server.headers);
    server.env = cleaned_map(server.env);
    server.auth_payload = cleaned_map(server.auth_payload);
    if server.timeout_ms == 0 {
        server.timeout_ms = 30_000;
    }
    if server.auth_type.trim().is_empty() {
        server.auth_type = "none".into();
    }
    if server.id.is_empty() {
        return Err(anyhow!("mcp server id is required"));
    }
    if server.label.is_empty() {
        return Err(anyhow!("mcp server label is required"));
    }
    Ok(server)
}

fn load_servers(state: &AppState) -> Result<Vec<McpServerProfile>> {
    if state.mcp_servers_path.trim().is_empty() {
        return Ok(vec![]);
    }
    let data = match fs::read_to_string(&state.mcp_servers_path) {
        Ok(data) => data,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(vec![]),
        Err(err) => return Err(err).context("read mcp servers"),
    };
    let servers: Vec<McpServerProfile> =
        serde_json::from_str(&data).context("parse mcp servers")?;
    servers.into_iter().map(sanitize_server).collect()
}

fn read_tool_cache(state: &AppState) -> Result<McpToolCacheFile> {
    if state.mcp_tool_cache_path.trim().is_empty() {
        return Ok(McpToolCacheFile::default());
    }
    let data = match fs::read_to_string(&state.mcp_tool_cache_path) {
        Ok(data) => data,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => {
            return Ok(McpToolCacheFile::default());
        }
        Err(err) => return Err(err).context("read mcp tool cache"),
    };
    serde_json::from_str(&data).context("parse mcp tool cache")
}

fn write_tool_cache(
    state: &AppState,
    server_id: &str,
    tools: &[McpDiscoveredTool],
    last_discovered_at: &str,
    last_error: &str,
) -> Result<()> {
    if state.mcp_tool_cache_path.trim().is_empty() {
        return Ok(());
    }
    let mut payload = read_tool_cache(state)?;
    let mut updated = false;
    for entry in &mut payload.servers {
        if entry.server_id.eq_ignore_ascii_case(server_id) {
            entry.tools = tools.to_vec();
            entry.last_discovered_at = last_discovered_at.to_string();
            entry.last_error = last_error.to_string();
            updated = true;
            break;
        }
    }
    if !updated {
        payload.servers.push(McpToolCacheEntry {
            server_id: server_id.to_string(),
            tools: tools.to_vec(),
            last_discovered_at: last_discovered_at.to_string(),
            last_error: last_error.to_string(),
        });
    }
    if let Some(parent) = Path::new(&state.mcp_tool_cache_path).parent() {
        fs::create_dir_all(parent).ok();
    }
    let mut data = serde_json::to_vec_pretty(&payload).context("encode mcp tool cache")?;
    data.push(b'\n');
    fs::write(&state.mcp_tool_cache_path, data).context("write mcp tool cache")
}

async fn sync_runtime_with_config(state: &AppState) -> Result<McpRuntimeStatusResponse> {
    let servers = load_servers(state)?;
    let signature = serde_json::to_string(&servers).unwrap_or_default();

    let mut runtime = state.mcp_runtime.lock().await;
    if runtime.signature != signature {
        stop_all_bridges(&mut runtime).await;
        runtime.signature = signature;
        start_enabled_bridges(state, &servers, &mut runtime).await?;
    } else {
        refresh_runtime_processes(&mut runtime).await;
    }

    let mut entries = Vec::with_capacity(servers.len());
    for server in servers {
        if server.kind == "native_streamable_http" {
            entries.push(McpRuntimeEntry {
                server_id: server.id.clone(),
                label: server.label.clone(),
                enabled: server.enabled,
                kind: server.kind.clone(),
                status: if server.enabled { "native" } else { "disabled" }.into(),
                bridge_port: 0,
                process_pid: 0,
                effective_openwebui_type: "mcp".into(),
                effective_connection_url: server.url.clone(),
                last_error: String::new(),
            });
            continue;
        }

        let process = runtime
            .bridges
            .iter()
            .find(|item| item.server_id.eq_ignore_ascii_case(&server.id));
        if let Some(process) = process {
            entries.push(McpRuntimeEntry {
                server_id: server.id.clone(),
                label: server.label.clone(),
                enabled: server.enabled,
                kind: server.kind.clone(),
                status: if process.last_error.is_empty() {
                    "running".into()
                } else {
                    "error".into()
                },
                bridge_port: process.port,
                process_pid: process.pid,
                effective_openwebui_type: "openapi".into(),
                effective_connection_url: format!(
                    "http://{}:{}",
                    state.config.mcp.bridge_host, process.port
                ),
                last_error: process.last_error.clone(),
            });
        } else {
            entries.push(McpRuntimeEntry {
                server_id: server.id.clone(),
                label: server.label.clone(),
                enabled: server.enabled,
                kind: server.kind.clone(),
                status: if server.enabled {
                    "stopped".into()
                } else {
                    "disabled".into()
                },
                bridge_port: 0,
                process_pid: 0,
                effective_openwebui_type: "openapi".into(),
                effective_connection_url: runtime_connection_base(state, &server),
                last_error: String::new(),
            });
        }
    }

    Ok(McpRuntimeStatusResponse { servers: entries })
}

async fn stop_all_bridges(runtime: &mut McpRuntimeState) {
    for bridge in &mut runtime.bridges {
        if let Some(child) = &mut bridge.child {
            let _ = child.kill().await;
        }
    }
    runtime.bridges.clear();
}

async fn refresh_runtime_processes(runtime: &mut McpRuntimeState) {
    for bridge in &mut runtime.bridges {
        if let Some(child) = &mut bridge.child {
            match child.try_wait() {
                Ok(Some(status)) => {
                    bridge.last_error = format!("mcpo exited with status {status}");
                }
                Ok(None) => {}
                Err(err) => {
                    bridge.last_error = err.to_string();
                }
            }
        }
    }
}

async fn start_enabled_bridges(
    state: &AppState,
    servers: &[McpServerProfile],
    runtime: &mut McpRuntimeState,
) -> Result<()> {
    let enabled: Vec<McpServerProfile> = servers
        .iter()
        .filter(|item| item.enabled && item.kind != "native_streamable_http")
        .cloned()
        .collect();
    for (index, server) in enabled.iter().enumerate() {
        let port = state
            .config
            .mcp
            .bridge_port_start
            .saturating_add(index as u16);
        if port > state.config.mcp.bridge_port_end {
            return Err(anyhow!("mcp bridge port range exhausted"));
        }
        let config_path = write_mcpo_server_config(state, server)?;
        let log_path = mcpo_log_path(state, server);
        let mut command = Command::new("mcpo");
        command
            .arg("--config")
            .arg(&config_path)
            .arg("--port")
            .arg(port.to_string());
        command.stdout(Stdio::piped()).stderr(Stdio::piped());
        let child_result = command.spawn();
        let mut process = McpBridgeProcess {
            server_id: server.id.clone(),
            kind: server.kind.clone(),
            port,
            config_path,
            pid: 0,
            last_error: String::new(),
            child: None,
        };
        match child_result {
            Ok(mut child) => {
                process.pid = child.id().unwrap_or_default();
                pipe_child_logs(&mut child, &log_path).await;
                process.child = Some(child);
                sleep(Duration::from_millis(250)).await;
            }
            Err(err) => {
                process.last_error = format!("spawn mcpo failed: {err}");
            }
        }
        runtime.bridges.push(process);
    }
    Ok(())
}

async fn pipe_child_logs(child: &mut tokio::process::Child, log_path: &str) {
    let stdout = child.stdout.take();
    let stderr = child.stderr.take();
    if let Some(parent) = Path::new(log_path).parent() {
        let _ = fs::create_dir_all(parent);
    }
    if let Some(mut stream) = stdout {
        let path = log_path.to_string();
        tokio::spawn(async move {
            if let Ok(mut file) = tokio::fs::OpenOptions::new()
                .create(true)
                .append(true)
                .open(&path)
                .await
            {
                let _ = tokio::io::copy(&mut stream, &mut file).await;
                let _ = file.flush().await;
            }
        });
    }
    if let Some(mut stream) = stderr {
        let path = log_path.to_string();
        tokio::spawn(async move {
            if let Ok(mut file) = tokio::fs::OpenOptions::new()
                .create(true)
                .append(true)
                .open(&path)
                .await
            {
                let _ = tokio::io::copy(&mut stream, &mut file).await;
                let _ = file.flush().await;
            }
        });
    }
}

fn write_mcpo_server_config(state: &AppState, server: &McpServerProfile) -> Result<String> {
    let root = PathBuf::from(&state.config.mcp.process_log_dir).join("mcpo");
    fs::create_dir_all(&root).ok();
    let path = root.join(format!("{}.json", server.id));
    let payload = match server.kind.as_str() {
        "mcpo_stdio" => {
            let command = server.command.first().cloned().unwrap_or_default();
            let args: Vec<String> = server.command.iter().skip(1).cloned().collect();
            json!({
                "mcpServers": {
                    server.id.clone(): {
                        "transport": "stdio",
                        "command": command,
                        "args": args,
                        "cwd": server.workdir,
                        "env": server.env,
                        "disabledTools": server.disabled_tools
                    }
                }
            })
        }
        "mcpo_sse" => {
            json!({
                "mcpServers": {
                    server.id.clone(): {
                        "transport": "sse",
                        "url": server.url,
                        "headers": merge_auth_headers(server)?,
                        "disabledTools": server.disabled_tools
                    }
                }
            })
        }
        _ => return Err(anyhow!("unsupported mcpo config kind: {}", server.kind)),
    };
    let mut data = serde_json::to_vec_pretty(&payload).context("encode mcpo config")?;
    data.push(b'\n');
    fs::write(&path, data).context("write mcpo config")?;
    Ok(path.to_string_lossy().replace('\\', "/"))
}

async fn discover_native_tools(server: &McpServerProfile) -> Result<Vec<McpDiscoveredTool>> {
    let client = http_client_for_server(server)?;
    let mut headers = auth_headers(server)?;
    headers.insert(
        reqwest::header::CONTENT_TYPE,
        HeaderValue::from_static("application/json"),
    );

    let initialize = json!({
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2025-03-26",
            "capabilities": {},
            "clientInfo": { "name": "awdp-tool-router", "version": "1.0.0" }
        }
    });
    let _ = client
        .post(&server.url)
        .headers(headers.clone())
        .json(&initialize)
        .send()
        .await
        .context("initialize native MCP")?;

    let list_tools = json!({
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/list",
        "params": {}
    });
    let value: Value = client
        .post(&server.url)
        .headers(headers)
        .json(&list_tools)
        .send()
        .await
        .context("list native MCP tools")?
        .json()
        .await
        .context("parse native MCP tools list")?;

    let tools = value
        .pointer("/result/tools")
        .and_then(Value::as_array)
        .cloned()
        .unwrap_or_default();
    Ok(tools
        .into_iter()
        .filter_map(|item| {
            let name = item.get("name")?.as_str()?.trim().to_string();
            if name.is_empty() {
                return None;
            }
            Some(McpDiscoveredTool {
                name,
                description: item
                    .get("description")
                    .and_then(Value::as_str)
                    .unwrap_or_default()
                    .trim()
                    .to_string(),
                disabled: false,
            })
        })
        .collect())
}

async fn discover_openapi_tools(base_url: &str, server_id: &str) -> Result<Vec<McpDiscoveredTool>> {
    let client = Client::builder()
        .danger_accept_invalid_certs(true)
        .timeout(Duration::from_secs(10))
        .build()
        .context("build openapi discovery client")?;

    let urls = [
        format!("{}/openapi.json", base_url.trim_end_matches('/')),
        format!(
            "{}/{}/openapi.json",
            base_url.trim_end_matches('/'),
            server_id
        ),
    ];
    let mut spec: Option<Value> = None;
    for url in urls {
        if let Ok(resp) = client.get(&url).send().await {
            if resp.status().is_success() {
                if let Ok(value) = resp.json::<Value>().await {
                    spec = Some(value);
                    break;
                }
            }
        }
    }
    let spec = spec.ok_or_else(|| anyhow!("unable to fetch mcpo openapi spec"))?;
    let mut tools = Vec::new();
    if let Some(paths) = spec.get("paths").and_then(Value::as_object) {
        for (path, item) in paths {
            if path.ends_with("/openapi.json") || path.ends_with("/docs") {
                continue;
            }
            let post = item.get("post").or_else(|| item.get("get"));
            let operation_id = post
                .and_then(|entry| entry.get("operationId"))
                .and_then(Value::as_str)
                .unwrap_or_default()
                .trim()
                .to_string();
            let summary = post
                .and_then(|entry| entry.get("summary"))
                .and_then(Value::as_str)
                .unwrap_or_default()
                .trim()
                .to_string();
            let name = if !operation_id.is_empty() {
                operation_id
            } else {
                path.trim_matches('/')
                    .rsplit('/')
                    .next()
                    .unwrap_or(path)
                    .to_string()
            };
            if name.is_empty() {
                continue;
            }
            tools.push(McpDiscoveredTool {
                name,
                description: summary,
                disabled: false,
            });
        }
    }
    tools.sort_by(|a, b| a.name.cmp(&b.name));
    tools.dedup_by(|a, b| a.name.eq_ignore_ascii_case(&b.name));
    Ok(tools)
}

async fn probe_native_streamable_http(client: &Client, server: &McpServerProfile) -> Result<()> {
    let value = json!({
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2025-03-26",
            "capabilities": {},
            "clientInfo": { "name": "awdp-tool-router", "version": "1.0.0" }
        }
    });
    client
        .post(&server.url)
        .headers(auth_headers(server)?)
        .json(&value)
        .send()
        .await
        .with_context(|| format!("probe native MCP {}", server.url))?;
    Ok(())
}

fn http_client_for_server(server: &McpServerProfile) -> Result<Client> {
    Client::builder()
        .danger_accept_invalid_certs(!server.verify_tls)
        .timeout(Duration::from_millis(server.timeout_ms.max(1_000)))
        .build()
        .context("build mcp client")
}

fn auth_headers(server: &McpServerProfile) -> Result<HeaderMap> {
    let mut headers = HeaderMap::new();
    for (key, value) in &server.headers {
        headers.insert(
            HeaderName::from_bytes(key.as_bytes()).context("invalid header name")?,
            HeaderValue::from_str(value).context("invalid header value")?,
        );
    }
    match server.auth_type.as_str() {
        "none" | "" => {}
        "bearer" => {
            if let Some(token) = server.auth_payload.get("token") {
                headers.insert(
                    AUTHORIZATION,
                    HeaderValue::from_str(&format!("Bearer {}", token.trim()))
                        .context("invalid bearer token")?,
                );
            }
        }
        "basic" => {
            let user = server
                .auth_payload
                .get("username")
                .cloned()
                .unwrap_or_default();
            let pass = server
                .auth_payload
                .get("password")
                .cloned()
                .unwrap_or_default();
            let encoded = STANDARD.encode(format!("{user}:{pass}"));
            headers.insert(
                AUTHORIZATION,
                HeaderValue::from_str(&format!("Basic {encoded}")).context("invalid basic auth")?,
            );
        }
        "header" => {
            if let (Some(name), Some(value)) = (
                server.auth_payload.get("name"),
                server.auth_payload.get("value"),
            ) {
                headers.insert(
                    HeaderName::from_bytes(name.as_bytes())
                        .context("invalid custom auth header")?,
                    HeaderValue::from_str(value).context("invalid custom auth header value")?,
                );
            }
        }
        _ => {}
    }
    Ok(headers)
}

fn merge_auth_headers(server: &McpServerProfile) -> Result<HashMap<String, String>> {
    let mut merged = server.headers.clone();
    match server.auth_type.as_str() {
        "bearer" => {
            if let Some(token) = server.auth_payload.get("token") {
                merged.insert("Authorization".into(), format!("Bearer {}", token.trim()));
            }
        }
        "basic" => {
            let user = server
                .auth_payload
                .get("username")
                .cloned()
                .unwrap_or_default();
            let pass = server
                .auth_payload
                .get("password")
                .cloned()
                .unwrap_or_default();
            merged.insert(
                "Authorization".into(),
                format!("Basic {}", STANDARD.encode(format!("{user}:{pass}"))),
            );
        }
        "header" => {
            if let (Some(name), Some(value)) = (
                server.auth_payload.get("name"),
                server.auth_payload.get("value"),
            ) {
                merged.insert(name.trim().into(), value.trim().into());
            }
        }
        "none" | "" => {}
        other => return Err(anyhow!("unsupported auth type: {}", other)),
    }
    Ok(merged)
}

fn unique_trimmed(items: Vec<String>) -> Vec<String> {
    let mut out = Vec::new();
    for item in items {
        let value = item.trim();
        if value.is_empty() {
            continue;
        }
        if out
            .iter()
            .any(|existing: &String| existing.eq_ignore_ascii_case(value))
        {
            continue;
        }
        out.push(value.to_string());
    }
    out
}

fn cleaned_map(map: HashMap<String, String>) -> HashMap<String, String> {
    map.into_iter()
        .filter_map(|(key, value)| {
            let key = key.trim().to_string();
            let value = value.trim().to_string();
            if key.is_empty() || value.is_empty() {
                return None;
            }
            Some((key, value))
        })
        .collect()
}

fn runtime_connection_base(state: &AppState, server: &McpServerProfile) -> String {
    let enabled: Vec<String> = load_servers(state)
        .unwrap_or_default()
        .into_iter()
        .filter(|item| item.enabled && item.kind != "native_streamable_http")
        .map(|item| item.id)
        .collect();
    let index = enabled
        .iter()
        .position(|item| item.eq_ignore_ascii_case(&server.id))
        .unwrap_or(0);
    let port = state
        .config
        .mcp
        .bridge_port_start
        .saturating_add(index as u16);
    format!("http://{}:{}", state.config.mcp.bridge_host, port)
}

async fn effective_connection_url(state: &AppState, server: &McpServerProfile) -> String {
    match server.kind.as_str() {
        "native_streamable_http" => server.url.clone(),
        _ => runtime_connection_base(state, server),
    }
}

fn effective_openwebui_type(server: &McpServerProfile) -> &'static str {
    match server.kind.as_str() {
        "native_streamable_http" => "mcp",
        _ => "openapi",
    }
}

fn mcpo_log_path(state: &AppState, server: &McpServerProfile) -> String {
    PathBuf::from(&state.config.mcp.process_log_dir)
        .join(format!("{}.log", server.id))
        .to_string_lossy()
        .replace('\\', "/")
}

fn now_iso_string() -> String {
    now_ms().to_string()
}
