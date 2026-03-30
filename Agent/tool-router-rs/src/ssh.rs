use std::{
    fs,
    path::{Path, PathBuf},
    sync::{Arc, Mutex},
    time::Duration,
};

use anyhow::{Context, Result, anyhow};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use russh::{
    ChannelMsg, Disconnect,
    client::{self, Handle},
    keys::{self, HashAlg, PrivateKeyWithHashAlg},
};
use serde_json::Value;

use crate::{
    config::normalize_runtime_path,
    models::{AppState, ErrorEnvelope, SshHostProfile, SshTestRequest, SshTestResponse, ToolEntry},
    policy::timeout_with_retry,
};

#[derive(Debug, Clone)]
pub struct EffectiveSshSettings {
    pub host: SshHostProfile,
    pub remote_shell: String,
    pub workdir: String,
    pub timeout_ms: u64,
}

#[derive(Debug)]
pub struct SshCommandOutput {
    pub exit_code: i32,
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
}

#[derive(Clone, Default)]
struct FingerprintCapture {
    value: Arc<Mutex<Option<String>>>,
}

impl FingerprintCapture {
    fn store(&self, fingerprint: String) {
        match self.value.lock() {
            Ok(mut slot) => *slot = Some(fingerprint),
            Err(poisoned) => {
                let mut slot = poisoned.into_inner();
                *slot = Some(fingerprint);
            }
        }
    }

    fn get(&self) -> Option<String> {
        match self.value.lock() {
            Ok(slot) => slot.clone(),
            Err(poisoned) => poisoned.into_inner().clone(),
        }
    }
}

#[derive(Clone, Default)]
struct CaptureHandler {
    fingerprint: FingerprintCapture,
}

impl client::Handler for CaptureHandler {
    type Error = russh::Error;

    async fn check_server_key(
        &mut self,
        server_public_key: &keys::PublicKey,
    ) -> Result<bool, Self::Error> {
        let fingerprint = server_public_key.fingerprint(HashAlg::Sha256).to_string();
        self.fingerprint.store(fingerprint);
        Ok(true)
    }
}

enum ProbeOutcome {
    PendingHostKey {
        fingerprint: String,
    },
    Ready {
        fingerprint: String,
        session: Handle<CaptureHandler>,
    },
}

pub fn load_ssh_hosts(path: &str) -> Result<Vec<SshHostProfile>> {
    let data = match fs::read_to_string(path) {
        Ok(data) => data,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(Vec::new()),
        Err(err) => return Err(err).with_context(|| format!("read ssh hosts {path}")),
    };

    parse_hosts_json(&data).with_context(|| format!("parse ssh hosts {path}"))
}

pub fn load_ssh_host(path: &str, host_id: &str) -> Result<Option<SshHostProfile>> {
    Ok(load_ssh_hosts(path)?
        .into_iter()
        .find(|host| host.id.eq_ignore_ascii_case(host_id)))
}

pub async fn test_ssh_host_request(
    state: &AppState,
    req: &SshTestRequest,
    timeout_ms: u64,
) -> Result<SshTestResponse> {
    let _ = state;
    test_ssh_host(&req.host, timeout_ms.max(1)).await
}

pub async fn test_ssh_host(host: &SshHostProfile, timeout_ms: u64) -> Result<SshTestResponse> {
    let host = sanitize_host_profile(host)?;
    let effective_timeout = timeout_ms.max(1);

    let response = match probe_host(&host, effective_timeout, false).await? {
        ProbeOutcome::PendingHostKey { fingerprint } => SshTestResponse {
            ok: false,
            summary: "ssh host key confirmation required".into(),
            host_key_status: "unknown".into(),
            host_key_fingerprint: fingerprint.clone(),
            error: Some(ErrorEnvelope {
                code: "host_key_unconfirmed".into(),
                message: format!("ssh host key confirmation required: {fingerprint}"),
            }),
        },
        ProbeOutcome::Ready {
            fingerprint,
            mut session,
        } => {
            let _ = disconnect_quietly(&mut session).await;
            SshTestResponse {
                ok: true,
                summary: "ssh host connection succeeded".into(),
                host_key_status: "trusted".into(),
                host_key_fingerprint: fingerprint,
                error: None,
            }
        }
    };

    Ok(response)
}

pub fn effective_ssh_settings(
    state: &AppState,
    tool: &ToolEntry,
    arguments: &Value,
) -> Result<EffectiveSshSettings, (String, String)> {
    let host_id = arguments
        .get("host_id")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .ok_or_else(|| {
            (
                "invalid_arguments".into(),
                "host_id is required for ssh transport".into(),
            )
        })?;

    let mut host = load_ssh_host(&state.ssh_hosts_path, host_id)
        .map_err(|err| ("config_error".into(), err.to_string()))?
        .ok_or_else(|| {
            (
                "host_not_found".into(),
                format!("unknown ssh host: {host_id}"),
            )
        })?;
    host = sanitize_host_profile(&host)
        .map_err(|err| ("invalid_arguments".into(), err.to_string()))?;
    if !host.enabled {
        return Err((
            "host_disabled".into(),
            format!("ssh host is disabled: {host_id}"),
        ));
    }

    let remote_shell = arguments
        .get("remote_shell")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or(host.remote_shell_default.as_str())
        .to_string();
    if remote_shell != "bash" && remote_shell != "powershell" {
        return Err((
            "invalid_arguments".into(),
            format!("unsupported remote_shell: {remote_shell}"),
        ));
    }

    let workdir = arguments
        .get("workdir")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(str::to_string)
        .or_else(|| host.default_workdir.clone())
        .or_else(|| host.allowed_paths.first().cloned())
        .unwrap_or_else(|| tool.workdir.clone());

    let timeout_ms = arguments
        .get("timeout_ms")
        .and_then(Value::as_u64)
        .unwrap_or(tool.timeout_ms);

    Ok(EffectiveSshSettings {
        host,
        remote_shell,
        workdir,
        timeout_ms,
    })
}

pub async fn execute_ssh_command(
    state: &AppState,
    tool: &ToolEntry,
    arguments: &Value,
    command: &str,
) -> Result<SshCommandOutput, (String, String)> {
    let effective = effective_ssh_settings(state, tool, arguments)?;
    let effective_timeout = timeout_with_retry(
        effective.timeout_ms,
        tool.retry.max_attempts,
        tool.retry.backoff_ms,
    );

    run_remote_command(
        &effective.host,
        &effective.remote_shell,
        &effective.workdir,
        command,
        effective_timeout,
    )
    .await
    .map_err(map_ssh_error)
}

pub fn parse_hosts_json(data: &str) -> Result<Vec<SshHostProfile>> {
    let value: Value = serde_json::from_str(data)?;
    match value {
        Value::Array(items) => {
            serde_json::from_value(Value::Array(items)).context("decode host array")
        }
        Value::Object(mut object) => {
            if let Some(hosts) = object.remove("hosts") {
                serde_json::from_value(hosts).context("decode hosts field")
            } else {
                Err(anyhow!(
                    "ssh hosts payload must be an array or an object with a hosts field"
                ))
            }
        }
        _ => Err(anyhow!(
            "ssh hosts payload must be an array or an object with a hosts field"
        )),
    }
}

pub fn sanitize_host_profile(host: &SshHostProfile) -> Result<SshHostProfile> {
    let mut cleaned = host.clone();
    cleaned.id = cleaned.id.trim().to_string();
    cleaned.label = cleaned.label.trim().to_string();
    cleaned.host = cleaned.host.trim().to_string();
    cleaned.username = cleaned.username.trim().to_string();
    cleaned.auth_type = cleaned.auth_type.trim().to_string();
    cleaned.remote_shell_default = cleaned.remote_shell_default.trim().to_string();
    cleaned.allowed_paths = cleaned
        .allowed_paths
        .iter()
        .map(|item| item.trim())
        .filter(|item| !item.is_empty())
        .map(str::to_string)
        .collect();
    cleaned.default_workdir = cleaned
        .default_workdir
        .as_ref()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty());
    cleaned.host_key_status = cleaned.host_key_status.trim().to_string();
    cleaned.host_key_fingerprint = cleaned
        .host_key_fingerprint
        .as_ref()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty());
    cleaned.password = cleaned
        .password
        .as_ref()
        .map(|value| value.to_string())
        .filter(|value| !value.trim().is_empty());
    cleaned.private_key = cleaned
        .private_key
        .as_ref()
        .map(|value| value.to_string())
        .filter(|value| !value.trim().is_empty());
    cleaned.passphrase = cleaned
        .passphrase
        .as_ref()
        .map(|value| value.to_string())
        .filter(|value| !value.trim().is_empty());

    if cleaned.id.is_empty() {
        return Err(anyhow!("ssh host id is required"));
    }
    if cleaned.label.is_empty() {
        return Err(anyhow!("ssh host label is required"));
    }
    if cleaned.host.is_empty() {
        return Err(anyhow!("ssh host address is required"));
    }
    if cleaned.port == 0 {
        return Err(anyhow!("ssh host port must be greater than 0"));
    }
    if cleaned.username.is_empty() {
        return Err(anyhow!("ssh username is required"));
    }
    if cleaned.auth_type != "password" && cleaned.auth_type != "private_key" {
        return Err(anyhow!("ssh auth_type must be password or private_key"));
    }
    if cleaned.remote_shell_default != "bash" && cleaned.remote_shell_default != "powershell" {
        return Err(anyhow!("remote_shell_default must be bash or powershell"));
    }
    if cleaned.allowed_paths.is_empty() {
        return Err(anyhow!("at least one ssh allowed_path is required"));
    }
    match cleaned.auth_type.as_str() {
        "password" if cleaned.password.is_none() => {
            return Err(anyhow!("password is required for password auth"));
        }
        "private_key" if cleaned.private_key.is_none() => {
            return Err(anyhow!("private_key is required for private_key auth"));
        }
        _ => {}
    }
    if cleaned.host_key_status.is_empty() {
        cleaned.host_key_status = "unknown".into();
    }
    if cleaned.host_key_status != "unknown" && cleaned.host_key_status != "trusted" {
        return Err(anyhow!("host_key_status must be unknown or trusted"));
    }
    if cleaned.host_key_status == "trusted" && cleaned.host_key_fingerprint.is_none() {
        return Err(anyhow!(
            "host_key_fingerprint is required when host_key_status is trusted"
        ));
    }

    Ok(cleaned)
}

async fn run_remote_command(
    host: &SshHostProfile,
    remote_shell: &str,
    workdir: &str,
    command: &str,
    timeout_ms: u64,
) -> Result<SshCommandOutput> {
    let total_timeout = Duration::from_millis(timeout_ms.max(1));
    tokio::time::timeout(total_timeout, async {
        let mut session = match probe_host(host, timeout_ms, true).await? {
            ProbeOutcome::PendingHostKey { fingerprint } => {
                return Err(anyhow!("ssh host key confirmation required: {fingerprint}"));
            }
            ProbeOutcome::Ready { session, .. } => session,
        };

        authenticate_session(&mut session, host).await?;

        let remote_command = build_remote_command(remote_shell, workdir, command);
        let output = execute_session_command(&mut session, &remote_command).await?;
        let _ = disconnect_quietly(&mut session).await;
        Ok(output)
    })
    .await
    .map_err(|_| anyhow!("ssh command timed out after {timeout_ms} ms"))?
}

async fn probe_host(
    host: &SshHostProfile,
    timeout_ms: u64,
    require_trusted_key: bool,
) -> Result<ProbeOutcome> {
    let fingerprint = FingerprintCapture::default();
    let handler = CaptureHandler {
        fingerprint: fingerprint.clone(),
    };
    let config = Arc::new(client::Config {
        inactivity_timeout: Some(Duration::from_millis(timeout_ms.max(1))),
        ..Default::default()
    });
    let address = format!("{}:{}", host.host, host.port);
    let session = client::connect(config, address.as_str(), handler)
        .await
        .with_context(|| format!("connect to {}:{}", host.host, host.port))?;
    let actual_fingerprint = fingerprint
        .get()
        .ok_or_else(|| anyhow!("ssh server did not present a host key"))?;

    if host.host_key_status != "trusted"
        || host
            .host_key_fingerprint
            .as_deref()
            .unwrap_or("")
            .trim()
            .is_empty()
    {
        if require_trusted_key {
            return Ok(ProbeOutcome::PendingHostKey {
                fingerprint: actual_fingerprint,
            });
        }
        return Ok(ProbeOutcome::PendingHostKey {
            fingerprint: actual_fingerprint,
        });
    }

    ensure_host_key(host, &actual_fingerprint)?;
    Ok(ProbeOutcome::Ready {
        fingerprint: actual_fingerprint,
        session,
    })
}

async fn authenticate_session(
    session: &mut Handle<CaptureHandler>,
    host: &SshHostProfile,
) -> Result<()> {
    match host.auth_type.as_str() {
        "password" => {
            let auth = session
                .authenticate_password(
                    host.username.clone(),
                    host.password.as_deref().unwrap_or_default().to_string(),
                )
                .await
                .context("ssh password authentication")?;
            if !auth.success() {
                return Err(anyhow!("ssh authentication failed"));
            }
        }
        "private_key" => {
            let key = keys::decode_secret_key(
                host.private_key.as_deref().unwrap_or_default(),
                host.passphrase.as_deref(),
            )
            .context("decode ssh private key")?;
            let hash_alg = session
                .best_supported_rsa_hash()
                .await
                .context("query ssh rsa hash support")?
                .flatten();
            let auth = session
                .authenticate_publickey(
                    host.username.clone(),
                    PrivateKeyWithHashAlg::new(Arc::new(key), hash_alg),
                )
                .await
                .context("ssh private key authentication")?;
            if !auth.success() {
                return Err(anyhow!("ssh authentication failed"));
            }
        }
        other => return Err(anyhow!("unsupported ssh auth_type: {other}")),
    }

    Ok(())
}

async fn execute_session_command(
    session: &mut Handle<CaptureHandler>,
    command: &str,
) -> Result<SshCommandOutput> {
    let mut channel = session
        .channel_open_session()
        .await
        .context("open ssh session channel")?;
    channel
        .exec(true, command)
        .await
        .with_context(|| format!("exec remote command: {command}"))?;

    let mut stdout = Vec::new();
    let mut stderr = Vec::new();
    let mut exit_code = None;

    while let Some(message) = channel.wait().await {
        match message {
            ChannelMsg::Data { data } => stdout.extend_from_slice(&data),
            ChannelMsg::ExtendedData { data, .. } => stderr.extend_from_slice(&data),
            ChannelMsg::ExitStatus { exit_status } => {
                exit_code = Some(i32::try_from(exit_status).unwrap_or(i32::MAX))
            }
            ChannelMsg::ExitSignal {
                signal_name,
                error_message,
                ..
            } => {
                if !error_message.is_empty() {
                    stderr.extend_from_slice(error_message.as_bytes());
                } else {
                    stderr.extend_from_slice(
                        format!("terminated by signal {signal_name:?}").as_bytes(),
                    );
                }
                if exit_code.is_none() {
                    exit_code = Some(255);
                }
            }
            ChannelMsg::Close => break,
            _ => {}
        }
    }

    Ok(SshCommandOutput {
        exit_code: exit_code.unwrap_or(0),
        stdout,
        stderr,
    })
}

async fn disconnect_quietly(session: &mut Handle<CaptureHandler>) -> Result<()> {
    session
        .disconnect(Disconnect::ByApplication, "", "English")
        .await
        .context("disconnect ssh session")
}

fn ensure_host_key(host: &SshHostProfile, actual_fingerprint: &str) -> Result<()> {
    let expected = host
        .host_key_fingerprint
        .as_deref()
        .ok_or_else(|| anyhow!("ssh host key fingerprint is not configured"))?;
    if !expected.eq_ignore_ascii_case(actual_fingerprint) {
        return Err(anyhow!(
            "ssh host key mismatch: expected {expected}, got {actual_fingerprint}"
        ));
    }
    Ok(())
}

fn build_remote_command(remote_shell: &str, workdir: &str, command: &str) -> String {
    match remote_shell {
        "powershell" => build_powershell_remote_command(workdir, command),
        _ => build_bash_remote_command(workdir, command),
    }
}

fn build_bash_remote_command(workdir: &str, command: &str) -> String {
    format!("cd -- {} && {}", bash_single_quote(workdir), command)
}

fn build_powershell_remote_command(workdir: &str, command: &str) -> String {
    let script = format!(
        "$ErrorActionPreference = 'Stop'; Set-Location -LiteralPath '{}'; {}",
        powershell_single_quote(workdir),
        command
    );
    let utf16: Vec<u8> = script
        .encode_utf16()
        .flat_map(|value| value.to_le_bytes())
        .collect();
    let encoded = STANDARD.encode(utf16);
    format!(
        "powershell -NoProfile -NonInteractive -ExecutionPolicy Bypass -EncodedCommand {encoded}"
    )
}

fn map_ssh_error(err: anyhow::Error) -> (String, String) {
    let message = err.to_string();
    let lowered = message.to_ascii_lowercase();
    if lowered.contains("confirmation required") {
        return ("host_key_unconfirmed".into(), message);
    }
    if lowered.contains("host key mismatch") {
        return ("host_key_mismatch".into(), message);
    }
    if lowered.contains("authentication failed")
        || lowered.contains("password authentication")
        || lowered.contains("private key authentication")
    {
        return ("auth_failed".into(), message);
    }
    if lowered.contains("timed out") {
        return ("tool_timeout".into(), message);
    }
    if lowered.contains("connect to") || lowered.contains("dns") {
        return ("ssh_connect_failed".into(), message);
    }
    ("tool_exec_failed".into(), message)
}

fn bash_single_quote(value: &str) -> String {
    format!("'{}'", value.replace('\'', "'\"'\"'"))
}

fn powershell_single_quote(value: &str) -> String {
    value.replace('\'', "''")
}

pub fn is_within_remote_allowed_path(candidate: &str, allowed: &str, remote_shell: &str) -> bool {
    if remote_shell == "powershell" {
        let candidate = normalize_windows_like_path(candidate);
        let allowed = normalize_windows_like_path(allowed);
        return Path::new(&candidate).starts_with(Path::new(&allowed));
    }

    let candidate = normalize_posix_like_path(candidate);
    let allowed = normalize_posix_like_path(allowed);
    Path::new(&candidate).starts_with(Path::new(&allowed))
}

fn normalize_windows_like_path(path: &str) -> String {
    let normalized = normalize_runtime_path(path).replace('\\', "/");
    let bytes = normalized.as_bytes();
    if bytes.len() >= 2 && bytes[1] == b':' {
        let drive = normalized[0..1].to_ascii_uppercase();
        return format!("{}{}", drive, &normalized[1..]).to_ascii_lowercase();
    }
    normalized.to_ascii_lowercase()
}

fn normalize_posix_like_path(path: &str) -> String {
    let normalized = normalize_runtime_path(path);
    let clean = PathBuf::from(&normalized);
    clean.to_string_lossy().replace('\\', "/")
}

#[cfg(test)]
mod tests {
    use super::{
        build_bash_remote_command, build_powershell_remote_command, is_within_remote_allowed_path,
        parse_hosts_json, powershell_single_quote,
    };

    #[test]
    fn parses_host_array_payload() {
        let hosts = parse_hosts_json(
            r#"[{"id":"box-a","label":"Box A","enabled":true,"host":"10.0.0.8","port":22,"username":"ctf","auth_type":"password","password":"pw","remote_shell_default":"bash","allowed_paths":["/home/ctf"],"host_key_status":"unknown"}]"#,
        )
        .expect("parse host array");
        assert_eq!(hosts.len(), 1);
        assert_eq!(hosts[0].id, "box-a");
    }

    #[test]
    fn detects_remote_path_membership() {
        assert!(is_within_remote_allowed_path(
            "/home/ctf/project",
            "/home/ctf",
            "bash"
        ));
        assert!(is_within_remote_allowed_path(
            "C:/Users/ctf/project",
            "C:/Users/ctf",
            "powershell"
        ));
    }

    #[test]
    fn quotes_powershell_single_quotes() {
        assert_eq!(powershell_single_quote("D:/O'Reilly"), "D:/O''Reilly");
    }

    #[test]
    fn builds_remote_commands() {
        assert!(build_bash_remote_command("/tmp/app", "pwd").contains("cd -- '/tmp/app'"));
        assert!(
            build_powershell_remote_command("C:/Work", "Get-Location").contains("-EncodedCommand")
        );
    }
}
