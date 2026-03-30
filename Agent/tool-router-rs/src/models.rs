use std::{
    collections::{BTreeMap, HashMap},
    sync::Arc,
};

use jsonschema::Validator;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use tokio::sync::Mutex;

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
    #[serde(default)]
    pub mcp: McpConfig,
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

#[derive(Debug, Deserialize, Clone, Default)]
pub struct McpConfig {
    #[serde(default)]
    pub servers_path: String,
    #[serde(default)]
    pub tool_cache_path: String,
    #[serde(default)]
    pub bridge_host: String,
    #[serde(default)]
    pub bridge_port_start: u16,
    #[serde(default)]
    pub bridge_port_end: u16,
    #[serde(default)]
    pub process_log_dir: String,
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
    pub workspace_root: String,
    pub ssh_hosts_path: String,
    pub mcp_servers_path: String,
    pub mcp_tool_cache_path: String,
    pub mcp_runtime: Arc<Mutex<McpRuntimeState>>,
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
    #[serde(default)]
    pub user_email: String,
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
    #[serde(default)]
    pub user_email: String,
    pub arguments: Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
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

#[derive(Debug, Deserialize)]
pub struct OpenAPITerminalRequest {
    pub command: String,
    pub transport: Option<String>,
    pub shell: Option<String>,
    pub host_id: Option<String>,
    pub remote_shell: Option<String>,
    pub workdir: Option<String>,
    pub timeout_ms: Option<u64>,
    pub user_email: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct OpenAPITerminalResponse {
    pub ok: bool,
    pub summary: String,
    pub exit_code: i32,
    pub stdout: String,
    pub stderr: String,
    pub timed_out: bool,
    pub duration_ms: u64,
    pub transport: String,
    pub host_id: String,
    pub remote_shell: String,
    pub shell: String,
    pub workdir: String,
    pub truncated: bool,
    pub error: Option<ErrorEnvelope>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SshHostProfile {
    pub id: String,
    pub label: String,
    #[serde(default = "default_true")]
    pub enabled: bool,
    pub host: String,
    #[serde(default = "default_ssh_port")]
    pub port: u16,
    pub username: String,
    pub auth_type: String,
    #[serde(default)]
    pub password: Option<String>,
    #[serde(default)]
    pub private_key: Option<String>,
    #[serde(default)]
    pub passphrase: Option<String>,
    #[serde(default = "default_remote_shell")]
    pub remote_shell_default: String,
    #[serde(default)]
    pub allowed_paths: Vec<String>,
    #[serde(default)]
    pub default_workdir: Option<String>,
    #[serde(default = "default_host_key_status")]
    pub host_key_status: String,
    #[serde(default)]
    pub host_key_fingerprint: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SshUserBinding {
    pub user_email: String,
    pub default_host_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SshTestRequest {
    pub host: SshHostProfile,
    #[serde(default)]
    pub timeout_ms: Option<u64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SshTestResponse {
    pub ok: bool,
    pub summary: String,
    pub host_key_status: String,
    pub host_key_fingerprint: String,
    #[serde(default)]
    pub error: Option<ErrorEnvelope>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct McpServerProfile {
    pub id: String,
    pub label: String,
    #[serde(default = "default_true")]
    pub enabled: bool,
    pub kind: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub plugin_scope: Vec<String>,
    #[serde(default = "default_mcp_auth_type")]
    pub auth_type: String,
    #[serde(default)]
    pub auth_payload: HashMap<String, String>,
    #[serde(default)]
    pub disabled_tools: Vec<String>,
    #[serde(default = "default_mcp_timeout")]
    pub timeout_ms: u64,
    #[serde(default = "default_verify_tls")]
    pub verify_tls: bool,
    #[serde(default)]
    pub notes: String,
    #[serde(default)]
    pub url: String,
    #[serde(default)]
    pub command: Vec<String>,
    #[serde(default)]
    pub workdir: String,
    #[serde(default)]
    pub env: HashMap<String, String>,
    #[serde(default)]
    pub headers: HashMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct McpDiscoveredTool {
    pub name: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub disabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct McpToolCacheFile {
    #[serde(default)]
    pub servers: Vec<McpToolCacheEntry>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct McpToolCacheEntry {
    pub server_id: String,
    #[serde(default)]
    pub tools: Vec<McpDiscoveredTool>,
    #[serde(default)]
    pub last_discovered_at: String,
    #[serde(default)]
    pub last_error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct McpValidateRequest {
    pub server: McpServerProfile,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct McpValidateResponse {
    pub ok: bool,
    pub summary: String,
    #[serde(default)]
    pub effective_openwebui_type: String,
    #[serde(default)]
    pub effective_connection_url: String,
    #[serde(default)]
    pub error: Option<ErrorEnvelope>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct McpDiscoverRequest {
    pub server_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct McpDiscoverResponse {
    pub ok: bool,
    pub summary: String,
    pub server_id: String,
    #[serde(default)]
    pub tools: Vec<McpDiscoveredTool>,
    #[serde(default)]
    pub last_discovered_at: String,
    #[serde(default)]
    pub effective_openwebui_type: String,
    #[serde(default)]
    pub effective_connection_url: String,
    #[serde(default)]
    pub error: Option<ErrorEnvelope>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct McpRuntimeStatusResponse {
    #[serde(default)]
    pub servers: Vec<McpRuntimeEntry>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct McpRuntimeEntry {
    pub server_id: String,
    pub label: String,
    pub enabled: bool,
    pub kind: String,
    pub status: String,
    #[serde(default)]
    pub bridge_port: u16,
    #[serde(default)]
    pub process_pid: u32,
    #[serde(default)]
    pub effective_openwebui_type: String,
    #[serde(default)]
    pub effective_connection_url: String,
    #[serde(default)]
    pub last_error: String,
}

#[derive(Debug, Default)]
pub struct McpRuntimeState {
    pub signature: String,
    pub bridges: Vec<McpBridgeProcess>,
}

#[derive(Debug)]
pub struct McpBridgeProcess {
    pub server_id: String,
    pub kind: String,
    pub port: u16,
    pub config_path: String,
    pub pid: u32,
    pub last_error: String,
    pub child: Option<tokio::process::Child>,
}

fn default_true() -> bool {
    true
}

fn default_ssh_port() -> u16 {
    22
}

fn default_remote_shell() -> String {
    "bash".into()
}

fn default_host_key_status() -> String {
    "unknown".into()
}

fn default_mcp_auth_type() -> String {
    "none".into()
}

fn default_mcp_timeout() -> u64 {
    30_000
}

fn default_verify_tls() -> bool {
    true
}
