package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Service) TerminalPathSettings() (*TerminalPathsSettingsResponse, error) {
	cfgPath := strings.TrimSpace(s.toolRouterConfigPath)
	if cfgPath == "" {
		return nil, fmt.Errorf("tool router config path is not configured")
	}

	paths, err := s.readAllowedPaths(cfgPath)
	if err != nil {
		return nil, err
	}

	return &TerminalPathsSettingsResponse{
		AllowedPaths:    paths,
		ConfigPath:      cfgPath,
		RestartRequired: true,
	}, nil
}

func (s *Service) UpdateTerminalPathSettings(ctx context.Context, paths []string, restartNow bool) (*TerminalPathsSettingsResponse, error) {
	cfgPath := strings.TrimSpace(s.toolRouterConfigPath)
	if cfgPath == "" {
		return nil, fmt.Errorf("tool router config path is not configured")
	}

	cleaned, err := sanitizeAllowedPaths(paths)
	if err != nil {
		return nil, err
	}
	if err := s.writeAllowedPaths(cfgPath, cleaned); err != nil {
		return nil, err
	}

	if restartNow {
		go s.restartToolRouterDetached()
	}

	return &TerminalPathsSettingsResponse{
		AllowedPaths:    cleaned,
		ConfigPath:      cfgPath,
		RestartRequired: !restartNow,
	}, nil
}

func (s *Service) restartToolRouterDetached() {
	restartCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_ = s.StopService(restartCtx, serviceToolRouter)
	time.Sleep(900 * time.Millisecond)
	_ = s.StartService(restartCtx, serviceToolRouter)
}

func (s *Service) readAllowedPaths(configPath string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read tool-router config: %w", err)
	}

	var payload struct {
		AllowedPaths []string `json:"allowed_paths"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse tool-router config: %w", err)
	}

	return sanitizeAllowedPaths(payload.AllowedPaths)
}

func (s *Service) writeAllowedPaths(configPath string, paths []string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read tool-router config: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parse tool-router config: %w", err)
	}
	payload["allowed_paths"] = paths

	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode tool-router config: %w", err)
	}
	formatted = append(formatted, '\n')

	if err := os.WriteFile(configPath, formatted, 0o644); err != nil {
		return fmt.Errorf("write tool-router config: %w", err)
	}
	return nil
}

func sanitizeAllowedPaths(paths []string) ([]string, error) {
	seen := make(map[string]struct{}, len(paths))
	cleaned := make([]string, 0, len(paths))

	for _, raw := range paths {
		path := filepath.Clean(strings.TrimSpace(raw))
		if path == "" || path == "." {
			continue
		}
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("path must be absolute: %s", raw)
		}
		key := strings.ToLower(path)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, path)
	}

	if len(cleaned) == 0 {
		return nil, fmt.Errorf("at least one allowed path is required")
	}

	return cleaned, nil
}
