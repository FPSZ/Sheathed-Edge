use std::path::PathBuf;

use serde_json::Value;

use crate::models::{AppState, ToolDef};

pub fn check_tool_access<'a>(
    state: &'a AppState,
    tool: &str,
    mode: &str,
    arguments: &Value,
) -> std::result::Result<&'a ToolDef, (String, String)> {
    let Some(def) = state.tools.get(tool) else {
        return Err(("tool_not_found".into(), format!("unknown tool: {tool}")));
    };
    if !def.entry.enabled {
        return Err((
            "tool_disabled".into(),
            format!("tool is reserved but not enabled: {tool}"),
        ));
    }

    let scopes: Vec<&str> = mode.split('+').collect();
    if !def
        .entry
        .plugin_scope
        .iter()
        .any(|scope| scopes.iter().any(|active| active == scope))
    {
        return Err((
            "scope_denied".into(),
            format!("tool {tool} is not allowed for mode {mode}"),
        ));
    }

    if let Err(error) = def.validator.validate(arguments) {
        return Err(("schema_invalid".into(), error.to_string()));
    }

    Ok(def)
}

pub fn is_within_allowed_path(candidate: &str, allowed: &str) -> bool {
    let candidate_path = PathBuf::from(candidate);
    let allowed_path = PathBuf::from(allowed);
    candidate_path.starts_with(allowed_path)
}

pub fn timeout_with_retry(timeout_ms: u64, retry_max_attempts: u32, retry_backoff_ms: u64) -> u64 {
    let extra = retry_backoff_ms.saturating_mul(retry_max_attempts.saturating_sub(1) as u64);
    timeout_ms.saturating_add(extra)
}
