package admin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/config"
)

func loadModes(cfg *config.Config) (*ModesResponse, error) {
	corePath := filepath.Join(cfg.Modes.CoreRoot, cfg.Modes.DefaultMode, "mode.json")
	core, err := loadModeDefinition(corePath)
	if err != nil {
		return nil, err
	}

	var plugins []ModeDefinition
	for _, name := range cfg.Modes.AllowedPlugins {
		pluginPath := filepath.Join(cfg.Modes.PluginRoot, name, "plugin.json")
		def, err := loadModeDefinition(pluginPath)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, *def)
	}

	return &ModesResponse{
		Core:    *core,
		Plugins: plugins,
	}, nil
}

func loadModeDefinition(path string) (*ModeDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mode definition %s: %w", path, err)
	}

	var def ModeDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse mode definition %s: %w", path, err)
	}
	return &def, nil
}
