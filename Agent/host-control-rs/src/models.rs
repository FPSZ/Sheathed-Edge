use std::{collections::HashMap, sync::Arc};

use serde::{Deserialize, Serialize};
use tokio::{process::Child, sync::Mutex};

#[derive(Debug, Deserialize, Clone)]
pub struct Config {
    pub listen_host: String,
    pub listen_port: u16,
    pub llama_server_config_path: String,
    pub model_profiles_path: String,
    pub default_profile_id: String,
    pub timeout_ms: u64,
}

#[derive(Debug, Deserialize, Clone)]
pub struct LlamaServerConfig {
    #[serde(default)]
    pub binary_path: String,
    #[serde(default)]
    pub binary_candidates: Vec<String>,
    pub listen_host: String,
    pub listen_port: u16,
    pub log_dir: String,
    #[serde(default)]
    pub cont_batching: bool,
}

#[derive(Debug, Deserialize, Clone, Serialize)]
pub struct ModelProfile {
    pub id: String,
    pub label: String,
    pub model_path: String,
    pub quant: String,
    pub ctx_size: u32,
    pub parallel: u32,
    pub threads: u32,
    pub n_gpu_layers: u32,
    pub flash_attn: bool,
    pub enabled: bool,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ModelProfilesFile {
    pub profiles: Vec<ModelProfile>,
}

#[derive(Debug, Serialize)]
pub struct StatusResponse {
    pub running: bool,
    pub managed: bool,
    pub pid: Option<u32>,
    pub active_profile_id: String,
    pub message: String,
    pub model_path: String,
}

#[derive(Debug, Deserialize)]
pub struct SwitchRequest {
    pub profile_id: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct UpdateProfileRequest {
    pub profile: ModelProfile,
}

pub struct ManagedProcess {
    pub child: Child,
}

pub struct RuntimeState {
    pub config: Config,
    pub llama: LlamaServerConfig,
    pub profiles: Mutex<HashMap<String, ModelProfile>>,
    pub active_profile_id: Mutex<String>,
    pub process: Mutex<Option<ManagedProcess>>,
    pub http_client: reqwest::Client,
}

pub type SharedState = Arc<RuntimeState>;
