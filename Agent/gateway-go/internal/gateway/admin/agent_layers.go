package admin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/config"
)

type agentLayerPresetFile struct {
	DefaultPresetID string             `json:"default_preset_id,omitempty"`
	Presets         []AgentLayerPreset `json:"presets"`
}

func (s *Service) AgentLayers() (*AgentLayersResponse, error) {
	payload, err := s.readAgentLayerPresetFile()
	if err != nil {
		return nil, err
	}
	withEffective, err := s.attachEffectiveAgentLayers(payload.Presets)
	if err != nil {
		return nil, err
	}
	return &AgentLayersResponse{
		ConfigPath:       s.agentLayersPath,
		DefaultPresetID:  payload.DefaultPresetID,
		Presets:          withEffective,
		RouterPromptFile: routerPromptPath(s.cfg),
		RestartRequired:  false,
	}, nil
}

func (s *Service) DefaultAgentLayerPreset() (*AgentLayerPreset, error) {
	payload, err := s.readAgentLayerPresetFile()
	if err != nil {
		return nil, err
	}
	if len(payload.Presets) == 0 {
		return nil, nil
	}
	defaultID := strings.TrimSpace(payload.DefaultPresetID)
	if defaultID != "" {
		for _, item := range payload.Presets {
			if strings.EqualFold(item.ID, defaultID) {
				preset := item
				return &preset, nil
			}
		}
	}
	preset := payload.Presets[0]
	return &preset, nil
}

func (s *Service) UpdateAgentLayers(req UpdateAgentLayersRequest) (*AgentLayersResponse, error) {
	cleaned, err := sanitizeAgentLayerPresets(req.Presets)
	if err != nil {
		return nil, err
	}
	defaultPresetID := strings.TrimSpace(req.DefaultPresetID)
	if defaultPresetID != "" && !containsPreset(cleaned, defaultPresetID) {
		return nil, fmt.Errorf("unknown default_preset_id: %s", defaultPresetID)
	}
	payload := agentLayerPresetFile{
		DefaultPresetID: defaultPresetID,
		Presets:         cleaned,
	}
	if err := writeJSONFile(s.agentLayersPath, payload); err != nil {
		return nil, err
	}
	return s.AgentLayers()
}

func (s *Service) readAgentLayerPresetFile() (*agentLayerPresetFile, error) {
	if strings.TrimSpace(s.agentLayersPath) == "" {
		return nil, fmt.Errorf("agent layer presets path is not configured")
	}
	data, err := os.ReadFile(s.agentLayersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultAgentLayerPresetFile(), nil
		}
		return nil, fmt.Errorf("read agent layer presets: %w", err)
	}
	var payload agentLayerPresetFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse agent layer presets: %w", err)
	}
	if len(payload.Presets) == 0 {
		return defaultAgentLayerPresetFile(), nil
	}
	cleaned, err := sanitizeAgentLayerPresets(payload.Presets)
	if err != nil {
		return nil, err
	}
	payload.Presets = cleaned
	if payload.DefaultPresetID != "" && !containsPreset(cleaned, payload.DefaultPresetID) {
		payload.DefaultPresetID = ""
	}
	return &payload, nil
}

func defaultAgentLayerPresetFile() *agentLayerPresetFile {
	return &agentLayerPresetFile{
		DefaultPresetID: "router-reverse",
		Presets: []AgentLayerPreset{
			{ID: "router-only", Label: "Router Only", EnableAgentRouter: true},
			{ID: "router-reverse", Label: "Router + Reverse", EnableAgentRouter: true, EnableReverseSkills: true},
			{ID: "router-pwn", Label: "Router + Pwn", EnableAgentRouter: true, EnablePwnSkills: true},
			{ID: "router-web", Label: "Router + Web", EnableAgentRouter: true, EnableWebSkills: true},
			{ID: "router-awdp-red", Label: "Router + AWDP Red", EnableAgentRouter: true, EnableAWDPRed: true},
			{ID: "router-awdp-blue", Label: "Router + AWDP Blue", EnableAgentRouter: true, EnableAWDPBlue: true},
			{ID: "router-web-awdp-red", Label: "Router + Web + AWDP Red", EnableAgentRouter: true, EnableWebSkills: true, EnableAWDPRed: true},
			{ID: "router-web-awdp-blue", Label: "Router + Web + AWDP Blue", EnableAgentRouter: true, EnableWebSkills: true, EnableAWDPBlue: true},
			{ID: "router-pwn-awdp-red", Label: "Router + Pwn + AWDP Red", EnableAgentRouter: true, EnablePwnSkills: true, EnableAWDPRed: true},
			{ID: "router-reverse-awdp-blue", Label: "Router + Reverse + AWDP Blue", EnableAgentRouter: true, EnableReverseSkills: true, EnableAWDPBlue: true},
			{ID: "router-reverse-pwn-web", Label: "Router + Reverse + Pwn + Web", EnableAgentRouter: true, EnableReverseSkills: true, EnablePwnSkills: true, EnableWebSkills: true},
		},
	}
}

func sanitizeAgentLayerPresets(items []AgentLayerPreset) ([]AgentLayerPreset, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("at least one agent layer preset is required")
	}
	seen := make(map[string]struct{}, len(items))
	cleaned := make([]AgentLayerPreset, 0, len(items))
	for _, raw := range items {
		item := AgentLayerPreset{
			ID:                  strings.TrimSpace(raw.ID),
			Label:               strings.TrimSpace(raw.Label),
			EnableAgentRouter:   raw.EnableAgentRouter,
			EnableReverseSkills: raw.EnableReverseSkills,
			EnablePwnSkills:     raw.EnablePwnSkills,
			EnableWebSkills:     raw.EnableWebSkills,
			EnableAWDPRed:       raw.EnableAWDPRed,
			EnableAWDPBlue:      raw.EnableAWDPBlue,
		}
		if item.ID == "" {
			return nil, fmt.Errorf("preset id is required")
		}
		if item.Label == "" {
			item.Label = item.ID
		}
		key := strings.ToLower(item.ID)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate preset id: %s", item.ID)
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, item)
	}
	slices.SortFunc(cleaned, func(a, b AgentLayerPreset) int {
		return strings.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
	})
	return cleaned, nil
}

func containsPreset(items []AgentLayerPreset, target string) bool {
	target = strings.TrimSpace(target)
	for _, item := range items {
		if strings.EqualFold(item.ID, target) {
			return true
		}
	}
	return false
}

func (s *Service) attachEffectiveAgentLayers(items []AgentLayerPreset) ([]AgentLayerPreset, error) {
	modes, err := loadModes(s.cfg)
	if err != nil {
		return nil, err
	}
	out := make([]AgentLayerPreset, 0, len(items))
	for _, item := range items {
		withEffective := item
		withEffective.EffectivePromptFiles = effectivePromptFiles(modes, item, s.cfg)
		withEffective.EffectiveSkillFiles = effectiveSkillFiles(modes, item)
		withEffective.EffectiveToolScope = effectiveToolScope(modes, item)
		withEffective.EffectiveRetrieval = effectiveRetrievalRoots(modes, item)
		withEffective.EffectivePlugins = effectivePlugins(item)
		out = append(out, withEffective)
	}
	return out, nil
}

func effectivePlugins(preset AgentLayerPreset) []string {
	var out []string
	if preset.EnableReverseSkills {
		out = append(out, "reverse")
	}
	if preset.EnablePwnSkills {
		out = append(out, "pwn")
	}
	if preset.EnableWebSkills {
		out = append(out, "web")
	}
	if preset.EnableAWDPRed {
		out = append(out, "awdp-red")
	}
	if preset.EnableAWDPBlue {
		out = append(out, "awdp-blue")
	}
	return out
}

func effectivePromptFiles(modes *ModesResponse, preset AgentLayerPreset, cfg *config.Config) []string {
	files := append([]string{}, modes.Core.PromptFiles...)
	if preset.EnableAgentRouter {
		files = append(files, routerPromptPath(cfg))
	}
	if preset.EnableReverseSkills || preset.EnablePwnSkills {
		files = append(files, binaryCorePromptPath(cfg))
	}
	if preset.EnableAWDPRed || preset.EnableAWDPBlue {
		files = append(files, awdpCorePromptPath(cfg))
	}
	for _, plugin := range modes.Plugins {
		if plugin.Name == "reverse" && !preset.EnableReverseSkills {
			continue
		}
		if plugin.Name == "pwn" && !preset.EnablePwnSkills {
			continue
		}
		if plugin.Name == "web" && !preset.EnableWebSkills {
			continue
		}
		if plugin.Name == "awdp-red" && !preset.EnableAWDPRed {
			continue
		}
		if plugin.Name == "awdp-blue" && !preset.EnableAWDPBlue {
			continue
		}
		for _, item := range plugin.PromptFiles {
			if samePath(item, routerPromptPath(cfg), cfg) || samePath(item, binaryCorePromptPath(cfg), cfg) || samePath(item, awdpCorePromptPath(cfg), cfg) {
				continue
			}
			files = append(files, item)
		}
	}
	return uniqueStringsPreserve(files)
}

func effectiveSkillFiles(modes *ModesResponse, preset AgentLayerPreset) []string {
	var files []string
	for _, plugin := range modes.Plugins {
		if plugin.Name == "reverse" && preset.EnableReverseSkills {
			files = append(files, plugin.SkillFiles...)
		}
		if plugin.Name == "pwn" && preset.EnablePwnSkills {
			files = append(files, plugin.SkillFiles...)
		}
		if plugin.Name == "web" && preset.EnableWebSkills {
			files = append(files, plugin.SkillFiles...)
		}
		if plugin.Name == "awdp-red" && preset.EnableAWDPRed {
			files = append(files, plugin.SkillFiles...)
		}
		if plugin.Name == "awdp-blue" && preset.EnableAWDPBlue {
			files = append(files, plugin.SkillFiles...)
		}
	}
	return uniqueStringsPreserve(files)
}

func effectiveToolScope(modes *ModesResponse, preset AgentLayerPreset) []string {
	items := append([]string{}, modes.Core.ToolScope...)
	for _, plugin := range modes.Plugins {
		if plugin.Name == "reverse" && preset.EnableReverseSkills {
			items = append(items, plugin.ToolScope...)
		}
		if plugin.Name == "pwn" && preset.EnablePwnSkills {
			items = append(items, plugin.ToolScope...)
		}
		if plugin.Name == "web" && preset.EnableWebSkills {
			items = append(items, plugin.ToolScope...)
		}
		if plugin.Name == "awdp-red" && preset.EnableAWDPRed {
			items = append(items, plugin.ToolScope...)
		}
		if plugin.Name == "awdp-blue" && preset.EnableAWDPBlue {
			items = append(items, plugin.ToolScope...)
		}
	}
	return uniqueStringsPreserve(items)
}

func effectiveRetrievalRoots(modes *ModesResponse, preset AgentLayerPreset) []string {
	items := append([]string{}, modes.Core.RetrievalRoots...)
	for _, plugin := range modes.Plugins {
		if plugin.Name == "reverse" && preset.EnableReverseSkills {
			items = append(items, plugin.RetrievalRoots...)
		}
		if plugin.Name == "pwn" && preset.EnablePwnSkills {
			items = append(items, plugin.RetrievalRoots...)
		}
		if plugin.Name == "web" && preset.EnableWebSkills {
			items = append(items, plugin.RetrievalRoots...)
		}
		if plugin.Name == "awdp-red" && preset.EnableAWDPRed {
			items = append(items, plugin.RetrievalRoots...)
		}
		if plugin.Name == "awdp-blue" && preset.EnableAWDPBlue {
			items = append(items, plugin.RetrievalRoots...)
		}
	}
	return uniqueStringsPreserve(items)
}

func routerPromptPath(cfg *config.Config) string {
	return filepath.Clean(filepath.Join(cfg.Modes.CoreRoot, cfg.Modes.DefaultMode, "prompts", "agent.md"))
}

func binaryCorePromptPath(cfg *config.Config) string {
	return filepath.Clean(filepath.Join(cfg.Modes.CoreRoot, cfg.Modes.DefaultMode, "prompts", "binary-core.md"))
}

func awdpCorePromptPath(cfg *config.Config) string {
	return filepath.Clean(filepath.Join(cfg.Modes.CoreRoot, cfg.Modes.DefaultMode, "prompts", "awdp-core.md"))
}

func samePath(candidate, absolute string, cfg *config.Config) bool {
	normalize := func(value string) string {
		value = strings.ReplaceAll(value, `\`, "/")
		return strings.ToLower(filepath.Clean(value))
	}
	target := normalize(absolute)
	normalizedCandidate := normalize(candidate)
	if normalizedCandidate == target {
		return true
	}
	if strings.HasSuffix(normalizedCandidate, "core/awdp/prompts/agent.md") {
		return true
	}
	if strings.HasSuffix(normalizedCandidate, "core/awdp/prompts/binary-core.md") {
		return true
	}
	if strings.HasSuffix(normalizedCandidate, "core/awdp/prompts/awdp-core.md") {
		return true
	}
	joinedFromPluginRoot := normalize(filepath.Join(cfg.Modes.PluginRoot, candidate))
	return joinedFromPluginRoot == target
}

func uniqueStringsPreserve(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}
