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
            "description": "Controlled local and SSH terminal execution for Open WebUI via OpenAPI. The model may switch targets per call: use local for scripts and file transfer orchestration, use ssh for remote execution."
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
                    "description": "Choose transport per call. Use local for host scripts, packaging, git, scp or rsync orchestration. Use ssh with host_id for remote directories, logs, processes, and running tasks on that host. A single conversation may mix local and ssh calls.",
                    "requestBody": {
                        "required": true,
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "object",
                                    "additionalProperties": false,
                                    "required": ["command"],
                                    "properties": {
                                        "command": {
                                            "type": "string",
                                            "minLength": 1,
                                            "description": "Shell command to run on the selected execution target."
                                        },
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
                                            "default": "local",
                                            "description": "Execution target kind. Use local for host commands and ssh for remote commands."
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
                                            "default": "powershell",
                                            "description": "Local shell. Only used when transport=local."
                                        },
                                        "host_id": {
                                            "anyOf": [
                                                { "type": "string", "minLength": 1 },
                                                { "type": "null" }
                                            ],
                                            "description": "Required when transport=ssh. Select one authorized SSH target."
                                        },
                                        "remote_shell": {
                                            "anyOf": [
                                                {
                                                    "type": "string",
                                                    "enum": ["bash", "powershell"]
                                                },
                                                { "type": "null" }
                                            ],
                                            "description": "Remote shell to launch on the SSH host. Only used when transport=ssh."
                                        },
                                        "workdir": {
                                            "anyOf": [
                                                { "type": "string", "minLength": 1 },
                                                { "type": "null" }
                                            ],
                                            "description": "Working directory on the selected target. Keep it inside the target allowed paths."
                                        },
                                        "timeout_ms": {
                                            "anyOf": [
                                                { "type": "integer", "minimum": 1 },
                                                { "type": "null" }
                                            ],
                                            "description": "Overall execution timeout in milliseconds."
                                        },
                                        "user_email": {
                                            "anyOf": [
                                                { "type": "string", "minLength": 3 },
                                                { "type": "null" }
                                            ],
                                            "description": "Current Open WebUI user email. Used to enforce per-user execution target authorization."
                                        },
                                        "available_execution_targets": {
                                            "anyOf": [
                                                {
                                                    "type": "array",
                                                    "items": {
                                                        "type": "object",
                                                        "additionalProperties": false,
                                                        "required": ["target_id", "kind", "label", "shells"],
                                                        "properties": {
                                                            "target_id": { "type": "string" },
                                                            "kind": { "type": "string", "enum": ["local", "ssh"] },
                                                            "label": { "type": "string" },
                                                            "shells": {
                                                                "type": "array",
                                                                "items": { "type": "string" }
                                                            },
                                                            "default_workdir": { "type": "string" },
                                                            "allowed_paths": {
                                                                "type": "array",
                                                                "items": { "type": "string" }
                                                            },
                                                            "recommended_use": { "type": "string" }
                                                        }
                                                    }
                                                },
                                                { "type": "null" }
                                            ],
                                            "description": "Dynamic per-user execution targets. When present, choose one of these targets for this call."
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
                                            "truncated",
                                            "stdout_truncated",
                                            "stderr_truncated",
                                            "connection_reused",
                                            "failure_phase",
                                            "queue_wait_ms"
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
                                            "stdout_truncated": { "type": "boolean" },
                                            "stderr_truncated": { "type": "boolean" },
                                            "connection_reused": { "type": "boolean" },
                                            "failure_phase": { "type": "string" },
                                            "queue_wait_ms": { "type": "integer" },
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
                phases: vec![],
                suggested_fix: String::new(),
                error: Some(ErrorEnvelope {
                    code: "ssh_test_failed".into(),
                    message: err.to_string(),
                }),
            }),
        )
            .into_response(),
    }
}

pub async fn ssh_runtime(State(state): State<AppState>) -> impl IntoResponse {
    match ssh::ssh_runtime_status(&state).await {
        Ok(resp) => (StatusCode::OK, Json(resp)).into_response(),
        Err(err) => (
            StatusCode::OK,
            Json(crate::models::SshRuntimeStatusResponse {
                configured_host_count: 0,
                active_connection_count: 0,
                hosts: vec![crate::models::SshRuntimeEntry {
                    last_connect_error: err.to_string(),
                    ..Default::default()
                }],
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
    let request_arguments = with_request_user_email(&req.arguments, &req.user_email);
    let normalized_arguments = normalize_tool_arguments(&state, &req.tool, &request_arguments);
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
    let request_arguments = with_request_user_email(&req.arguments, &req.user_email);
    let normalized_arguments = normalize_tool_arguments(&state, &req.tool, &request_arguments);
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
    if let Some(ref available_execution_targets) = req.available_execution_targets {
        arguments.insert(
            "available_execution_targets".into(),
            serde_json::to_value(available_execution_targets).unwrap_or_else(|_| json!([])),
        );
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
        stdout_truncated: value_as_bool(response.result.get("stdout_truncated")),
        stderr_truncated: value_as_bool(response.result.get("stderr_truncated")),
        connection_reused: value_as_bool(response.result.get("connection_reused")),
        failure_phase: value_as_string(response.result.get("failure_phase")),
        queue_wait_ms: value_as_u64(response.result.get("queue_wait_ms")),
        error: response.error,
    };

    (StatusCode::OK, Json(openapi_response))
}

fn with_request_user_email(arguments: &serde_json::Value, user_email: &str) -> serde_json::Value {
    if user_email.trim().is_empty() {
        return arguments.clone();
    }
    let serde_json::Value::Object(mut object) = arguments.clone() else {
        return arguments.clone();
    };
    if !object.contains_key("user_email") {
        object.insert("user_email".into(), json!(user_email.trim()));
    }
    serde_json::Value::Object(object)
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
