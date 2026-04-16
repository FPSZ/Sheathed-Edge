use std::{
    fs,
    path::{Path, PathBuf},
};

use serde_json::{Map, Value};

use crate::{
    config::normalize_runtime_path,
    mcp,
    models::{AppState, ExecutionTarget, ToolDef, UserWorkspace},
    ssh,
};

pub fn check_tool_access(
    state: &AppState,
    tool: &str,
    mode: &str,
    arguments: &Value,
) -> std::result::Result<ToolDef, (String, String)> {
    let def = if let Some(def) = state.tools.get(tool) {
        def.clone()
    } else if let Some(user_email) = arguments.get("user_email").and_then(Value::as_str) {
        mcp::resolve_dynamic_tool(state, tool, mode, user_email)?
            .ok_or_else(|| ("tool_not_found".into(), format!("unknown tool: {tool}")))?
    } else {
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
        validate_terminal_access(state, &def, arguments)?;
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

    let user_workspace = arguments
        .get("user_email")
        .and_then(Value::as_str)
        .and_then(|user_email| load_user_workspace(state, user_email));

    if !arguments.contains_key("available_execution_targets") {
        if let Some(workspace) = user_workspace.as_ref() {
            let targets = build_execution_targets(state, workspace);
            if !targets.is_empty() {
                arguments.insert(
                    "available_execution_targets".into(),
                    serde_json::to_value(targets).unwrap_or(Value::Array(vec![])),
                );
            }
        }
    }

    let transport_is_missing = !arguments.contains_key("transport")
        || arguments.get("transport").is_some_and(Value::is_null)
        || arguments
            .get("transport")
            .and_then(Value::as_str)
            .is_some_and(|value| value.trim().is_empty());
    if transport_is_missing {
        let preferred_transport = preferred_terminal_transport(user_workspace.as_ref());
        arguments.insert(
            "transport".into(),
            Value::String(preferred_transport.to_string()),
        );
    }

    let transport = arguments
        .get("transport")
        .and_then(Value::as_str)
        .unwrap_or("local");

    if transport == "local" {
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
    } else if transport == "ssh" {
        if let Some(host_id) = arguments.get("host_id").and_then(Value::as_str) {
            arguments.insert("host_id".into(), Value::String(host_id.trim().to_string()));
        }

        let host_id_is_missing = !arguments.contains_key("host_id")
            || arguments.get("host_id").is_some_and(Value::is_null)
            || arguments
                .get("host_id")
                .and_then(Value::as_str)
                .is_some_and(|value| value.trim().is_empty());
        if host_id_is_missing {
            if let Some(host_id) = preferred_ssh_host_id(user_workspace.as_ref()) {
                arguments.insert("host_id".into(), Value::String(host_id));
            }
        }

        let workdir_is_missing = !arguments.contains_key("workdir")
            || arguments.get("workdir").is_some_and(Value::is_null)
            || arguments
                .get("workdir")
                .and_then(Value::as_str)
                .is_some_and(|value| value.trim().is_empty());
        if workdir_is_missing {
            if let Ok(settings) =
                ssh::effective_ssh_settings(state, &def.entry, &Value::Object(arguments.clone()))
            {
                arguments.insert("workdir".into(), Value::String(settings.workdir));
                arguments.insert("remote_shell".into(), Value::String(settings.remote_shell));
            }
        } else if let Some(workdir) = arguments.get("workdir").and_then(Value::as_str) {
            arguments.insert("workdir".into(), Value::String(workdir.trim().to_string()));
        }

        let remote_shell_is_missing = !arguments.contains_key("remote_shell")
            || arguments.get("remote_shell").is_some_and(Value::is_null)
            || arguments
                .get("remote_shell")
                .and_then(Value::as_str)
                .is_some_and(|value| value.trim().is_empty());
        if remote_shell_is_missing {
            if let Ok(settings) =
                ssh::effective_ssh_settings(state, &def.entry, &Value::Object(arguments.clone()))
            {
                arguments.insert("remote_shell".into(), Value::String(settings.remote_shell));
            }
        }
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

fn validate_terminal_access(
    state: &AppState,
    def: &ToolDef,
    arguments: &Value,
) -> std::result::Result<(), (String, String)> {
    let transport = arguments
        .get("transport")
        .and_then(Value::as_str)
        .unwrap_or("local");
    let user_email = arguments
        .get("user_email")
        .and_then(Value::as_str)
        .unwrap_or_default();
    validate_execution_target_access(state, user_email, transport, arguments)?;

    match transport {
        "local" => {
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
        "ssh" => {
            let settings = ssh::effective_ssh_settings(state, &def.entry, arguments)?;
            if !settings.host.allowed_paths.iter().any(|allowed| {
                ssh::is_within_remote_allowed_path(
                    &settings.workdir,
                    allowed,
                    &settings.remote_shell,
                )
            }) {
                return Err((
                    "path_denied".into(),
                    format!(
                        "ssh terminal workdir is outside allowed paths: {}",
                        settings.workdir
                    ),
                ));
            }
            Ok(())
        }
        other => Err((
            "invalid_arguments".into(),
            format!("unsupported transport: {other}"),
        )),
    }
}

fn validate_execution_target_access(
    state: &AppState,
    user_email: &str,
    transport: &str,
    arguments: &Value,
) -> std::result::Result<(), (String, String)> {
    if user_email.trim().is_empty() {
        return Ok(());
    }

    let workspace = load_user_workspace(state, user_email).ok_or_else(|| {
        (
            "terminal_not_authorized".into(),
            format!("terminal is not enabled for user {user_email}"),
        )
    })?;
    let allowed_targets = hydrated_execution_targets(&workspace);

    match transport {
        "local" => {
            if allowed_targets
                .iter()
                .any(|target| target.eq_ignore_ascii_case("local"))
            {
                Ok(())
            } else {
                Err((
                    "terminal_not_authorized".into(),
                    format!("local terminal is not enabled for user {user_email}"),
                ))
            }
        }
        "ssh" => {
            let host_id = arguments
                .get("host_id")
                .and_then(Value::as_str)
                .map(str::trim)
                .unwrap_or_default();
            if host_id.is_empty() {
                return Err((
                    "terminal_not_authorized".into(),
                    "ssh terminal requires host_id".into(),
                ));
            }
            let target_id = format!("ssh:{host_id}");
            if allowed_targets
                .iter()
                .any(|target| target.eq_ignore_ascii_case(&target_id))
            {
                Ok(())
            } else {
                Err((
                    "terminal_not_authorized".into(),
                    format!("ssh target {host_id} is not enabled for user {user_email}"),
                ))
            }
        }
        _ => Ok(()),
    }
}

fn available_execution_targets_for_user(
    state: &AppState,
    user_email: &str,
) -> Vec<ExecutionTarget> {
    let Some(workspace) = load_user_workspace(state, user_email) else {
        return vec![];
    };
    build_execution_targets(state, &workspace)
}

fn preferred_terminal_transport(workspace: Option<&UserWorkspace>) -> &'static str {
    let Some(workspace) = workspace else {
        return "local";
    };

    if preferred_ssh_host_id(Some(workspace)).is_some() {
        return "ssh";
    }

    let targets = hydrated_execution_targets(workspace);
    if targets
        .iter()
        .any(|target| target.eq_ignore_ascii_case("local"))
    {
        "local"
    } else if targets
        .iter()
        .any(|target| target.to_ascii_lowercase().starts_with("ssh:"))
    {
        "ssh"
    } else {
        "local"
    }
}

fn preferred_ssh_host_id(workspace: Option<&UserWorkspace>) -> Option<String> {
    let workspace = workspace?;
    let default_host_id = workspace.default_ssh_host_id.trim();
    if !default_host_id.is_empty() {
        return Some(default_host_id.to_string());
    }

    let targets = hydrated_execution_targets(workspace);
    if targets
        .iter()
        .any(|target| target.eq_ignore_ascii_case("local"))
    {
        return None;
    }

    let mut ssh_targets = targets
        .into_iter()
        .filter_map(|target| target.strip_prefix("ssh:").map(str::to_string));
    let first = ssh_targets.next()?;
    if ssh_targets.next().is_none() {
        return Some(first);
    }
    None
}

pub fn load_user_workspace(state: &AppState, user_email: &str) -> Option<UserWorkspace> {
    let normalized_email = user_email.trim().to_ascii_lowercase();
    if normalized_email.is_empty() {
        return None;
    }

    let path = if state.user_settings_path.trim().is_empty() {
        return None;
    } else {
        state.user_settings_path.trim()
    };
    let data = fs::read_to_string(path).ok()?;

    let users: Vec<UserWorkspace> = match serde_json::from_str::<Vec<UserWorkspace>>(&data) {
        Ok(items) => items,
        Err(_) => {
            let wrapped = serde_json::from_str::<serde_json::Value>(&data).ok()?;
            serde_json::from_value::<Vec<UserWorkspace>>(
                wrapped
                    .get("users")
                    .cloned()
                    .unwrap_or(Value::Array(vec![])),
            )
            .ok()?
        }
    };

    users.into_iter().find(|workspace| {
        workspace
            .user_email
            .trim()
            .eq_ignore_ascii_case(&normalized_email)
    })
}

fn hydrated_execution_targets(workspace: &UserWorkspace) -> Vec<String> {
    let mut targets = sanitize_string_list(&workspace.enabled_execution_targets);
    if targets.is_empty()
        && workspace
            .user_email
            .trim()
            .eq_ignore_ascii_case("3223659402@qq.com")
    {
        targets.push("local".into());
    }
    if !workspace.default_ssh_host_id.trim().is_empty() {
        targets.push(format!("ssh:{}", workspace.default_ssh_host_id.trim()));
    }
    sanitize_string_list(&targets)
}

fn build_execution_targets(state: &AppState, workspace: &UserWorkspace) -> Vec<ExecutionTarget> {
    let targets = hydrated_execution_targets(workspace);
    let hosts = ssh::load_ssh_hosts(&state.ssh_hosts_path).unwrap_or_default();
    let mut out = Vec::with_capacity(targets.len());

    for target in targets {
        if target.eq_ignore_ascii_case("local") {
            let default_workdir = if workspace.default_local_workdir.trim().is_empty() {
                workspace
                    .terminal_allowed_paths
                    .first()
                    .cloned()
                    .unwrap_or_default()
            } else {
                workspace.default_local_workdir.trim().to_string()
            };
            out.push(ExecutionTarget {
                target_id: "local".into(),
                kind: "local".into(),
                label: "Local".into(),
                shells: vec!["powershell".into(), "wsl-bash".into()],
                default_workdir,
                allowed_paths: workspace.terminal_allowed_paths.clone(),
                recommended_use:
                    "Use local for scripts, repo operations, packaging, scp or rsync orchestration."
                        .into(),
            });
            continue;
        }

        if let Some(host_id) = target.strip_prefix("ssh:") {
            let Some(host) = hosts
                .iter()
                .find(|item| item.id.trim().eq_ignore_ascii_case(host_id.trim()) && item.enabled)
            else {
                continue;
            };
            let default_workdir = host
                .default_workdir
                .clone()
                .unwrap_or_else(|| host.allowed_paths.first().cloned().unwrap_or_default());
            let mut shells = vec![host.remote_shell_default.clone()];
            if !shells.iter().any(|shell| shell == "bash") {
                shells.push("bash".into());
            }
            if !shells.iter().any(|shell| shell == "powershell") {
                shells.push("powershell".into());
            }
            out.push(ExecutionTarget {
                target_id: format!("ssh:{}", host.id),
                kind: "ssh".into(),
                label: if host.label.trim().is_empty() {
                    host.id.clone()
                } else {
                    host.label.clone()
                },
                shells,
                default_workdir,
                allowed_paths: host.allowed_paths.clone(),
                recommended_use:
                    "Use ssh for remote directories, remote process control, and running tasks on that host."
                        .into(),
            });
        }
    }

    out
}

fn sanitize_string_list(items: &[String]) -> Vec<String> {
    let mut cleaned = Vec::with_capacity(items.len());
    for item in items {
        let value = item.trim();
        if value.is_empty() {
            continue;
        }
        if cleaned
            .iter()
            .any(|existing: &String| existing.eq_ignore_ascii_case(value))
        {
            continue;
        }
        cleaned.push(value.to_string());
    }
    cleaned
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
    use super::{
        preferred_ssh_host_id, preferred_terminal_transport, runtime_path_to_windows,
        windows_path_to_wsl,
    };
    use crate::models::UserWorkspace;

    #[test]
    fn converts_wsl_path_to_windows_path() {
        assert_eq!(runtime_path_to_windows("/mnt/d/AI/Local"), "D:/AI/Local");
    }

    #[test]
    fn converts_windows_path_to_wsl_path() {
        assert_eq!(
            windows_path_to_wsl("D:/Environment2/Create").as_deref(),
            Some("/mnt/d/Environment2/Create")
        );
    }

    #[test]
    fn prefers_default_ssh_transport_when_workspace_declares_one() {
        let workspace = UserWorkspace {
            user_email: "teammate@example.com".into(),
            default_ssh_host_id: "team-a-box".into(),
            ..Default::default()
        };

        assert_eq!(preferred_terminal_transport(Some(&workspace)), "ssh");
        assert_eq!(
            preferred_ssh_host_id(Some(&workspace)).as_deref(),
            Some("team-a-box")
        );
    }

    #[test]
    fn falls_back_to_only_ssh_target_when_no_local_is_enabled() {
        let workspace = UserWorkspace {
            user_email: "teammate@example.com".into(),
            enabled_execution_targets: vec!["ssh:team-b-box".into()],
            ..Default::default()
        };

        assert_eq!(preferred_terminal_transport(Some(&workspace)), "ssh");
        assert_eq!(
            preferred_ssh_host_id(Some(&workspace)).as_deref(),
            Some("team-b-box")
        );
    }

    #[test]
    fn keeps_local_as_default_when_local_is_available_and_no_default_ssh() {
        let workspace = UserWorkspace {
            user_email: "owner@example.com".into(),
            enabled_execution_targets: vec!["local".into(), "ssh:team-b-box".into()],
            ..Default::default()
        };

        assert_eq!(preferred_terminal_transport(Some(&workspace)), "local");
        assert_eq!(preferred_ssh_host_id(Some(&workspace)), None);
    }
}
