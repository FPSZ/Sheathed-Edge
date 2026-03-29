use std::path::{Path, PathBuf};

use serde_json::{Map, Value};

use crate::{
    config::normalize_runtime_path,
    models::{AppState, ToolDef},
};

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

    if tool == "terminal" {
        validate_terminal_access(def, arguments)?;
    }

    Ok(def)
}

pub fn normalize_tool_arguments(state: &AppState, tool: &str, arguments: &Value) -> Value {
    let Value::Object(object) = arguments else {
        return arguments.clone();
    };

    let mut normalized = object.clone();
    match tool {
        "retrieval" | "filesystem/search" => normalize_search_arguments(&mut normalized),
        "terminal" => normalize_terminal_arguments(state, tool, &mut normalized),
        _ => {}
    }

    Value::Object(normalized)
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

fn normalize_search_arguments(arguments: &mut Map<String, Value>) {
    if !arguments.contains_key("limit") {
        if let Some(max_docs) = arguments.remove("max_docs") {
            arguments.insert("limit".into(), max_docs);
        }
    } else {
        arguments.remove("max_docs");
    }

    arguments.remove("source");
}

fn normalize_terminal_arguments(state: &AppState, tool: &str, arguments: &mut Map<String, Value>) {
    let Some(def) = state.tools.get(tool) else {
        return;
    };

    let shell_is_missing = !arguments.contains_key("shell")
        || arguments.get("shell").is_some_and(Value::is_null)
        || arguments
            .get("shell")
            .and_then(Value::as_str)
            .is_some_and(|value| value.trim().is_empty());
    if shell_is_missing {
        arguments.insert("shell".into(), Value::String("powershell".into()));
    }

    let workdir_is_missing = !arguments.contains_key("workdir")
        || arguments.get("workdir").is_some_and(Value::is_null)
        || arguments
            .get("workdir")
            .and_then(Value::as_str)
            .is_some_and(|value| value.trim().is_empty());
    if workdir_is_missing {
        arguments.insert("workdir".into(), Value::String(def.entry.workdir.clone()));
    } else if let Some(workdir) = arguments.get("workdir").and_then(Value::as_str) {
        arguments.insert(
            "workdir".into(),
            Value::String(normalize_runtime_path(workdir)),
        );
    }

    let default_timeout = if state.config.timeouts.default_ms > 0 {
        state.config.timeouts.default_ms
    } else {
        def.entry.timeout_ms
    };
    let max_timeout = if state.config.timeouts.max_ms > 0 {
        state.config.timeouts.max_ms
    } else {
        def.entry.timeout_ms.max(default_timeout)
    };
    let timeout = arguments
        .get("timeout_ms")
        .and_then(Value::as_u64)
        .unwrap_or(default_timeout)
        .clamp(1, max_timeout);
    arguments.insert(
        "timeout_ms".into(),
        Value::Number(serde_json::Number::from(timeout)),
    );
}

fn validate_terminal_access(def: &ToolDef, arguments: &Value) -> std::result::Result<(), (String, String)> {
    let shell = arguments
        .get("shell")
        .and_then(Value::as_str)
        .unwrap_or("powershell");
    if shell != "powershell" && shell != "wsl-bash" {
        return Err((
            "invalid_arguments".into(),
            format!("unsupported shell: {shell}"),
        ));
    }

    let workdir = arguments
        .get("workdir")
        .and_then(Value::as_str)
        .unwrap_or(&def.entry.workdir);
    let candidate = runtime_path_to_windows(workdir);
    if !def
        .entry
        .allowed_paths
        .iter()
        .map(|allowed| runtime_path_to_windows(allowed))
        .any(|allowed| is_within_allowed_path_windows(&candidate, &allowed))
    {
        return Err((
            "path_denied".into(),
            format!("terminal workdir is outside allowed paths: {workdir}"),
        ));
    }

    Ok(())
}

pub fn runtime_path_to_windows(path: &str) -> String {
    let normalized = path.replace('\\', "/");
    let bytes = normalized.as_bytes();
    if normalized.starts_with("/mnt/") && bytes.len() >= 7 && bytes[6] == b'/' {
        let drive = normalized[5..6].to_ascii_uppercase();
        let rest = &normalized[7..];
        if rest.is_empty() {
            return format!("{drive}:/");
        }
        return format!("{drive}:/{rest}");
    }
    if bytes.len() >= 2 && bytes[1] == b':' {
        let drive = normalized[0..1].to_ascii_uppercase();
        return format!("{drive}{}", &normalized[1..]);
    }
    normalized
}

pub fn windows_path_to_wsl(path: &str) -> Option<String> {
    let normalized = runtime_path_to_windows(path);
    let bytes = normalized.as_bytes();
    if bytes.len() >= 3 && bytes[1] == b':' && (bytes[2] == b'/' || bytes[2] == b'\\') {
        let drive = normalized[0..1].to_ascii_lowercase();
        let rest = normalized[3..].replace('\\', "/");
        if rest.is_empty() {
            return Some(format!("/mnt/{drive}"));
        }
        return Some(format!("/mnt/{drive}/{rest}"));
    }
    if normalized.starts_with("/mnt/") {
        return Some(normalized);
    }
    None
}

fn is_within_allowed_path_windows(candidate: &str, allowed: &str) -> bool {
    let candidate_lower = runtime_path_to_windows(candidate).to_ascii_lowercase();
    let allowed_lower = runtime_path_to_windows(allowed).to_ascii_lowercase();

    let candidate_path = Path::new(&candidate_lower);
    let allowed_path = Path::new(&allowed_lower);
    candidate_path.starts_with(allowed_path)
}

#[cfg(test)]
mod tests {
    use super::{runtime_path_to_windows, windows_path_to_wsl};

    #[test]
    fn converts_wsl_path_to_windows_path() {
        assert_eq!(
            runtime_path_to_windows("/mnt/d/AI/Local"),
            "D:/AI/Local"
        );
    }

    #[test]
    fn converts_windows_path_to_wsl_path() {
        assert_eq!(
            windows_path_to_wsl("D:/Environment2/Create").as_deref(),
            Some("/mnt/d/Environment2/Create")
        );
    }
}
