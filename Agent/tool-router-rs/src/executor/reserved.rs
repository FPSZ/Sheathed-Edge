use serde_json::Value;

use crate::models::{AppState, ExecuteResponse, ToolDef};

pub async fn not_implemented(
    _state: &AppState,
    def: &ToolDef,
    _arguments: &Value,
) -> std::result::Result<ExecuteResponse, (String, String)> {
    Err((
        "tool_not_implemented".into(),
        format!("tool not implemented in current phase: {}", def.entry.name),
    ))
}
