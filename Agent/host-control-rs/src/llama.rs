use std::{
    fs::{self, OpenOptions},
    path::{Path, PathBuf},
    process::Stdio,
};

use anyhow::{Context, Result, anyhow};
use tokio::process::Command;

use crate::models::{ManagedProcess, ModelProfile, SharedState, StatusResponse};

pub async fn status(state: &SharedState) -> StatusResponse {
    let active_profile_id = state.active_profile_id.lock().await.clone();
    let active_profile = state.profiles.get(&active_profile_id);
    let endpoint = format!(
        "http://{}:{}/health",
        state.llama.listen_host, state.llama.listen_port
    );

    let running = match state.http_client.get(&endpoint).send().await {
        Ok(resp) => resp.status().is_success(),
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
        model_path: active_profile
            .map(|p| p.model_path.clone())
            .unwrap_or_default(),
    }
}

pub async fn start(state: &SharedState) -> Result<StatusResponse> {
    let current = status(state).await;
    if current.running {
        return Ok(current);
    }

    let profile_id = state.active_profile_id.lock().await.clone();
    let profile = state
        .profiles
        .get(&profile_id)
        .ok_or_else(|| anyhow!("active profile not found: {profile_id}"))?;
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

    let mut args = build_args(&state.llama, profile);

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
        model_path: profile.model_path.clone(),
    })
}

pub async fn stop(state: &SharedState) -> Result<StatusResponse> {
    if let Some(mut managed) = state.process.lock().await.take() {
        managed.child.kill().await.ok();
        managed.child.wait().await.ok();
    }

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

pub async fn restart(state: &SharedState) -> Result<StatusResponse> {
    let _ = stop(state).await?;
    start(state).await
}

pub async fn switch_profile(state: &SharedState, profile_id: &str) -> Result<StatusResponse> {
    let profile = state
        .profiles
        .get(profile_id)
        .ok_or_else(|| anyhow!("profile not found: {profile_id}"))?;
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
        model_path: profile.model_path.clone(),
    })
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
    if config.cont_batching {
        args.push("--cont-batching".into());
    }
    args
}
