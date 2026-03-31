use std::{
    collections::HashMap,
    fs,
    path::{Path, PathBuf},
    sync::{Arc, Mutex as StdMutex},
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
use tokio::sync::{Mutex, Semaphore};

use crate::{
    config::normalize_runtime_path,
    logging::now_ms,
    models::{
        AppState, ErrorEnvelope, SshDiagnosticPhase, SshHostProfile, SshRuntimeEntry,
        SshRuntimeStatusResponse, SshTestRequest, SshTestResponse, ToolEntry,
    },
    policy::timeout_with_retry,
};

const DEFAULT_SSH_POOL_LIMIT: usize = 1;
const DEFAULT_SSH_IDLE_TTL_MS: u64 = 5 * 60 * 1000;

pub type SharedSshRuntime = Arc<SshRuntimeManager>;

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
    pub connection_reused: bool,
    pub queue_wait_ms: u64,
}

#[derive(Debug, Clone)]
pub struct SshExecutionError {
    pub code: String,
    pub message: String,
    pub failure_phase: String,
    pub timed_out: bool,
    pub connection_reused: bool,
    pub queue_wait_ms: u64,
}

#[derive(Debug, Clone, Copy)]
struct TimeoutBudget {
    queue_ms: u64,
    connect_ms: u64,
    auth_ms: u64,
    exec_ms: u64,
}

struct ConnectedSession {
    handle: Handle<CaptureHandler>,
    fingerprint: String,
}

struct PooledSession {
    handle: Handle<CaptureHandler>,
    connected_at_ms: u64,
    last_used_at_ms: u64,
}

#[derive(Default)]
struct HostRuntimeMetrics {
    active_commands: u32,
    queued_commands: u32,
    last_connect_error: String,
    last_exec_error: String,
    last_connected_at: u64,
    last_used_at: u64,
}

struct HostRuntime {
    pool_limit: usize,
    semaphore: Arc<Semaphore>,
    session: Mutex<Option<PooledSession>>,
    metrics: Mutex<HostRuntimeMetrics>,
}

impl HostRuntime {
    fn new(pool_limit: usize) -> Self {
        Self {
            pool_limit,
            semaphore: Arc::new(Semaphore::new(pool_limit)),
            session: Mutex::new(None),
            metrics: Mutex::new(HostRuntimeMetrics::default()),
        }
    }

    async fn record_queue_delta(&self, delta: i32) {
        let mut metrics = self.metrics.lock().await;
        if delta.is_negative() {
            metrics.queued_commands = metrics.queued_commands.saturating_sub(delta.unsigned_abs());
        } else {
            metrics.queued_commands = metrics.queued_commands.saturating_add(delta as u32);
        }
    }

    async fn record_active_delta(&self, delta: i32) {
        let mut metrics = self.metrics.lock().await;
        if delta.is_negative() {
            metrics.active_commands = metrics.active_commands.saturating_sub(delta.unsigned_abs());
        } else {
            metrics.active_commands = metrics.active_commands.saturating_add(delta as u32);
        }
    }

    async fn record_connect_success(&self, at_ms: u64) {
        let mut metrics = self.metrics.lock().await;
        metrics.last_connected_at = at_ms;
        metrics.last_connect_error.clear();
    }

    async fn record_exec_success(&self, at_ms: u64) {
        let mut metrics = self.metrics.lock().await;
        metrics.last_used_at = at_ms;
        metrics.last_exec_error.clear();
    }

    async fn record_connect_error(&self, message: &str) {
        let mut metrics = self.metrics.lock().await;
        metrics.last_connect_error = message.to_string();
    }

    async fn record_exec_error(&self, message: &str) {
        let mut metrics = self.metrics.lock().await;
        metrics.last_exec_error = message.to_string();
    }

    async fn snapshot(&self, host: &SshHostProfile) -> SshRuntimeEntry {
        let pooled_connections = {
            let session = self.session.lock().await;
            if session
                .as_ref()
                .is_some_and(|item| !pooled_session_is_stale(item))
            {
                1
            } else {
                0
            }
        };
        let metrics = self.metrics.lock().await;
        SshRuntimeEntry {
            host_id: host.id.clone(),
            label: host.label.clone(),
            enabled: host.enabled,
            pool_limit: self.pool_limit as u32,
            pooled_connections,
            active_commands: metrics.active_commands,
            queued_commands: metrics.queued_commands,
            last_connect_error: metrics.last_connect_error.clone(),
            last_exec_error: metrics.last_exec_error.clone(),
            last_connected_at: metrics.last_connected_at,
            last_used_at: metrics.last_used_at,
        }
    }
}

#[derive(Default)]
pub struct SshRuntimeManager {
    hosts: Mutex<HashMap<String, Arc<HostRuntime>>>,
}

impl SshRuntimeManager {
    async fn runtime_for_host(&self, host_id: &str) -> Arc<HostRuntime> {
        let key = host_id.trim().to_ascii_lowercase();
        let mut hosts = self.hosts.lock().await;
        hosts
            .entry(key)
            .or_insert_with(|| Arc::new(HostRuntime::new(DEFAULT_SSH_POOL_LIMIT)))
            .clone()
    }

    async fn snapshot(&self, configured_hosts: &[SshHostProfile]) -> SshRuntimeStatusResponse {
        let mut entries = Vec::with_capacity(configured_hosts.len());
        let mut active_connection_count = 0usize;
        for host in configured_hosts {
            let runtime = self.runtime_for_host(&host.id).await;
            let entry = runtime.snapshot(host).await;
            active_connection_count += entry.pooled_connections as usize;
            entries.push(entry);
        }
        SshRuntimeStatusResponse {
            configured_host_count: configured_hosts.len(),
            active_connection_count,
            hosts: entries,
        }
    }
}

pub fn new_runtime() -> SharedSshRuntime {
    Arc::new(SshRuntimeManager::default())
}

#[derive(Clone, Default)]
struct FingerprintCapture {
    value: Arc<StdMutex<Option<String>>>,
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

pub async fn ssh_runtime_status(state: &AppState) -> Result<SshRuntimeStatusResponse> {
    let hosts = load_ssh_hosts(&state.ssh_hosts_path)?
        .into_iter()
        .map(|host| sanitize_host_profile(&host))
        .collect::<Result<Vec<_>>>()?;
    Ok(state.ssh_runtime.snapshot(&hosts).await)
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
    let budget = split_timeout_budget(timeout_ms.max(1));
    let mut phases = Vec::new();

    let mut connected = match connect_session(&host, budget.connect_ms).await {
        Ok(session) => {
            phases.push(SshDiagnosticPhase {
                phase: "connect".into(),
                status: "ok".into(),
                message: format!("connected to {}:{}", host.host, host.port),
            });
            session
        }
        Err(err) => {
            phases.push(SshDiagnosticPhase {
                phase: "connect".into(),
                status: "failed".into(),
                message: err.message.clone(),
            });
            return Ok(SshTestResponse {
                ok: false,
                summary: err.message.clone(),
                host_key_status: "unknown".into(),
                host_key_fingerprint: String::new(),
                phases,
                suggested_fix: "检查主机地址、端口和网络连通性。".into(),
                error: Some(ErrorEnvelope {
                    code: err.code,
                    message: err.message,
                }),
            });
        }
    };

    match assess_host_key(&host, &connected.fingerprint, false) {
        Ok(HostKeyAssessment::Trusted) => phases.push(SshDiagnosticPhase {
            phase: "host_key".into(),
            status: "ok".into(),
            message: format!("host key verified: {}", connected.fingerprint),
        }),
        Ok(HostKeyAssessment::NeedsConfirmation) => {
            let _ = disconnect_quietly(&mut connected.handle).await;
            phases.push(SshDiagnosticPhase {
                phase: "host_key".into(),
                status: "pending".into(),
                message: format!("host key confirmation required: {}", connected.fingerprint),
            });
            phases.push(skipped_phase(
                "auth",
                "等待确认 host key 后再继续认证测试。",
            ));
            phases.push(skipped_phase(
                "shell",
                "等待确认 host key 后再继续 shell 测试。",
            ));
            phases.push(skipped_phase(
                "workdir",
                "等待确认 host key 后再继续工作目录测试。",
            ));
            return Ok(SshTestResponse {
                ok: false,
                summary: "ssh host key confirmation required".into(),
                host_key_status: "unknown".into(),
                host_key_fingerprint: connected.fingerprint,
                phases,
                suggested_fix: "先确认 host key，再重新测试。".into(),
                error: Some(ErrorEnvelope {
                    code: "host_key_unconfirmed".into(),
                    message: "ssh host key confirmation required".into(),
                }),
            });
        }
        Err(err) => {
            let _ = disconnect_quietly(&mut connected.handle).await;
            phases.push(SshDiagnosticPhase {
                phase: "host_key".into(),
                status: "failed".into(),
                message: err.message.clone(),
            });
            phases.push(skipped_phase("auth", "host key 校验失败，认证测试已跳过。"));
            phases.push(skipped_phase(
                "shell",
                "host key 校验失败，shell 测试已跳过。",
            ));
            phases.push(skipped_phase(
                "workdir",
                "host key 校验失败，工作目录测试已跳过。",
            ));
            return Ok(SshTestResponse {
                ok: false,
                summary: err.message.clone(),
                host_key_status: host.host_key_status.clone(),
                host_key_fingerprint: connected.fingerprint,
                phases,
                suggested_fix: "校对已保存的 host key 指纹，必要时重新确认。".into(),
                error: Some(ErrorEnvelope {
                    code: err.code,
                    message: err.message,
                }),
            });
        }
    }

    if let Err(err) =
        authenticate_session_with_timeout(&mut connected.handle, &host, budget.auth_ms).await
    {
        let _ = disconnect_quietly(&mut connected.handle).await;
        phases.push(SshDiagnosticPhase {
            phase: "auth".into(),
            status: "failed".into(),
            message: err.message.clone(),
        });
        phases.push(skipped_phase("shell", "认证失败，shell 测试已跳过。"));
        phases.push(skipped_phase("workdir", "认证失败，工作目录测试已跳过。"));
        return Ok(SshTestResponse {
            ok: false,
            summary: err.message.clone(),
            host_key_status: host.host_key_status.clone(),
            host_key_fingerprint: connected.fingerprint,
            phases,
            suggested_fix: "检查用户名、密码或私钥配置是否正确。".into(),
            error: Some(ErrorEnvelope {
                code: err.code,
                message: err.message,
            }),
        });
    }
    phases.push(SshDiagnosticPhase {
        phase: "auth".into(),
        status: "ok".into(),
        message: "authentication succeeded".into(),
    });

    if let Err(err) = execute_session_command(
        &mut connected.handle,
        &build_shell_probe_command(&host.remote_shell_default),
        budget.exec_ms,
    )
    .await
    {
        let _ = disconnect_quietly(&mut connected.handle).await;
        phases.push(SshDiagnosticPhase {
            phase: "shell".into(),
            status: "failed".into(),
            message: err.message.clone(),
        });
        phases.push(skipped_phase(
            "workdir",
            "shell 启动失败，工作目录测试已跳过。",
        ));
        return Ok(SshTestResponse {
            ok: false,
            summary: err.message.clone(),
            host_key_status: host.host_key_status.clone(),
            host_key_fingerprint: connected.fingerprint,
            phases,
            suggested_fix: "确认远程 shell 类型是否正确，并检查目标机是否可启动该 shell。".into(),
            error: Some(ErrorEnvelope {
                code: err.code,
                message: err.message,
            }),
        });
    }
    phases.push(SshDiagnosticPhase {
        phase: "shell".into(),
        status: "ok".into(),
        message: format!("{} shell probe succeeded", host.remote_shell_default),
    });

    let workdir = host
        .default_workdir
        .clone()
        .or_else(|| host.allowed_paths.first().cloned())
        .unwrap_or_default();
    if workdir.trim().is_empty() {
        phases.push(skipped_phase(
            "workdir",
            "未配置 default_workdir，工作目录测试已跳过。",
        ));
    } else {
        let workdir_command = build_remote_command(
            &host.remote_shell_default,
            &workdir,
            &shell_probe_body(&host.remote_shell_default),
        );
        if let Err(err) =
            execute_session_command(&mut connected.handle, &workdir_command, budget.exec_ms).await
        {
            let _ = disconnect_quietly(&mut connected.handle).await;
            phases.push(SshDiagnosticPhase {
                phase: "workdir".into(),
                status: "failed".into(),
                message: err.message.clone(),
            });
            return Ok(SshTestResponse {
                ok: false,
                summary: err.message.clone(),
                host_key_status: host.host_key_status.clone(),
                host_key_fingerprint: connected.fingerprint,
                phases,
                suggested_fix:
                    "检查 default_workdir 和 allowed_paths 是否指向远程主机上真实可访问的路径。"
                        .into(),
                error: Some(ErrorEnvelope {
                    code: "workdir_out_of_bounds".into(),
                    message: err.message,
                }),
            });
        }
        phases.push(SshDiagnosticPhase {
            phase: "workdir".into(),
            status: "ok".into(),
            message: format!("workdir is reachable: {workdir}"),
        });
    }

    let _ = disconnect_quietly(&mut connected.handle).await;
    Ok(SshTestResponse {
        ok: true,
        summary: "ssh host diagnostics passed".into(),
        host_key_status: host.host_key_status.clone(),
        host_key_fingerprint: connected.fingerprint,
        phases,
        suggested_fix: String::new(),
        error: None,
    })
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
) -> std::result::Result<SshCommandOutput, SshExecutionError> {
    let effective = effective_ssh_settings(state, tool, arguments).map_err(|(code, message)| {
        SshExecutionError {
            code,
            message,
            failure_phase: "workdir".into(),
            timed_out: false,
            connection_reused: false,
            queue_wait_ms: 0,
        }
    })?;
    let effective_timeout = timeout_with_retry(
        effective.timeout_ms,
        tool.retry.max_attempts,
        tool.retry.backoff_ms,
    );

    run_remote_command(
        state,
        &effective.host,
        &effective.remote_shell,
        &effective.workdir,
        command,
        effective_timeout,
    )
    .await
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
    state: &AppState,
    host: &SshHostProfile,
    remote_shell: &str,
    workdir: &str,
    command: &str,
    timeout_ms: u64,
) -> std::result::Result<SshCommandOutput, SshExecutionError> {
    let runtime = state.ssh_runtime.runtime_for_host(&host.id).await;
    let budget = split_timeout_budget(timeout_ms.max(1));
    let remote_command = build_remote_command(remote_shell, workdir, command);
    let permit = acquire_host_slot(runtime.clone(), budget.queue_ms).await?;
    let queue_wait_ms = permit.1;
    runtime.record_active_delta(1).await;

    let result = run_remote_command_inner(
        runtime.clone(),
        host,
        &remote_command,
        budget,
        queue_wait_ms,
    )
    .await;

    runtime.record_active_delta(-1).await;
    drop(permit.0);
    result
}

async fn run_remote_command_inner(
    runtime: Arc<HostRuntime>,
    host: &SshHostProfile,
    remote_command: &str,
    budget: TimeoutBudget,
    queue_wait_ms: u64,
) -> std::result::Result<SshCommandOutput, SshExecutionError> {
    let mut session_slot = runtime.session.lock().await;
    drop_stale_session(&mut session_slot).await;

    if let Some(existing) = session_slot.as_mut() {
        match execute_session_command(&mut existing.handle, remote_command, budget.exec_ms).await {
            Ok(mut output) => {
                let used_at = now_ms() as u64;
                existing.last_used_at_ms = used_at;
                runtime.record_exec_success(used_at).await;
                output.connection_reused = true;
                output.queue_wait_ms = queue_wait_ms;
                return Ok(output);
            }
            Err(err) if is_retryable_reuse_error(&err) => {
                runtime.record_exec_error(&err.message).await;
                drop_session(session_slot.take()).await;
            }
            Err(mut err) => {
                runtime.record_exec_error(&err.message).await;
                drop_session(session_slot.take()).await;
                err.connection_reused = true;
                err.queue_wait_ms = queue_wait_ms;
                return Err(err);
            }
        }
    }

    let mut pooled = connect_and_auth_new_session(host, runtime.clone(), budget).await?;
    match execute_session_command(&mut pooled.handle, remote_command, budget.exec_ms).await {
        Ok(mut output) => {
            let used_at = now_ms() as u64;
            pooled.last_used_at_ms = used_at;
            runtime.record_exec_success(used_at).await;
            output.connection_reused = false;
            output.queue_wait_ms = queue_wait_ms;
            *session_slot = Some(pooled);
            Ok(output)
        }
        Err(mut err) => {
            runtime.record_exec_error(&err.message).await;
            drop_session(Some(pooled)).await;
            err.connection_reused = false;
            err.queue_wait_ms = queue_wait_ms;
            Err(err)
        }
    }
}

async fn connect_and_auth_new_session(
    host: &SshHostProfile,
    runtime: Arc<HostRuntime>,
    budget: TimeoutBudget,
) -> std::result::Result<PooledSession, SshExecutionError> {
    let mut connected = connect_session(host, budget.connect_ms).await?;
    match assess_host_key(host, &connected.fingerprint, true) {
        Ok(_) => {}
        Err(err) => {
            runtime.record_connect_error(&err.message).await;
            let _ = disconnect_quietly(&mut connected.handle).await;
            return Err(err);
        }
    }
    authenticate_session_with_timeout(&mut connected.handle, host, budget.auth_ms).await?;
    let connected_at_ms = now_ms() as u64;
    runtime.record_connect_success(connected_at_ms).await;
    Ok(PooledSession {
        handle: connected.handle,
        connected_at_ms,
        last_used_at_ms: connected_at_ms,
    })
}

async fn acquire_host_slot(
    runtime: Arc<HostRuntime>,
    queue_timeout_ms: u64,
) -> std::result::Result<(tokio::sync::OwnedSemaphorePermit, u64), SshExecutionError> {
    runtime.record_queue_delta(1).await;
    let started = now_ms() as u64;
    let permit = match tokio::time::timeout(
        Duration::from_millis(queue_timeout_ms.max(1)),
        runtime.semaphore.clone().acquire_owned(),
    )
    .await
    {
        Ok(Ok(permit)) => permit,
        Ok(Err(_)) => {
            runtime.record_queue_delta(-1).await;
            return Err(SshExecutionError {
                code: "ssh_exec_failed".into(),
                message: "ssh execution queue is unavailable".into(),
                failure_phase: "exec".into(),
                timed_out: false,
                connection_reused: false,
                queue_wait_ms: now_ms() as u64 - started,
            });
        }
        Err(_) => {
            runtime.record_queue_delta(-1).await;
            return Err(SshExecutionError {
                code: "ssh_exec_timeout".into(),
                message: format!("ssh execution queue timed out after {queue_timeout_ms} ms"),
                failure_phase: "exec".into(),
                timed_out: true,
                connection_reused: false,
                queue_wait_ms: now_ms() as u64 - started,
            });
        }
    };
    runtime.record_queue_delta(-1).await;
    Ok((permit, now_ms() as u64 - started))
}

async fn connect_session(
    host: &SshHostProfile,
    timeout_ms: u64,
) -> std::result::Result<ConnectedSession, SshExecutionError> {
    let fingerprint = FingerprintCapture::default();
    let handler = CaptureHandler {
        fingerprint: fingerprint.clone(),
    };
    let config = Arc::new(client::Config {
        inactivity_timeout: Some(Duration::from_millis(timeout_ms.max(1))),
        ..Default::default()
    });
    let address = format!("{}:{}", host.host, host.port);
    let session = match tokio::time::timeout(
        Duration::from_millis(timeout_ms.max(1)),
        client::connect(config, address.as_str(), handler),
    )
    .await
    {
        Ok(Ok(session)) => session,
        Ok(Err(err)) => {
            return Err(SshExecutionError {
                code: "ssh_connect_failed".into(),
                message: format!("connect to {}:{} failed: {err}", host.host, host.port),
                failure_phase: "connect".into(),
                timed_out: false,
                connection_reused: false,
                queue_wait_ms: 0,
            });
        }
        Err(_) => {
            return Err(SshExecutionError {
                code: "ssh_connect_timeout".into(),
                message: format!("ssh connect timed out after {timeout_ms} ms"),
                failure_phase: "connect".into(),
                timed_out: true,
                connection_reused: false,
                queue_wait_ms: 0,
            });
        }
    };
    let actual_fingerprint = fingerprint.get().ok_or_else(|| SshExecutionError {
        code: "ssh_connect_failed".into(),
        message: "ssh server did not present a host key".into(),
        failure_phase: "connect".into(),
        timed_out: false,
        connection_reused: false,
        queue_wait_ms: 0,
    })?;
    Ok(ConnectedSession {
        handle: session,
        fingerprint: actual_fingerprint,
    })
}

async fn authenticate_session_with_timeout(
    session: &mut Handle<CaptureHandler>,
    host: &SshHostProfile,
    timeout_ms: u64,
) -> std::result::Result<(), SshExecutionError> {
    let auth = tokio::time::timeout(Duration::from_millis(timeout_ms.max(1)), async {
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
        Ok::<(), anyhow::Error>(())
    })
    .await;

    match auth {
        Ok(Ok(())) => Ok(()),
        Ok(Err(err)) => Err(SshExecutionError {
            code: "ssh_auth_failed".into(),
            message: err.to_string(),
            failure_phase: "auth".into(),
            timed_out: false,
            connection_reused: false,
            queue_wait_ms: 0,
        }),
        Err(_) => Err(SshExecutionError {
            code: "ssh_auth_failed".into(),
            message: format!("ssh authentication timed out after {timeout_ms} ms"),
            failure_phase: "auth".into(),
            timed_out: true,
            connection_reused: false,
            queue_wait_ms: 0,
        }),
    }
}

async fn execute_session_command(
    session: &mut Handle<CaptureHandler>,
    command: &str,
    timeout_ms: u64,
) -> std::result::Result<SshCommandOutput, SshExecutionError> {
    execute_session_command_with_timeout(session, command, timeout_ms).await
}

async fn execute_session_command_with_timeout(
    session: &mut Handle<CaptureHandler>,
    command: &str,
    timeout_ms: u64,
) -> std::result::Result<SshCommandOutput, SshExecutionError> {
    let execution = tokio::time::timeout(Duration::from_millis(timeout_ms.max(1)), async {
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

        Ok::<SshCommandOutput, anyhow::Error>(SshCommandOutput {
            exit_code: exit_code.unwrap_or(0),
            stdout,
            stderr,
            connection_reused: false,
            queue_wait_ms: 0,
        })
    })
    .await;

    match execution {
        Ok(Ok(output)) => Ok(output),
        Ok(Err(err)) => Err(SshExecutionError {
            code: "ssh_exec_failed".into(),
            message: err.to_string(),
            failure_phase: "exec".into(),
            timed_out: false,
            connection_reused: false,
            queue_wait_ms: 0,
        }),
        Err(_) => Err(SshExecutionError {
            code: "ssh_exec_timeout".into(),
            message: format!("ssh command timed out after {timeout_ms} ms"),
            failure_phase: "exec".into(),
            timed_out: true,
            connection_reused: false,
            queue_wait_ms: 0,
        }),
    }
}

async fn disconnect_quietly(session: &mut Handle<CaptureHandler>) -> Result<()> {
    session
        .disconnect(Disconnect::ByApplication, "", "English")
        .await
        .context("disconnect ssh session")
}

async fn drop_session(session: Option<PooledSession>) {
    if let Some(mut session) = session {
        let _ = disconnect_quietly(&mut session.handle).await;
    }
}

async fn drop_stale_session(session_slot: &mut Option<PooledSession>) {
    if session_slot.as_ref().is_some_and(pooled_session_is_stale) {
        drop_session(session_slot.take()).await;
    }
}

fn pooled_session_is_stale(session: &PooledSession) -> bool {
    let now = now_ms() as u64;
    now.saturating_sub(session.last_used_at_ms) > DEFAULT_SSH_IDLE_TTL_MS
        || now.saturating_sub(session.connected_at_ms) > DEFAULT_SSH_IDLE_TTL_MS * 3
}

fn assess_host_key(
    host: &SshHostProfile,
    actual_fingerprint: &str,
    require_trusted_key: bool,
) -> std::result::Result<HostKeyAssessment, SshExecutionError> {
    if host.host_key_status != "trusted"
        || host
            .host_key_fingerprint
            .as_deref()
            .unwrap_or("")
            .trim()
            .is_empty()
    {
        if require_trusted_key {
            return Err(SshExecutionError {
                code: "host_key_unconfirmed".into(),
                message: format!("ssh host key confirmation required: {actual_fingerprint}"),
                failure_phase: "host_key".into(),
                timed_out: false,
                connection_reused: false,
                queue_wait_ms: 0,
            });
        }
        return Ok(HostKeyAssessment::NeedsConfirmation);
    }

    ensure_host_key(host, actual_fingerprint).map_err(|err| SshExecutionError {
        code: "host_key_mismatch".into(),
        message: err.to_string(),
        failure_phase: "host_key".into(),
        timed_out: false,
        connection_reused: false,
        queue_wait_ms: 0,
    })?;
    Ok(HostKeyAssessment::Trusted)
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

fn build_shell_probe_command(remote_shell: &str) -> String {
    match remote_shell {
        "powershell" => {
            "powershell -NoProfile -NonInteractive -Command \"$PSVersionTable.PSVersion | Out-Null; Write-Output ready\"".into()
        }
        _ => "bash -lc 'printf ready'".into(),
    }
}

fn shell_probe_body(remote_shell: &str) -> String {
    match remote_shell {
        "powershell" => "Write-Output ready".into(),
        _ => "printf ready".into(),
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

fn split_timeout_budget(timeout_ms: u64) -> TimeoutBudget {
    let total = timeout_ms.max(1_500);
    let queue_ms = (total / 6).clamp(250, 1_500);
    let remaining = total.saturating_sub(queue_ms).max(900);
    let connect_ms = (remaining / 4).clamp(250, remaining.saturating_sub(500));
    let remaining_after_connect = remaining.saturating_sub(connect_ms);
    let auth_ms =
        (remaining_after_connect / 3).clamp(250, remaining_after_connect.saturating_sub(250));
    let exec_ms = remaining_after_connect.saturating_sub(auth_ms).max(250);
    TimeoutBudget {
        queue_ms,
        connect_ms,
        auth_ms,
        exec_ms,
    }
}

fn skipped_phase(phase: &str, message: &str) -> SshDiagnosticPhase {
    SshDiagnosticPhase {
        phase: phase.into(),
        status: "skipped".into(),
        message: message.into(),
    }
}

fn is_retryable_reuse_error(err: &SshExecutionError) -> bool {
    err.code == "ssh_exec_failed" && !err.timed_out
}

enum HostKeyAssessment {
    Trusted,
    NeedsConfirmation,
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
