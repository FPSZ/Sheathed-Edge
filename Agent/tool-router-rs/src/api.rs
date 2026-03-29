use axum::{Json, extract::State, http::StatusCode, response::IntoResponse};
use serde_json::json;

use crate::{
    executor,
    logging::{append_tool_log, now_ms},
    models::{
        AppState, ErrorEnvelope, ExecuteRequest, ExecuteResponse, OpenAPITerminalRequest,
        OpenAPITerminalResponse, ResolveRequest, ResolveResponse,
    },
    policy::{check_tool_access, normalize_tool_arguments},
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
            "description": "Controlled local terminal execution for Open WebUI via OpenAPI."
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
                    "summary": "Run a controlled local shell command",
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

    append_tool_log(&state, &req, &normalized_arguments, &response, now_ms() - start_ms);
    (status, Json(response))
}

pub async fn openapi_terminal(
    State(state): State<AppState>,
    Json(req): Json<OpenAPITerminalRequest>,
) -> impl IntoResponse {
    let arguments = json!({
        "command": req.command,
        "shell": req.shell.unwrap_or_else(|| "powershell".into()),
        "workdir": req.workdir,
        "timeout_ms": req.timeout_ms,
    });
    let execute_req = ExecuteRequest {
        session_id: format!("openapi-terminal-{}", now_ms()),
        mode: "awdp".into(),
        tool: "terminal".into(),
        arguments,
    };

    let start_ms = now_ms();
    let normalized_arguments = normalize_tool_arguments(&state, &execute_req.tool, &execute_req.arguments);
    let response =
        match check_tool_access(&state, &execute_req.tool, &execute_req.mode, &normalized_arguments) {
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
