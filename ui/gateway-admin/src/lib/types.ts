export type ServiceStatus = {
  name: string;
  status: string;
  address?: string;
  last_check_at: string;
  message?: string;
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

export type ServicesResponse = {
  services: ServiceStatus[];
};

export type HostIPsResponse = {
  ips: string[];
  share_port: number;
  share_urls: string[];
};
