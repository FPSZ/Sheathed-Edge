use std::{
    fs::{self, OpenOptions},
    path::{Path, PathBuf},
    process::Stdio,
};

use anyhow::{Context, Result, anyhow};
use tokio::process::Command;

use crate::{
    config,
    models::{ManagedProcess, ModelProfile, SharedState, StatusResponse},
};

pub async fn status(state: &SharedState) -> StatusResponse {
    let active_profile_id = state.active_profile_id.lock().await.clone();
    let active_profile = {
        let profiles = state.profiles.lock().await;
        profiles.get(&active_profile_id).cloned()
    };
    let health_host = health_check_host(&state.llama.listen_host);
    let endpoint = format!("http://{}:{}/health", health_host, state.llama.listen_port);

    let running = match state.http_client.get(&endpoint).send().await {
        Ok(_) => true, // any HTTP response means the server is alive (503 = busy but running)
        Err(_) => false,
    };

    let process = state.process.lock().await;
    let pid = process.as_ref().and_then(|managed| managed.child.id());

    StatusResponse {
        running,
        managed: process.is_some(),
        pid,
        active_profile_id,
        message: if running {
            "llama-server is reachable".into()
        } else {
            "llama-server is not reachable".into()
        },
        model_path: active_profile.map(|p| p.model_path).unwrap_or_default(),
    }
}

async fn port_in_use(listen_host: &str, port: u16) -> bool {
    let addr = format!("{}:{}", health_check_host(listen_host), port);
    tokio::net::TcpStream::connect(addr).await.is_ok()
}

fn health_check_host(listen_host: &str) -> &str {
    match listen_host.trim() {
        "" | "0.0.0.0" | "::" | "[::]" => "127.0.0.1",
        other => other,
    }
}

pub async fn start(state: &SharedState) -> Result<StatusResponse> {
    let current = status(state).await;
    if current.running {
        return Ok(current);
    }

    // Health check failed, but the port may still be occupied by an orphan process
    // (e.g. host-control was restarted and lost its managed reference).
    // Guard against spawning a duplicate by probing the TCP port directly.
    if port_in_use(&state.llama.listen_host, state.llama.listen_port).await {
        return Err(anyhow!(
            "port {} is occupied by an unmanaged process; kill it before starting llama-server",
            state.llama.listen_port
        ));
    }

    let profile_id = state.active_profile_id.lock().await.clone();
    let profile = {
        let profiles = state.profiles.lock().await;
        profiles
            .get(&profile_id)
            .cloned()
            .ok_or_else(|| anyhow!("active profile not found: {profile_id}"))?
    };
    if !profile.enabled {
        return Err(anyhow!("active profile is disabled: {profile_id}"));
    }

    let binary = resolve_binary(state)?;
    fs::create_dir_all(&state.llama.log_dir)?;

    let stdout_path = Path::new(&state.llama.log_dir).join("stdout.log");
    let stderr_path = Path::new(&state.llama.log_dir).join("stderr.log");
    let stdout = OpenOptions::new()
        .create(true)
        .append(true)
        .open(stdout_path)?;
    let stderr = OpenOptions::new()
        .create(true)
        .append(true)
        .open(stderr_path)?;

    let mut args = build_args(&state.llama, &profile);

    let child = Command::new(binary)
        .args(args.drain(..))
        .stdout(Stdio::from(stdout))
        .stderr(Stdio::from(stderr))
        .spawn()
        .context("spawn llama-server")?;

    let pid = child.id();
    *state.process.lock().await = Some(ManagedProcess { child });

    Ok(StatusResponse {
        running: true,
        managed: true,
        pid,
        active_profile_id: profile.id.clone(),
        message: "llama-server started".into(),
        model_path: profile.model_path,
    })
}

pub async fn stop(state: &SharedState) -> Result<StatusResponse> {
    if let Some(mut managed) = state.process.lock().await.take() {
        managed.child.kill().await.ok();
        managed.child.wait().await.ok();
    }

    // Also kill any process occupying the port (handles unmanaged / orphan processes).
    kill_by_port(state.llama.listen_port).await;

    // Brief pause so the OS releases the port before the caller tries to rebind it.
    tokio::time::sleep(tokio::time::Duration::from_millis(800)).await;

    let current = status(state).await;
    Ok(StatusResponse {
        running: false,
        managed: false,
        pid: None,
        active_profile_id: current.active_profile_id,
        message: "llama-server stopped".into(),
        model_path: current.model_path,
    })
}

/// Kill the process (if any) that is listening on `port` using PowerShell's
/// Get-NetTCPConnection.  Best-effort — errors are silently swallowed.
async fn kill_by_port(port: u16) {
    let ps_cmd = format!(
        "$p = (Get-NetTCPConnection -LocalPort {port} -State Listen \
         -ErrorAction SilentlyContinue | Select-Object -First 1 \
         -ExpandProperty OwningProcess); \
         if ($p) {{ Stop-Process -Id $p -Force -ErrorAction SilentlyContinue }}"
    );
    let _ = tokio::process::Command::new("powershell.exe")
        .args(["-NoProfile", "-NonInteractive", "-Command", &ps_cmd])
        .output()
        .await;
}

pub async fn restart(state: &SharedState) -> Result<StatusResponse> {
    let _ = stop(state).await?;
    start(state).await
}

pub async fn switch_profile(state: &SharedState, profile_id: &str) -> Result<StatusResponse> {
    let profile = {
        let profiles = state.profiles.lock().await;
        profiles
            .get(profile_id)
            .cloned()
            .ok_or_else(|| anyhow!("profile not found: {profile_id}"))?
    };
    if !profile.enabled {
        return Err(anyhow!("profile is disabled: {profile_id}"));
    }

    *state.active_profile_id.lock().await = profile_id.to_string();
    let current = status(state).await;
    Ok(StatusResponse {
        running: current.running,
        managed: current.managed,
        pid: current.pid,
        active_profile_id: profile_id.to_string(),
        message: "active profile switched".into(),
        model_path: profile.model_path,
    })
}

pub async fn update_profile(state: &SharedState, profile: ModelProfile) -> Result<ModelProfile> {
    let mut profiles_file = config::load_profiles_file(&state.config.model_profiles_path)?;
    let position = profiles_file
        .iter()
        .position(|item| item.id == profile.id)
        .ok_or_else(|| anyhow!("profile not found: {}", profile.id))?;

    profiles_file[position] = profile.clone();
    config::save_profiles_file(&state.config.model_profiles_path, &profiles_file)?;

    let mut profiles_map = state.profiles.lock().await;
    profiles_map.insert(profile.id.clone(), profile.clone());

    Ok(profile)
}

fn resolve_binary(state: &SharedState) -> Result<PathBuf> {
    if !state.llama.binary_path.trim().is_empty() {
        let path = PathBuf::from(&state.llama.binary_path);
        if path.exists() {
            return Ok(path);
        }
    }

    for candidate in &state.llama.binary_candidates {
        let path = PathBuf::from(candidate);
        if path.exists() {
            return Ok(path);
        }
    }

    Err(anyhow!("unable to find llama-server binary"))
}

fn build_args(config: &crate::models::LlamaServerConfig, profile: &ModelProfile) -> Vec<String> {
    let mut args = vec![
        "--model".into(),
        profile.model_path.clone(),
        "--host".into(),
        config.listen_host.clone(),
        "--port".into(),
        config.listen_port.to_string(),
        "--ctx-size".into(),
        profile.ctx_size.to_string(),
        "--n-gpu-layers".into(),
        profile.n_gpu_layers.to_string(),
        "--threads".into(),
        profile.threads.to_string(),
        "--parallel".into(),
        profile.parallel.to_string(),
    ];

    if profile.flash_attn {
        args.push("--flash-attn".into());
        args.push("on".into());
    }
    args.push("--reasoning-format".into());
    args.push("deepseek".into());
    if config.cont_batching {
        args.push("--cont-batching".into());
    }
    args
}
