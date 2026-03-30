use axum::{Json, extract::State, http::StatusCode};

use crate::{
    llama,
    models::{SharedState, SwitchRequest, UpdateProfileRequest},
};

pub async fn healthz() -> Json<serde_json::Value> {
    Json(serde_json::json!({ "status": "ok" }))
}

pub async fn llama_status(State(state): State<SharedState>) -> Json<crate::models::StatusResponse> {
    Json(llama::status(&state).await)
}

pub async fn llama_start(
    State(state): State<SharedState>,
) -> Result<Json<crate::models::StatusResponse>, (StatusCode, Json<serde_json::Value>)> {
    llama::start(&state).await.map(Json).map_err(internal_error)
}

pub async fn llama_stop(
    State(state): State<SharedState>,
) -> Result<Json<crate::models::StatusResponse>, (StatusCode, Json<serde_json::Value>)> {
    llama::stop(&state).await.map(Json).map_err(internal_error)
}

pub async fn llama_restart(
    State(state): State<SharedState>,
) -> Result<Json<crate::models::StatusResponse>, (StatusCode, Json<serde_json::Value>)> {
    llama::restart(&state)
        .await
        .map(Json)
        .map_err(internal_error)
}

pub async fn llama_switch(
    State(state): State<SharedState>,
    Json(req): Json<SwitchRequest>,
) -> Result<Json<crate::models::StatusResponse>, (StatusCode, Json<serde_json::Value>)> {
    llama::switch_profile(&state, &req.profile_id)
        .await
        .map(Json)
        .map_err(internal_error)
}

pub async fn profile_update(
    State(state): State<SharedState>,
    Json(req): Json<UpdateProfileRequest>,
) -> Result<Json<crate::models::ModelProfile>, (StatusCode, Json<serde_json::Value>)> {
    llama::update_profile(&state, req.profile)
        .await
        .map(Json)
        .map_err(internal_error)
}

pub async fn shutdown(
    State(state): State<SharedState>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    let _ = llama::stop(&state).await;
    state.shutdown.notify_waiters();
    Ok(Json(serde_json::json!({
        "ok": true,
        "summary": "host-agent shutdown requested"
    })))
}

fn internal_error(err: anyhow::Error) -> (StatusCode, Json<serde_json::Value>) {
    (
        StatusCode::BAD_GATEWAY,
        Json(serde_json::json!({
            "error": {
                "code": "host_agent_error",
                "message": err.to_string()
            }
        })),
    )
}
