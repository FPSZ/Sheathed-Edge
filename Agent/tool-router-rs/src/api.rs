use axum::{Json, extract::State, http::StatusCode, response::IntoResponse};
use serde_json::json;

use crate::{
    executor,
    logging::{append_tool_log, now_ms},
    mcp,
    models::{
        AppState, ErrorEnvelope, ExecuteRequest, ExecuteResponse, McpDiscoverRequest,
        McpValidateRequest, OpenAPITerminalRequest, OpenAPITerminalResponse, ResolveRequest,
        ResolveResponse, SshTestRequest,
    },
    policy::{check_tool_access, normalize_tool_arguments},
    ssh,
};

pub async fn healthz(State(state): State<AppState>) -> impl IntoResponse {
    Json(json!({
        "status": "ok",
        "tool_count": state.tools.len(),
        "resources_count": state.resources_count,
        "prompts_count": state.prompts_count
    }))
}

pub async fn openapi_spec(State(state): State<AppState>) -> impl IntoResponse {
    let server_url = format!(
        "http://{}:{}",
        state.config.listen_host, state.config.listen_port
    );
    Json(json!({
        "openapi": "3.1.0",
        "info": {
            "title": "AWDP Terminal Tool Server",
            "version": "1.0.0",
            "description": "Controlled local and SSH terminal execution for Open WebUI via OpenAPI."
        },
        "servers": [
            {
                "url": server_url
            }
        ],
        "paths": {
            "/api/tools/terminal": {
                "post": {
                    "operationId": "runTerminal",
                    "summary": "Run a controlled local or SSH shell command",
                    "requestBody": {
                        "required": true,
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "object",
                                    "additionalProperties": false,
                                    "required": ["command"],
                                    "properties": {
                                        "command": { "type": "string", "minLength": 1 },
                                        "transport": {
                                            "anyOf": [
                                                {
                                                    "type": "string",
                                                    "enum": ["local", "ssh"]
                                                },
                                                {
                                                    "type": "null"
                                                }
                                            ],
                                            "default": "local"
                                        },
                                        "shell": {
                                            "anyOf": [
                                                {
                                                    "type": "string",
                                                    "enum": ["powershell", "wsl-bash"]
                                                },
                                                {
                                                    "type": "null"
                                                }
                                            ],
                                            "default": "powershell"
                                        },
                                        "host_id": {
                                            "anyOf": [
                                                { "type": "string", "minLength": 1 },
                                                { "type": "null" }
                                            ]
                                        },
                                        "remote_shell": {
                                            "anyOf": [
                                                {
                                                    "type": "string",
                                                    "enum": ["bash", "powershell"]
                                                },
                                                { "type": "null" }
                                            ]
                                        },
                                        "workdir": {
                                            "anyOf": [
                                                { "type": "string", "minLength": 1 },
                                                { "type": "null" }
                                            ]
                                        },
                                        "timeout_ms": {
                                            "anyOf": [
                                                { "type": "integer", "minimum": 1 },
                                                { "type": "null" }
                                            ]
                                        },
                                        "user_email": {
                                            "anyOf": [
                                                { "type": "string", "minLength": 3 },
                                                { "type": "null" }
                                            ]
                                        }
                                    }
                                }
                            }
                        }
                    },
                    "responses": {
                        "200": {
                            "description": "Terminal command result",
                            "content": {
                                "application/json": {
                                    "schema": {
                                        "type": "object",
                                        "required": [
                                            "ok",
                                            "summary",
                                            "exit_code",
                                            "stdout",
                                            "stderr",
                                            "timed_out",
                                            "duration_ms",
                                            "transport",
                                            "host_id",
                                            "remote_shell",
                                            "shell",
                                            "workdir",
                                            "truncated"
                                        ],
                                        "properties": {
                                            "ok": { "type": "boolean" },
                                            "summary": { "type": "string" },
                                            "exit_code": { "type": "integer" },
                                            "stdout": { "type": "string" },
                                            "stderr": { "type": "string" },
                                            "timed_out": { "type": "boolean" },
                                            "duration_ms": { "type": "integer" },
                                            "transport": { "type": "string" },
                                            "host_id": { "type": "string" },
                                            "remote_shell": { "type": "string" },
                                            "shell": { "type": "string" },
                                            "workdir": { "type": "string" },
                                            "truncated": { "type": "boolean" },
                                            "error": {
                                                "type": "object",
                                                "nullable": true,
                                                "properties": {
                                                    "code": { "type": "string" },
                                                    "message": { "type": "string" }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }))
}

pub async fn test_ssh_host(
    State(state): State<AppState>,
    Json(req): Json<SshTestRequest>,
) -> impl IntoResponse {
    let timeout_ms = req.timeout_ms.unwrap_or(10_000).clamp(1, 180_000);
    match ssh::test_ssh_host_request(&state, &req, timeout_ms).await {
        Ok(resp) => (StatusCode::OK, Json(resp)).into_response(),
        Err(err) => (
            StatusCode::OK,
            Json(crate::models::SshTestResponse {
                ok: false,
                summary: err.to_string(),
                host_key_status: "unknown".into(),
                host_key_fingerprint: String::new(),
                error: Some(ErrorEnvelope {
                    code: "ssh_test_failed".into(),
                    message: err.to_string(),
                }),
            }),
        )
            .into_response(),
    }
}

pub async fn validate_mcp_server(
    State(state): State<AppState>,
    Json(req): Json<McpValidateRequest>,
) -> impl IntoResponse {
    match mcp::validate_server(&state, req).await {
        Ok(resp) => (StatusCode::OK, Json(resp)).into_response(),
        Err(err) => (
            StatusCode::OK,
            Json(crate::models::McpValidateResponse {
                ok: false,
                summary: err.to_string(),
                effective_openwebui_type: String::new(),
                effective_connection_url: String::new(),
                error: Some(ErrorEnvelope {
                    code: "mcp_validate_failed".into(),
                    message: err.to_string(),
                }),
            }),
        )
            .into_response(),
    }
}

pub async fn discover_mcp_tools(
    State(state): State<AppState>,
    Json(req): Json<McpDiscoverRequest>,
) -> impl IntoResponse {
    match mcp::discover_tools(&state, req).await {
        Ok(resp) => (StatusCode::OK, Json(resp)).into_response(),
        Err(err) => (
            StatusCode::OK,
            Json(crate::models::McpDiscoverResponse {
                ok: false,
                summary: err.to_string(),
                server_id: String::new(),
                tools: vec![],
                last_discovered_at: String::new(),
                effective_openwebui_type: String::new(),
                effective_connection_url: String::new(),
                error: Some(ErrorEnvelope {
                    code: "mcp_discover_failed".into(),
                    message: err.to_string(),
                }),
            }),
        )
            .into_response(),
    }
}

pub async fn mcp_runtime(State(state): State<AppState>) -> impl IntoResponse {
    match mcp::runtime_status(&state).await {
        Ok(resp) => (StatusCode::OK, Json(resp)).into_response(),
        Err(err) => (
            StatusCode::OK,
            Json(crate::models::McpRuntimeStatusResponse {
                servers: vec![crate::models::McpRuntimeEntry {
                    status: "error".into(),
                    last_error: err.to_string(),
                    ..Default::default()
                }],
            }),
        )
            .into_response(),
    }
}

pub async fn resolve_tool(
    State(state): State<AppState>,
    Json(req): Json<ResolveRequest>,
) -> impl IntoResponse {
    let normalized_arguments = normalize_tool_arguments(&state, &req.tool, &req.arguments);
    match check_tool_access(&state, &req.tool, &req.mode, &normalized_arguments) {
        Ok(def) => (
            StatusCode::OK,
            Json(ResolveResponse {
                allowed: true,
                tool: def.entry.name.clone(),
                reason: "allowed".into(),
                normalized_arguments,
            }),
        ),
        Err((code, message)) => (
            StatusCode::OK,
            Json(ResolveResponse {
                allowed: false,
                tool: req.tool,
                reason: format!("{code}: {message}"),
                normalized_arguments: json!({}),
            }),
        ),
    }
}

pub async fn execute_tool(
    State(state): State<AppState>,
    Json(req): Json<ExecuteRequest>,
) -> impl IntoResponse {
    let start_ms = now_ms();
    let normalized_arguments = normalize_tool_arguments(&state, &req.tool, &req.arguments);
    let (status, response) =
        match check_tool_access(&state, &req.tool, &req.mode, &normalized_arguments) {
            Ok(def) => match executor::dispatch_tool(&state, def, &normalized_arguments).await {
                Ok(resp) => (StatusCode::OK, resp),
                Err((code, message)) => (
                    StatusCode::OK,
                    ExecuteResponse {
                        ok: false,
                        tool: req.tool.clone(),
                        result: Default::default(),
                        summary: message.clone(),
                        truncated: false,
                        error: Some(ErrorEnvelope { code, message }),
                    },
                ),
            },
            Err((code, message)) => (
                StatusCode::OK,
                ExecuteResponse {
                    ok: false,
                    tool: req.tool.clone(),
                    result: Default::default(),
                    summary: message.clone(),
                    truncated: false,
                    error: Some(ErrorEnvelope { code, message }),
                },
            ),
        };

    append_tool_log(
        &state,
        &req,
        &normalized_arguments,
        &response,
        now_ms() - start_ms,
    );
    (status, Json(response))
}

pub async fn openapi_terminal(
    State(state): State<AppState>,
    Json(req): Json<OpenAPITerminalRequest>,
) -> impl IntoResponse {
    let mut arguments = serde_json::Map::new();
    arguments.insert("command".into(), json!(req.command));
    arguments.insert(
        "transport".into(),
        json!(req.transport.unwrap_or_else(|| "local".into())),
    );
    arguments.insert(
        "shell".into(),
        json!(req.shell.unwrap_or_else(|| "powershell".into())),
    );
    if let Some(host_id) = req.host_id {
        arguments.insert("host_id".into(), json!(host_id));
    }
    if let Some(remote_shell) = req.remote_shell {
        arguments.insert("remote_shell".into(), json!(remote_shell));
    }
    if let Some(workdir) = req.workdir {
        arguments.insert("workdir".into(), json!(workdir));
    }
    if let Some(timeout_ms) = req.timeout_ms {
        arguments.insert("timeout_ms".into(), json!(timeout_ms));
    }
    if let Some(ref user_email) = req.user_email {
        arguments.insert("user_email".into(), json!(user_email));
    }
    let execute_req = ExecuteRequest {
        session_id: format!("openapi-terminal-{}", now_ms()),
        mode: "awdp".into(),
        tool: "terminal".into(),
        user_email: req.user_email.clone().unwrap_or_default(),
        arguments: json!(arguments),
    };

    let start_ms = now_ms();
    let normalized_arguments =
        normalize_tool_arguments(&state, &execute_req.tool, &execute_req.arguments);
    let response = match check_tool_access(
        &state,
        &execute_req.tool,
        &execute_req.mode,
        &normalized_arguments,
    ) {
        Ok(def) => match executor::dispatch_tool(&state, def, &normalized_arguments).await {
            Ok(resp) => resp,
            Err((code, message)) => ExecuteResponse {
                ok: false,
                tool: execute_req.tool.clone(),
                result: Default::default(),
                summary: message.clone(),
                truncated: false,
                error: Some(ErrorEnvelope { code, message }),
            },
        },
        Err((code, message)) => ExecuteResponse {
            ok: false,
            tool: execute_req.tool.clone(),
            result: Default::default(),
            summary: message.clone(),
            truncated: false,
            error: Some(ErrorEnvelope { code, message }),
        },
    };

    append_tool_log(
        &state,
        &execute_req,
        &normalized_arguments,
        &response,
        now_ms() - start_ms,
    );

    let openapi_response = OpenAPITerminalResponse {
        ok: response.ok,
        summary: response.summary,
        exit_code: value_as_i32(response.result.get("exit_code")),
        stdout: value_as_string(response.result.get("stdout")),
        stderr: value_as_string(response.result.get("stderr")),
        timed_out: value_as_bool(response.result.get("timed_out")),
        duration_ms: value_as_u64(response.result.get("duration_ms")),
        transport: value_as_string(response.result.get("transport")),
        host_id: value_as_string(response.result.get("host_id")),
        remote_shell: value_as_string(response.result.get("remote_shell")),
        shell: value_as_string(response.result.get("shell")),
        workdir: value_as_string(response.result.get("workdir")),
        truncated: response.truncated,
        error: response.error,
    };

    (StatusCode::OK, Json(openapi_response))
}

fn value_as_string(value: Option<&serde_json::Value>) -> String {
    value
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default()
        .to_string()
}

fn value_as_bool(value: Option<&serde_json::Value>) -> bool {
    value.and_then(serde_json::Value::as_bool).unwrap_or(false)
}

fn value_as_u64(value: Option<&serde_json::Value>) -> u64 {
    value.and_then(serde_json::Value::as_u64).unwrap_or(0)
}

fn value_as_i32(value: Option<&serde_json::Value>) -> i32 {
    value
        .and_then(serde_json::Value::as_i64)
        .and_then(|number| i32::try_from(number).ok())
        .unwrap_or(-1)
}
