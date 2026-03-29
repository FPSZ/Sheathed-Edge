package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/pathutil"
)

type Config struct {
	ListenHost string `json:"listen_host"`
	ListenPort int    `json:"listen_port"`

	ProviderModelAlias string `json:"provider_model_alias"`

	LlamaServer struct {
		BaseURL   string `json:"base_url"`
		TimeoutMS int    `json:"timeout_ms"`
	} `json:"llama_server"`

	ToolRouter struct {
		BaseURL   string `json:"base_url"`
		TimeoutMS int    `json:"timeout_ms"`
	} `json:"tool_router"`

	Modes struct {
		CoreRoot       string   `json:"core_root"`
		DefaultMode    string   `json:"default_mode"`
		PluginRoot     string   `json:"plugin_root"`
		AllowedPlugins []string `json:"allowed_plugins"`
	} `json:"modes"`

	KnowledgeRoots []string `json:"knowledge_roots"`

	Retrieval struct {
		SQLiteIndex      string `json:"sqlite_index"`
		FallbackCommand  string `json:"fallback_command"`
		MaxFragments     int    `json:"max_fragments"`
		MaxFragmentChars int    `json:"max_fragment_chars"`
	} `json:"retrieval"`

	ActionEnvelopeSchema string `json:"action_envelope_schema"`

	Logs struct {
		SessionLogDir string `json:"session_log_dir"`
		AuditLogDir   string `json:"audit_log_dir"`
	} `json:"logs"`

	Admin struct {
		HostAgentURL      string `json:"host_agent_url"`
		HostAgentBinary   string `json:"host_agent_binary"`
		HostAgentConfig   string `json:"host_agent_config"`
		ToolRouterConfig  string `json:"tool_router_config_path"`
		OpenWebUIURL      string `json:"open_webui_url"`
		WebUISharePort    int    `json:"webui_share_port"`
		ToolLogDir        string `json:"tool_log_dir"`
		ModelProfilesPath string `json:"model_profiles_path"`
		UIDistDir         string `json:"ui_dist_dir"`
		TimeoutMS         int    `json:"timeout_ms"`
	} `json:"admin"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.Modes.CoreRoot = pathutil.NormalizeRuntimePath(cfg.Modes.CoreRoot)
	cfg.Modes.PluginRoot = pathutil.NormalizeRuntimePath(cfg.Modes.PluginRoot)
	cfg.Retrieval.SQLiteIndex = pathutil.NormalizeRuntimePath(cfg.Retrieval.SQLiteIndex)
	cfg.ActionEnvelopeSchema = pathutil.NormalizeRuntimePath(cfg.ActionEnvelopeSchema)
	cfg.Logs.SessionLogDir = pathutil.NormalizeRuntimePath(cfg.Logs.SessionLogDir)
	cfg.Logs.AuditLogDir = pathutil.NormalizeRuntimePath(cfg.Logs.AuditLogDir)
	cfg.Admin.ToolLogDir = pathutil.NormalizeRuntimePath(cfg.Admin.ToolLogDir)
	cfg.Admin.ModelProfilesPath = pathutil.NormalizeRuntimePath(cfg.Admin.ModelProfilesPath)
	cfg.Admin.UIDistDir = pathutil.NormalizeRuntimePath(cfg.Admin.UIDistDir)
	cfg.Admin.ToolRouterConfig = pathutil.NormalizeRuntimePath(cfg.Admin.ToolRouterConfig)
	for i, root := range cfg.KnowledgeRoots {
		cfg.KnowledgeRoots[i] = pathutil.NormalizeRuntimePath(root)
	}

	if cfg.ProviderModelAlias == "" {
		cfg.ProviderModelAlias = "awdp-r1-70b"
	}
	if cfg.Retrieval.MaxFragments <= 0 {
		cfg.Retrieval.MaxFragments = 8
	}
	if cfg.Retrieval.MaxFragmentChars <= 0 {
		cfg.Retrieval.MaxFragmentChars = 2400
	}
	if cfg.Admin.HostAgentURL == "" {
		cfg.Admin.HostAgentURL = "http://127.0.0.1:8098"
	}
	if cfg.Admin.OpenWebUIURL == "" {
		cfg.Admin.OpenWebUIURL = "http://127.0.0.1:3000"
	}
	if cfg.Admin.ToolRouterConfig == "" {
		cfg.Admin.ToolRouterConfig = pathutil.NormalizeRuntimePath(ResolveSiblingPath(path, "tool-router.config.json"))
	}
	if cfg.Admin.WebUISharePort <= 0 {
		cfg.Admin.WebUISharePort = 3001
	}
	if cfg.Admin.TimeoutMS <= 0 {
		cfg.Admin.TimeoutMS = 5000
	}

	for _, dir := range []string{cfg.Logs.SessionLogDir, cfg.Logs.AuditLogDir, cfg.Admin.ToolLogDir} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create log dir %s: %w", dir, err)
		}
	}

	return &cfg, nil
}

func ResolveSiblingPath(basePath, sibling string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(basePath), sibling))
}
