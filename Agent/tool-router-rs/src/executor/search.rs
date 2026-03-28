use std::{collections::BTreeMap, process::Stdio};

use serde_json::Value;
use tokio::process::Command;

use crate::{config::normalize_runtime_path, models::{ExecuteResponse, ToolEntry}, policy::{is_within_allowed_path, timeout_with_retry}};

pub async fn run_rg_search(
    tool: &ToolEntry,
    arguments: &Value,
    retrieval_mode: bool,
) -> std::result::Result<ExecuteResponse, (String, String)> {
    let query = arguments
        .get("query")
        .and_then(Value::as_str)
        .ok_or_else(|| ("invalid_arguments".into(), "missing query".into()))?;
    let limit = arguments
        .get("limit")
        .and_then(Value::as_u64)
        .unwrap_or(if retrieval_mode { 8 } else { 10 });

    let mut roots = tool.allowed_paths.clone();
    if let Some(requested_roots) = arguments.get("roots").and_then(Value::as_array) {
        let mut filtered = Vec::new();
        for item in requested_roots {
            if let Some(root) = item.as_str() {
                let normalized = normalize_runtime_path(root);
                if tool
                    .allowed_paths
                    .iter()
                    .any(|allowed| is_within_allowed_path(&normalized, allowed))
                {
                    filtered.push(normalized);
                }
            }
        }
        if !filtered.is_empty() {
            roots = filtered;
        }
    }

    if roots.is_empty() {
        return Err(("path_denied".into(), "no allowed search roots".into()));
    }

    let mut cmd = Command::new("rg");
    cmd.arg("-n")
        .arg("-S")
        .arg("--hidden")
        .arg("--max-count")
        .arg(limit.to_string())
        .arg(query);

    for root in &roots {
        cmd.arg(root);
    }

    if !tool.env.is_empty() {
        cmd.envs(tool.env.iter());
    }
    cmd.stdout(Stdio::piped()).stderr(Stdio::piped());
    cmd.current_dir(&tool.workdir);

    let timeout_ms = timeout_with_retry(tool.timeout_ms, tool.retry.max_attempts, tool.retry.backoff_ms);
    let output = tokio::time::timeout(std::time::Duration::from_millis(timeout_ms), cmd.output())
        .await
        .map_err(|_| ("tool_timeout".into(), format!("tool timed out: {}", tool.name)))?
        .map_err(|err| ("tool_exec_failed".into(), err.to_string()))?;

    let stdout = String::from_utf8_lossy(&output.stdout).to_string();
    let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
    if stdout.trim().is_empty() && !output.status.success() && stderr.is_empty() {
        return Err((
            "tool_exec_failed".into(),
            format!("rg failed with status {}", output.status),
        ));
    }

    let mut matches = Vec::new();
    for line in stdout.lines().take(limit as usize) {
        let parts: Vec<&str> = line.splitn(3, ':').collect();
        let mut item = serde_json::Map::new();
        item.insert("raw".into(), Value::String(line.to_string()));
        if parts.len() >= 3 {
            item.insert("path".into(), Value::String(parts[0].to_string()));
            item.insert("line".into(), Value::String(parts[1].to_string()));
            item.insert("text".into(), Value::String(parts[2].to_string()));
        }
        matches.push(Value::Object(item));
    }

    let mut result = BTreeMap::new();
    result.insert("query".into(), Value::String(query.to_string()));
    result.insert("matches".into(), Value::Array(matches.clone()));
    result.insert(
        "scope".into(),
        Value::Array(tool.plugin_scope.iter().cloned().map(Value::String).collect()),
    );
    result.insert(
        "capabilities".into(),
        Value::Array(tool.capabilities.iter().cloned().map(Value::String).collect()),
    );
    if !stderr.is_empty() {
        result.insert("stderr".into(), Value::String(stderr.clone()));
    }

    let summary = if retrieval_mode {
        format!("retrieval returned {} snippets", matches.len())
    } else {
        format!("filesystem search returned {} matches", matches.len())
    };

    Ok(ExecuteResponse {
        ok: true,
        tool: tool.name.clone(),
        result,
        summary,
        truncated: false,
        error: None,
    })
}
