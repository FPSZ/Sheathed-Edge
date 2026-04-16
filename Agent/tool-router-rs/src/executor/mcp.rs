use std::collections::BTreeMap;

use serde_json::Value;

use crate::{
    mcp,
    models::{AppState, ExecuteResponse, ToolDef},
};

pub async fn run_mcp_tool(
    state: &AppState,
    def: &ToolDef,
    arguments: &Value,
) -> std::result::Result<ExecuteResponse, (String, String)> {
    let server_id = def
        .entry
        .env
        .get("mcp_server_id")
        .map(|value| value.trim())
        .filter(|value| !value.is_empty())
        .ok_or_else(|| {
            (
                "mcp_invalid_config".into(),
                format!("missing mcp_server_id for tool {}", def.entry.name),
            )
        })?;

    let result = mcp::invoke_tool(state, server_id, &def.entry.name, arguments).await?;
    let summary = build_summary(&result);

    Ok(ExecuteResponse {
        ok: true,
        tool: def.entry.name.clone(),
        result: to_btreemap(result),
        summary,
        truncated: false,
        error: None,
    })
}

fn build_summary(result: &Value) -> String {
    if let Some(path) = result.get("path").and_then(Value::as_str) {
        if let Some(entries) = result.get("entries").and_then(Value::as_array) {
            return format!("listed {} entries from {}", entries.len(), path);
        }
        if result.get("content").is_some() {
            return format!("read text file {}", path);
        }
    }
    "mcp tool executed".into()
}

fn to_btreemap(value: Value) -> BTreeMap<String, Value> {
    match value {
        Value::Object(map) => map.into_iter().collect(),
        other => {
            let mut map = BTreeMap::new();
            map.insert("value".into(), other);
            map
        }
    }
}
