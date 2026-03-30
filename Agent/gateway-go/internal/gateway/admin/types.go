package admin

type ServiceStatus struct {
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	Address     string       `json:"address,omitempty"`
	LastCheckAt string       `json:"last_check_at"`
	Message     string       `json:"message,omitempty"`
	Control     ControlState `json:"control"`
}

type ControlState struct {
	CanStart          bool   `json:"can_start"`
	CanStop           bool   `json:"can_stop"`
	UnsupportedReason string `json:"unsupported_reason,omitempty"`
}

type ModelProfile struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	ModelPath  string `json:"model_path"`
	Quant      string `json:"quant"`
	CtxSize    int    `json:"ctx_size"`
	Parallel   int    `json:"parallel"`
	Threads    int    `json:"threads"`
	NGPULayers int    `json:"n_gpu_layers"`
	FlashAttn  bool   `json:"flash_attn"`
	Enabled    bool   `json:"enabled"`
}

type ActiveModel struct {
	ProfileID string `json:"profile_id,omitempty"`
	Label     string `json:"label,omitempty"`
	ModelPath string `json:"model_path,omitempty"`
	Quant     string `json:"quant,omitempty"`
	Running   bool   `json:"running"`
	PID       int    `json:"pid,omitempty"`
	Managed   bool   `json:"managed"`
	Message   string `json:"message,omitempty"`
}

type OverviewResponse struct {
	Services               []ServiceStatus  `json:"services"`
	ActiveModel            ActiveModel      `json:"active_model"`
	AvailableProfiles      []ModelProfile   `json:"available_profiles"`
	RecentSessionSummaries []map[string]any `json:"recent_session_summaries"`
	RecentToolSummaries    []map[string]any `json:"recent_tool_summaries"`
	RecentFailures         []map[string]any `json:"recent_failures"`
}

type ModelsResponse struct {
	ActiveProfileID string         `json:"active_profile_id,omitempty"`
	ActiveModel     ActiveModel    `json:"active_model"`
	Profiles        []ModelProfile `json:"profiles"`
}

type ModeDefinition struct {
	Name                    string   `json:"name"`
	Type                    string   `json:"type"`
	PromptFiles             []string `json:"prompt_files"`
	ConversationPromptFiles []string `json:"conversation_prompt_files,omitempty"`
	ToolScope               []string `json:"tool_scope"`
	RetrievalRoots          []string `json:"retrieval_roots"`
	EvalTags                []string `json:"eval_tags"`
	PluginCapabilities      []string `json:"plugin_capabilities,omitempty"`
}

type ModesResponse struct {
	Core    ModeDefinition   `json:"core"`
	Plugins []ModeDefinition `json:"plugins"`
}

type UserSummary struct {
	UserEmail   string `json:"user_email"`
	Label       string `json:"label"`
	LastSeenAt  string `json:"last_seen_at,omitempty"`
	HasSettings bool   `json:"has_settings"`
	HasLegacy   bool   `json:"has_legacy"`
}

type UsersResponse struct {
	Users      []UserSummary `json:"users"`
	ConfigPath string        `json:"config_path"`
}

type UserWorkspace struct {
	UserEmail                string              `json:"user_email"`
	Label                    string              `json:"label"`
	TerminalAllowedPaths     []string            `json:"terminal_allowed_paths"`
	DefaultLocalWorkdir      string              `json:"default_local_workdir,omitempty"`
	DefaultSSHHostID         string              `json:"default_ssh_host_id,omitempty"`
	EnabledMCPServerIDs      []string            `json:"enabled_mcp_server_ids,omitempty"`
	DisabledMCPToolsByServer map[string][]string `json:"disabled_mcp_tools_by_server,omitempty"`
}

type UserWorkspaceResponse struct {
	Workspace          UserWorkspace `json:"workspace"`
	ConfigPath         string        `json:"config_path"`
	GlobalAllowedPaths []string      `json:"global_allowed_paths"`
	RestartRequired    bool          `json:"restart_required"`
	LegacyBindingsPath string        `json:"legacy_bindings_path,omitempty"`
}

type UpdateUserWorkspaceRequest struct {
	Workspace UserWorkspace `json:"workspace"`
}

type SwitchModelRequest struct {
	ProfileID string `json:"profile_id"`
}

type HostIPsResponse struct {
	IPs       []string `json:"ips"`
	SharePort int      `json:"share_port"`
	ShareURLs []string `json:"share_urls"`
}

type ServiceActionRequest struct {
	Name string `json:"name"`
}

type ServiceActionResult struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type StartAllResponse struct {
	OK      bool                  `json:"ok"`
	Results []ServiceActionResult `json:"results"`
}

type SelfCheckItem struct {
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type SelfCheckResponse struct {
	OK     bool            `json:"ok"`
	Checks []SelfCheckItem `json:"checks"`
}

type UpdateModelProfileRequest struct {
	Profile  ModelProfile `json:"profile"`
	ApplyNow bool         `json:"apply_now"`
}

type TerminalPathsSettingsResponse struct {
	AllowedPaths    []string `json:"allowed_paths"`
	ConfigPath      string   `json:"config_path"`
	RestartRequired bool     `json:"restart_required"`
}

type UpdateTerminalPathsRequest struct {
	AllowedPaths []string `json:"allowed_paths"`
	RestartNow   bool     `json:"restart_now"`
}

type SSHHostProfile struct {
	ID                 string   `json:"id"`
	Label              string   `json:"label"`
	Enabled            bool     `json:"enabled"`
	Host               string   `json:"host"`
	Port               int      `json:"port"`
	Username           string   `json:"username"`
	AuthType           string   `json:"auth_type"`
	Password           string   `json:"password,omitempty"`
	PrivateKey         string   `json:"private_key,omitempty"`
	Passphrase         string   `json:"passphrase,omitempty"`
	RemoteShellDefault string   `json:"remote_shell_default"`
	AllowedPaths       []string `json:"allowed_paths"`
	DefaultWorkdir     string   `json:"default_workdir,omitempty"`
	HostKeyStatus      string   `json:"host_key_status"`
	HostKeyFingerprint string   `json:"host_key_fingerprint,omitempty"`
	HasPassword        bool     `json:"has_password,omitempty"`
	HasPrivateKey      bool     `json:"has_private_key,omitempty"`
}

type SSHHostsResponse struct {
	Hosts      []SSHHostProfile `json:"hosts"`
	ConfigPath string           `json:"config_path"`
	ToolRouter string           `json:"tool_router"`
}

type UpdateSSHHostsRequest struct {
	Hosts []SSHHostProfile `json:"hosts"`
}

type SSHHostTestRequest struct {
	Host      SSHHostProfile `json:"host"`
	TimeoutMS int            `json:"timeout_ms,omitempty"`
}

type SSHHostTestResponse struct {
	OK                 bool         `json:"ok"`
	Summary            string       `json:"summary"`
	HostKeyStatus      string       `json:"host_key_status"`
	HostKeyFingerprint string       `json:"host_key_fingerprint"`
	Error              *ErrorDetail `json:"error,omitempty"`
}

type ConfirmSSHHostKeyRequest struct {
	HostID      string `json:"host_id"`
	Fingerprint string `json:"fingerprint"`
}

type SSHUserBinding struct {
	UserEmail     string `json:"user_email"`
	DefaultHostID string `json:"default_host_id"`
}

type SSHBindingsResponse struct {
	Bindings   []SSHUserBinding `json:"bindings"`
	ConfigPath string           `json:"config_path"`
}

type UpdateSSHBindingsRequest struct {
	Bindings []SSHUserBinding `json:"bindings"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MCPServerProfile struct {
	ID            string            `json:"id"`
	Label         string            `json:"label"`
	Enabled       bool              `json:"enabled"`
	Kind          string            `json:"kind"`
	Description   string            `json:"description,omitempty"`
	PluginScope   []string          `json:"plugin_scope"`
	AuthType      string            `json:"auth_type"`
	AuthPayload   map[string]string `json:"auth_payload,omitempty"`
	DisabledTools []string          `json:"disabled_tools"`
	TimeoutMS     int               `json:"timeout_ms,omitempty"`
	VerifyTLS     bool              `json:"verify_tls"`
	Notes         string            `json:"notes,omitempty"`
	URL           string            `json:"url,omitempty"`
	Command       []string          `json:"command,omitempty"`
	Workdir       string            `json:"workdir,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
}

type MCPDiscoveredTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled"`
}

type MCPRuntimeEntry struct {
	ServerID               string `json:"server_id"`
	Label                  string `json:"label"`
	Enabled                bool   `json:"enabled"`
	Kind                   string `json:"kind"`
	Status                 string `json:"status"`
	BridgePort             int    `json:"bridge_port,omitempty"`
	ProcessPID             int    `json:"process_pid,omitempty"`
	EffectiveOpenWebUIType string `json:"effective_openwebui_type,omitempty"`
	EffectiveConnectionURL string `json:"effective_connection_url,omitempty"`
	LastError              string `json:"last_error,omitempty"`
}

type MCPServerState struct {
	Profile                MCPServerProfile    `json:"profile"`
	DiscoveredTools        []MCPDiscoveredTool `json:"discovered_tools"`
	LastDiscoveredAt       string              `json:"last_discovered_at,omitempty"`
	RuntimeStatus          MCPRuntimeEntry     `json:"runtime_status"`
	EffectiveOpenWebUIType string              `json:"effective_openwebui_type,omitempty"`
	EffectiveConnectionURL string              `json:"effective_connection_url,omitempty"`
	LastError              string              `json:"last_error,omitempty"`
}

type MCPServersResponse struct {
	Servers        []MCPServerState `json:"servers"`
	ConfigPath     string           `json:"config_path"`
	ToolCachePath  string           `json:"tool_cache_path"`
	ToolRouterBase string           `json:"tool_router_base"`
}

type UpdateMCPServersRequest struct {
	Servers []MCPServerProfile `json:"servers"`
}

type MCPValidateRequest struct {
	Server MCPServerProfile `json:"server"`
}

type MCPValidateResponse struct {
	OK                     bool         `json:"ok"`
	Summary                string       `json:"summary"`
	EffectiveOpenWebUIType string       `json:"effective_openwebui_type,omitempty"`
	EffectiveConnectionURL string       `json:"effective_connection_url,omitempty"`
	Error                  *ErrorDetail `json:"error,omitempty"`
}

type MCPDiscoverToolsRequest struct {
	ServerID string `json:"server_id"`
}

type MCPDiscoverToolsResponse struct {
	OK                     bool                `json:"ok"`
	Summary                string              `json:"summary"`
	ServerID               string              `json:"server_id"`
	Tools                  []MCPDiscoveredTool `json:"tools"`
	LastDiscoveredAt       string              `json:"last_discovered_at,omitempty"`
	EffectiveOpenWebUIType string              `json:"effective_openwebui_type,omitempty"`
	EffectiveConnectionURL string              `json:"effective_connection_url,omitempty"`
	Error                  *ErrorDetail        `json:"error,omitempty"`
}

type MCPRuntimeStatusResponse struct {
	Servers []MCPRuntimeEntry `json:"servers"`
}

type OpenWebUIToolConnection struct {
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	Enabled                bool              `json:"enabled"`
	Type                   string            `json:"type"`
	URL                    string            `json:"url"`
	Path                   string            `json:"path,omitempty"`
	SpecType               string            `json:"spec_type,omitempty"`
	AuthType               string            `json:"auth_type"`
	Key                    string            `json:"key,omitempty"`
	Headers                map[string]string `json:"headers,omitempty"`
	FunctionNameFilterList string            `json:"function_name_filter_list,omitempty"`
	Config                 map[string]any    `json:"config,omitempty"`
	Info                   map[string]any    `json:"info,omitempty"`
}

type MCPOpenWebUIPreviewResponse struct {
	Connections               []OpenWebUIToolConnection `json:"connections"`
	ToolServerConnectionsJSON string                    `json:"tool_server_connections_json"`
	RestartRequired           bool                      `json:"restart_required"`
}
