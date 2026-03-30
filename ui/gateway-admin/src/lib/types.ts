export type ServiceStatus = {
  name: string;
  status: string;
  address?: string;
  last_check_at: string;
  message?: string;
  control: {
    can_start: boolean;
    can_stop: boolean;
    unsupported_reason?: string;
  };
};

export type ModelProfile = {
  id: string;
  label: string;
  model_path: string;
  quant: string;
  ctx_size: number;
  parallel: number;
  threads: number;
  n_gpu_layers: number;
  flash_attn: boolean;
  enabled: boolean;
};

export type ActiveModel = {
  profile_id?: string;
  label?: string;
  model_path?: string;
  quant?: string;
  running: boolean;
  pid?: number;
  managed: boolean;
  message?: string;
};

export type OverviewResponse = {
  services: ServiceStatus[];
  active_model: ActiveModel;
  available_profiles: ModelProfile[];
  recent_session_summaries: Record<string, unknown>[];
  recent_tool_summaries: Record<string, unknown>[];
  recent_failures: Record<string, unknown>[];
};

export type ModelsResponse = {
  active_profile_id?: string;
  active_model: ActiveModel;
  profiles: ModelProfile[];
};

export type ModeDefinition = {
  name: string;
  type: string;
  prompt_files: string[];
  conversation_prompt_files?: string[];
  tool_scope: string[];
  retrieval_roots: string[];
  eval_tags: string[];
  plugin_capabilities?: string[];
};

export type ModesResponse = {
  core: ModeDefinition;
  plugins: ModeDefinition[];
};

export type LogListResponse = {
  items: Record<string, unknown>[];
};

export type UserSummary = {
  user_email: string;
  label: string;
  last_seen_at?: string;
  has_settings: boolean;
  has_legacy: boolean;
};

export type UsersResponse = {
  users: UserSummary[];
  config_path: string;
};

export type UserWorkspace = {
  user_email: string;
  label: string;
  terminal_allowed_paths: string[];
  default_local_workdir?: string;
  default_ssh_host_id?: string;
  enabled_mcp_server_ids?: string[];
  disabled_mcp_tools_by_server?: Record<string, string[]>;
};

export type UserWorkspaceResponse = {
  workspace: UserWorkspace;
  config_path: string;
  global_allowed_paths: string[];
  restart_required: boolean;
  legacy_bindings_path?: string;
};

export type ServicesResponse = {
  services: ServiceStatus[];
};

export type HostIPsResponse = {
  ips: string[];
  share_port: number;
  share_urls: string[];
};

export type TerminalPathsSettings = {
  allowed_paths: string[];
  config_path: string;
  restart_required: boolean;
};

export type SSHHostProfile = {
  id: string;
  label: string;
  enabled: boolean;
  host: string;
  port: number;
  username: string;
  auth_type: "password" | "private_key";
  password?: string;
  private_key?: string;
  passphrase?: string;
  remote_shell_default: "bash" | "powershell";
  allowed_paths: string[];
  default_workdir?: string;
  host_key_status: "unknown" | "trusted";
  host_key_fingerprint?: string;
  has_password?: boolean;
  has_private_key?: boolean;
};

export type SSHHostsResponse = {
  hosts: SSHHostProfile[];
  config_path: string;
  tool_router: string;
};

export type SSHHostTestResponse = {
  ok: boolean;
  summary: string;
  host_key_status: "unknown" | "trusted";
  host_key_fingerprint: string;
  error?: {
    code: string;
    message: string;
  };
};

export type SSHUserBinding = {
  user_email: string;
  default_host_id: string;
};

export type SSHBindingsResponse = {
  bindings: SSHUserBinding[];
  config_path: string;
};

export type MCPServerProfile = {
  id: string;
  label: string;
  enabled: boolean;
  kind: "native_streamable_http" | "mcpo_stdio" | "mcpo_sse";
  description?: string;
  plugin_scope: string[];
  auth_type: "none" | "bearer" | "basic" | "header";
  auth_payload: Record<string, string>;
  disabled_tools: string[];
  timeout_ms: number;
  verify_tls: boolean;
  notes?: string;
  url?: string;
  command?: string[];
  workdir?: string;
  env?: Record<string, string>;
  headers?: Record<string, string>;
};

export type MCPDiscoveredTool = {
  name: string;
  description?: string;
  disabled: boolean;
};

export type MCPRuntimeEntry = {
  server_id: string;
  label: string;
  enabled: boolean;
  kind: string;
  status: string;
  bridge_port?: number;
  process_pid?: number;
  effective_openwebui_type?: string;
  effective_connection_url?: string;
  last_error?: string;
};

export type MCPServerState = {
  profile: MCPServerProfile;
  discovered_tools: MCPDiscoveredTool[];
  last_discovered_at?: string;
  runtime_status: MCPRuntimeEntry;
  effective_openwebui_type?: string;
  effective_connection_url?: string;
  last_error?: string;
};

export type MCPServersResponse = {
  servers: MCPServerState[];
  config_path: string;
  tool_cache_path: string;
  tool_router_base: string;
};

export type MCPValidateResponse = {
  ok: boolean;
  summary: string;
  effective_openwebui_type?: string;
  effective_connection_url?: string;
  error?: {
    code: string;
    message: string;
  };
};

export type MCPDiscoverToolsResponse = {
  ok: boolean;
  summary: string;
  server_id: string;
  tools: MCPDiscoveredTool[];
  last_discovered_at?: string;
  effective_openwebui_type?: string;
  effective_connection_url?: string;
  error?: {
    code: string;
    message: string;
  };
};

export type MCPOpenWebUIConnection = {
  id: string;
  name: string;
  enabled: boolean;
  type: string;
  url: string;
  path?: string;
  spec_type?: string;
  auth_type: string;
  key?: string;
  headers?: Record<string, string>;
  function_name_filter_list?: string;
};

export type MCPOpenWebUIPreviewResponse = {
  connections: MCPOpenWebUIConnection[];
  tool_server_connections_json: string;
  restart_required: boolean;
};
