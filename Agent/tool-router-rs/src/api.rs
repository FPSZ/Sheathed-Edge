use axum::{Json, extract::State, http::StatusCode, response::IntoResponse};
use serde_json::json;

use crate::{
    executor,
    logging::{append_tool_log, now_ms},
    models::{
        AppState, ErrorEnvelope, ExecuteRequest, ExecuteResponse, ResolveRequest, ResolveResponse,
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

pub async fn resolve_tool(
    State(state): State<AppState>,
    Json(req): Json<ResolveRequest>,
) -> impl IntoResponse {
    let normalized_arguments = normalize_tool_arguments(&req.tool, &req.arguments);
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
    let normalized_arguments = normalize_tool_arguments(&req.tool, &req.arguments);
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

    append_tool_log(&state, &req, &response, now_ms() - start_ms);
    (status, Json(response))
}
