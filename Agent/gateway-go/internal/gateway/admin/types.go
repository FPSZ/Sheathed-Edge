package admin

type ServiceStatus struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Address     string `json:"address,omitempty"`
	LastCheckAt string `json:"last_check_at"`
	Message     string `json:"message,omitempty"`
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

type SwitchModelRequest struct {
	ProfileID string `json:"profile_id"`
}

type HostIPsResponse struct {
	IPs       []string `json:"ips"`
	SharePort int      `json:"share_port"`
	ShareURLs []string `json:"share_urls"`
}
