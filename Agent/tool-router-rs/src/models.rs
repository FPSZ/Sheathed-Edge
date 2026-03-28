use std::{collections::{BTreeMap, HashMap}, sync::Arc};

use jsonschema::Validator;
use serde::{Deserialize, Serialize};
use serde_json::Value;

#[derive(Debug, Deserialize, Clone)]
pub struct Config {
    pub listen_host: String,
    pub listen_port: u16,
    pub registry_path: String,
    pub logs: LogsConfig,
    #[serde(default)]
    pub timeouts: TimeoutConfig,
    #[serde(default)]
    pub allowed_paths: Vec<String>,
    #[serde(default)]
    pub plugin_scope_defaults: HashMap<String, Vec<String>>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct LogsConfig {
    pub tool_log_dir: String,
}

#[derive(Debug, Deserialize, Clone, Default)]
pub struct TimeoutConfig {
    pub default_ms: u64,
    pub max_ms: u64,
}

#[derive(Debug, Deserialize, Clone)]
pub struct Registry {
    pub tools: Vec<ToolEntry>,
    #[serde(default)]
    pub resources: Vec<Value>,
    #[serde(default)]
    pub prompts: Vec<Value>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct RetryConfig {
    pub max_attempts: u32,
    pub backoff_ms: u64,
}

#[derive(Debug, Deserialize, Clone)]
pub struct ToolEntry {
    pub name: String,
    pub description: String,
    #[serde(rename = "type")]
    pub tool_type: String,
    pub transport: String,
    pub command: Vec<String>,
    #[serde(default)]
    pub env: HashMap<String, String>,
    pub workdir: String,
    pub timeout_ms: u64,
    pub retry: RetryConfig,
    pub allowed_paths: Vec<String>,
    pub allowed_hosts: Vec<String>,
    #[serde(default)]
    pub allowed_cidrs: Vec<String>,
    #[serde(default)]
    pub allowed_schemes: Vec<String>,
    #[serde(default)]
    pub allowed_ports: Vec<u16>,
    pub plugin_scope: Vec<String>,
    pub parameter_schema: Value,
    pub capabilities: Vec<String>,
    pub enabled: bool,
}

#[derive(Clone)]
pub struct AppState {
    pub config: Config,
    pub tools: Arc<HashMap<String, ToolDef>>,
    pub resources_count: usize,
    pub prompts_count: usize,
}

#[derive(Clone)]
pub struct ToolDef {
    pub entry: ToolEntry,
    pub validator: Arc<Validator>,
}

#[derive(Debug, Deserialize)]
pub struct ResolveRequest {
    pub session_id: String,
    pub mode: String,
    pub tool: String,
    pub arguments: Value,
}

#[derive(Debug, Serialize)]
pub struct ResolveResponse {
    pub allowed: bool,
    pub tool: String,
    pub reason: String,
    pub normalized_arguments: Value,
}

#[derive(Debug, Deserialize)]
pub struct ExecuteRequest {
    pub session_id: String,
    pub mode: String,
    pub tool: String,
    pub arguments: Value,
}

#[derive(Debug, Serialize)]
pub struct ErrorEnvelope {
    pub code: String,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct ExecuteResponse {
    pub ok: bool,
    pub tool: String,
    pub result: BTreeMap<String, Value>,
    pub summary: String,
    pub truncated: bool,
    pub error: Option<ErrorEnvelope>,
}
