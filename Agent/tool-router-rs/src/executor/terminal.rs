use std::{collections::BTreeMap, process::Stdio, time::Instant};

use serde_json::Value;
use tokio::process::Command;

use crate::{
    models::{AppState, ExecuteResponse, ToolEntry},
    policy::{runtime_path_to_windows, timeout_with_retry, windows_path_to_wsl},
    ssh,
};

const MAX_CAPTURE_BYTES: usize = 64 * 1024;

pub async fn run_terminal(
    state: &AppState,
    tool: &ToolEntry,
    arguments: &Value,
) -> std::result::Result<ExecuteResponse, (String, String)> {
    let original_command = arguments
        .get("command")
        .and_then(Value::as_str)
        .ok_or_else(|| ("invalid_arguments".into(), "missing command".into()))?;
    let transport = arguments
        .get("transport")
        .and_then(Value::as_str)
        .unwrap_or("local");
    let shell = arguments
        .get("shell")
        .and_then(Value::as_str)
        .unwrap_or("powershell");
    let workdir = arguments
        .get("workdir")
        .and_then(Value::as_str)
        .unwrap_or(&tool.workdir);
    let timeout_ms = arguments
        .get("timeout_ms")
        .and_then(Value::as_u64)
        .unwrap_or_else(|| default_timeout(state, tool));
    let command = if transport == "local" {
        normalize_command(shell, original_command)
    } else {
        original_command.to_string()
    };

    if transport == "ssh" {
        return run_ssh_terminal(state, tool, arguments, original_command).await;
    }

    let effective_timeout =
        timeout_with_retry(timeout_ms, tool.retry.max_attempts, tool.retry.backoff_ms);
    let mut process = build_command(shell, &command, workdir)?;
    if !tool.env.is_empty() {
        process.envs(tool.env.iter());
    }
    process.kill_on_drop(true);
    process.stdout(Stdio::piped()).stderr(Stdio::piped());

    let started = Instant::now();
    let output = tokio::time::timeout(
        std::time::Duration::from_millis(effective_timeout),
        process.output(),
    )
    .await;
    let duration_ms = started.elapsed().as_millis();

    let output = match output {
        Ok(result) => result.map_err(|err| ("tool_exec_failed".into(), err.to_string()))?,
        Err(_) => {
            return Ok(timeout_response(
                tool,
                shell,
                &command,
                workdir,
                duration_ms,
                effective_timeout,
            ));
        }
    };

    let (stdout, stdout_truncated) = truncate_output(&output.stdout);
    let (stderr, stderr_truncated) = truncate_output(&output.stderr);
    let exit_code = output.status.code().unwrap_or(-1);
    let truncated = stdout_truncated || stderr_truncated;

    let mut result = BTreeMap::new();
    result.insert("transport".into(), Value::String("local".into()));
    result.insert("host_id".into(), Value::String(String::new()));
    result.insert("remote_shell".into(), Value::String(String::new()));
    result.insert("shell".into(), Value::String(shell.to_string()));
    result.insert("command".into(), Value::String(command.to_string()));
    result.insert(
        "original_command".into(),
        Value::String(original_command.to_string()),
    );
    result.insert("workdir".into(), Value::String(workdir.to_string()));
    result.insert("exit_code".into(), Value::Number(exit_code.into()));
    result.insert("stdout".into(), Value::String(stdout.clone()));
    result.insert("stderr".into(), Value::String(stderr.clone()));
    result.insert("timed_out".into(), Value::Bool(false));
    result.insert(
        "duration_ms".into(),
        Value::Number(serde_json::Number::from(duration_ms as u64)),
    );

    let summary = if output.status.success() {
        format!("terminal command completed with exit code {exit_code}")
    } else {
        format!("terminal command failed with exit code {exit_code}")
    };

    Ok(ExecuteResponse {
        ok: output.status.success(),
        tool: tool.name.clone(),
        result,
        summary,
        truncated,
        error: if output.status.success() {
            None
        } else {
            Some(crate::models::ErrorEnvelope {
                code: "command_failed".into(),
                message: stderr_if_any(&stderr, exit_code),
            })
        },
    })
}

async fn run_ssh_terminal(
    state: &AppState,
    tool: &ToolEntry,
    arguments: &Value,
    original_command: &str,
) -> std::result::Result<ExecuteResponse, (String, String)> {
    let settings = ssh::effective_ssh_settings(state, tool, arguments)?;
    let duration_start = Instant::now();
    let output = ssh::execute_ssh_command(state, tool, arguments, original_command).await?;
    let duration_ms = duration_start.elapsed().as_millis();
    let (stdout, stdout_truncated) = truncate_output(&output.stdout);
    let (stderr, stderr_truncated) = truncate_output(&output.stderr);
    let truncated = stdout_truncated || stderr_truncated;

    let mut result = BTreeMap::new();
    result.insert("transport".into(), Value::String("ssh".into()));
    result.insert("host_id".into(), Value::String(settings.host.id.clone()));
    result.insert(
        "remote_shell".into(),
        Value::String(settings.remote_shell.clone()),
    );
    result.insert("shell".into(), Value::String(settings.remote_shell.clone()));
    result.insert(
        "host_key_status".into(),
        Value::String(settings.host.host_key_status.clone()),
    );
    if let Some(user_email) = arguments.get("user_email").and_then(Value::as_str) {
        result.insert("user_email".into(), Value::String(user_email.to_string()));
    }
    result.insert(
        "command".into(),
        Value::String(original_command.to_string()),
    );
    result.insert(
        "original_command".into(),
        Value::String(original_command.to_string()),
    );
    result.insert("workdir".into(), Value::String(settings.workdir.clone()));
    result.insert("exit_code".into(), Value::Number(output.exit_code.into()));
    result.insert("stdout".into(), Value::String(stdout.clone()));
    result.insert("stderr".into(), Value::String(stderr.clone()));
    result.insert("timed_out".into(), Value::Bool(false));
    result.insert(
        "duration_ms".into(),
        Value::Number(serde_json::Number::from(duration_ms as u64)),
    );

    let summary = if output.exit_code == 0 {
        format!(
            "ssh terminal command completed with exit code 0 on {}",
            settings.host.id
        )
    } else {
        format!(
            "ssh terminal command failed with exit code {} on {}",
            output.exit_code, settings.host.id
        )
    };

    Ok(ExecuteResponse {
        ok: output.exit_code == 0,
        tool: tool.name.clone(),
        result,
        summary,
        truncated,
        error: if output.exit_code == 0 {
            None
        } else {
            Some(crate::models::ErrorEnvelope {
                code: "command_failed".into(),
                message: stderr_if_any(&stderr, output.exit_code),
            })
        },
    })
}

fn build_command(
    shell: &str,
    command: &str,
    workdir: &str,
) -> std::result::Result<Command, (String, String)> {
    match shell {
        "powershell" => {
            let workdir = runtime_path_to_windows(workdir);
            let mut cmd = Command::new("powershell.exe");
            cmd.args([
                "-NoProfile",
                "-NonInteractive",
                "-ExecutionPolicy",
                "Bypass",
                "-Command",
                command,
            ]);
            cmd.current_dir(workdir);
            Ok(cmd)
        }
        "wsl-bash" => {
            let wsl_workdir = windows_path_to_wsl(workdir).ok_or_else(|| {
                (
                    "invalid_arguments".into(),
                    format!("workdir cannot be mapped into WSL: {workdir}"),
                )
            })?;
            let shell_command = format!("cd -- {} && {}", single_quote(&wsl_workdir), command);
            let mut cmd = Command::new("wsl.exe");
            cmd.args(["-e", "bash", "-lc", &shell_command]);
            Ok(cmd)
        }
        other => Err((
            "invalid_arguments".into(),
            format!("unsupported shell: {other}"),
        )),
    }
}

fn normalize_command(shell: &str, command: &str) -> String {
    if shell != "powershell" {
        return command.to_string();
    }

    let normalized = command.trim().trim_end_matches('&').trim();
    let lowered = normalized.to_ascii_lowercase();
    match lowered.as_str() {
        "gnome-calculator" | "xcalc" | "kcalc" | "mate-calc" | "galculator" => {
            "Start-Process calc".to_string()
        }
        "open -a calculator" | "calc" | "start calc" => "Start-Process calc".to_string(),
        _ => command.to_string(),
    }
}

fn default_timeout(state: &AppState, tool: &ToolEntry) -> u64 {
    if state.config.timeouts.default_ms > 0 {
        state.config.timeouts.default_ms
    } else {
        tool.timeout_ms
    }
}

fn truncate_output(bytes: &[u8]) -> (String, bool) {
    if bytes.len() <= MAX_CAPTURE_BYTES {
        return (String::from_utf8_lossy(bytes).to_string(), false);
    }

    let truncated = String::from_utf8_lossy(&bytes[..MAX_CAPTURE_BYTES]).to_string();
    (
        format!(
            "{truncated}\n... [truncated {} bytes]",
            bytes.len() - MAX_CAPTURE_BYTES
        ),
        true,
    )
}

fn timeout_response(
    tool: &ToolEntry,
    shell: &str,
    command: &str,
    workdir: &str,
    duration_ms: u128,
    timeout_ms: u64,
) -> ExecuteResponse {
    let mut result = BTreeMap::new();
    result.insert("transport".into(), Value::String("local".into()));
    result.insert("host_id".into(), Value::String(String::new()));
    result.insert("remote_shell".into(), Value::String(String::new()));
    result.insert("shell".into(), Value::String(shell.to_string()));
    result.insert("command".into(), Value::String(command.to_string()));
    result.insert("workdir".into(), Value::String(workdir.to_string()));
    result.insert("stdout".into(), Value::String(String::new()));
    result.insert("stderr".into(), Value::String(String::new()));
    result.insert("timed_out".into(), Value::Bool(true));
    result.insert(
        "duration_ms".into(),
        Value::Number(serde_json::Number::from(duration_ms as u64)),
    );
    result.insert("exit_code".into(), Value::Number((-1).into()));

    ExecuteResponse {
        ok: false,
        tool: tool.name.clone(),
        result,
        summary: format!("terminal command timed out after {timeout_ms} ms"),
        truncated: false,
        error: Some(crate::models::ErrorEnvelope {
            code: "tool_timeout".into(),
            message: format!("terminal command timed out after {timeout_ms} ms"),
        }),
    }
}

fn stderr_if_any(stderr: &str, exit_code: i32) -> String {
    if stderr.trim().is_empty() {
        return format!("command exited with code {exit_code}");
    }
    stderr.trim().to_string()
}

fn single_quote(value: &str) -> String {
    format!("'{}'", value.replace('\'', "'\"'\"'"))
}

#[cfg(test)]
mod tests {
    use super::{normalize_command, single_quote};

    #[test]
    fn quotes_single_quotes_for_bash() {
        assert_eq!(single_quote("/mnt/d/O'Reilly"), "'/mnt/d/O'\"'\"'Reilly'");
    }

    #[test]
    fn normalizes_linux_calculator_commands_for_powershell() {
        assert_eq!(
            normalize_command("powershell", "gnome-calculator &"),
            "Start-Process calc"
        );
        assert_eq!(
            normalize_command("powershell", "open -a Calculator"),
            "Start-Process calc"
        );
        assert_eq!(
            normalize_command("powershell", "xcalc"),
            "Start-Process calc"
        );
        assert_eq!(
            normalize_command("powershell", "calc"),
            "Start-Process calc"
        );
        assert_eq!(
            normalize_command("powershell", "start calc"),
            "Start-Process calc"
        );
    }

    #[test]
    fn preserves_other_commands() {
        assert_eq!(normalize_command("powershell", "git status"), "git status");
        assert_eq!(
            normalize_command("wsl-bash", "gnome-calculator &"),
            "gnome-calculator &"
        );
    }
}
