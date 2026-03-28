package mode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/config"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/pathutil"
)

type Config struct {
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	AlwaysEnabled      bool     `json:"always_enabled"`
	ExtendsCoreMode    string   `json:"extends_core_mode"`
	PromptFiles        []string `json:"prompt_files"`
	SkillFiles         []string `json:"skill_files"`
	ToolScope          []string `json:"tool_scope"`
	RetrievalRoots     []string `json:"retrieval_roots"`
	EvalTags           []string `json:"eval_tags"`
	PluginCapabilities []string `json:"plugin_capabilities"`
}

type Active struct {
	Name               string
	Plugins            []string
	SystemPrompt       string
	ToolScope          []string
	RetrievalRoots     []string
	EvalTags           []string
	PluginCapabilities []string
}

type Loader struct {
	cfg *config.Config
}

func NewLoader(cfg *config.Config) *Loader {
	return &Loader{cfg: cfg}
}

func (l *Loader) Load(plugins []string) (*Active, error) {
	corePath := filepath.Join(l.cfg.Modes.CoreRoot, l.cfg.Modes.DefaultMode, "mode.json")
	core, err := loadConfig(corePath)
	if err != nil {
		return nil, err
	}

	active := &Active{
		Name:               core.Name,
		ToolScope:          append([]string{}, core.ToolScope...),
		RetrievalRoots:     append([]string{}, core.RetrievalRoots...),
		EvalTags:           append([]string{}, core.EvalTags...),
		PluginCapabilities: append([]string{}, core.PluginCapabilities...),
	}

	var promptParts []string
	coreDir := filepath.Dir(corePath)
	for _, rel := range core.PromptFiles {
		content, err := os.ReadFile(filepath.Join(coreDir, rel))
		if err != nil {
			return nil, fmt.Errorf("read core prompt %s: %w", rel, err)
		}
		promptParts = append(promptParts, strings.TrimSpace(string(content)))
	}

	for _, plugin := range plugins {
		if !slices.Contains(l.cfg.Modes.AllowedPlugins, plugin) {
			return nil, fmt.Errorf("plugin not allowed: %s", plugin)
		}
		pluginPath := filepath.Join(l.cfg.Modes.PluginRoot, plugin, "plugin.json")
		pcfg, err := loadConfig(pluginPath)
		if err != nil {
			return nil, err
		}
		active.Plugins = append(active.Plugins, plugin)
		active.ToolScope = uniqueStrings(active.ToolScope, pcfg.ToolScope)
		active.RetrievalRoots = uniqueStrings(active.RetrievalRoots, pcfg.RetrievalRoots)
		active.EvalTags = uniqueStrings(active.EvalTags, pcfg.EvalTags)
		active.PluginCapabilities = uniqueStrings(active.PluginCapabilities, pcfg.PluginCapabilities)

		pluginDir := filepath.Dir(pluginPath)
		for _, rel := range pcfg.PromptFiles {
			content, err := os.ReadFile(filepath.Join(pluginDir, rel))
			if err != nil {
				return nil, fmt.Errorf("read plugin prompt %s: %w", rel, err)
			}
			promptParts = append(promptParts, strings.TrimSpace(string(content)))
		}
	}

	active.SystemPrompt = strings.Join(promptParts, "\n\n")
	return active, nil
}

func BuildLabel(active *Active) string {
	if len(active.Plugins) == 0 {
		return active.Name
	}
	return active.Name + "+" + strings.Join(active.Plugins, "+")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(pathutil.NormalizeRuntimePath(path))
	if err != nil {
		return nil, fmt.Errorf("read mode config %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse mode config %s: %w", path, err)
	}
	for i, root := range cfg.RetrievalRoots {
		cfg.RetrievalRoots[i] = pathutil.NormalizeRuntimePath(root)
	}
	return &cfg, nil
}

func uniqueStrings(base []string, add []string) []string {
	seen := make(map[string]struct{}, len(base))
	for _, item := range base {
		seen[item] = struct{}{}
	}
	for _, item := range add {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		base = append(base, item)
	}
	return base
}
