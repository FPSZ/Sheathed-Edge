package admin

import (
	"encoding/json"
	"fmt"
	"os"
)

type profilesFile struct {
	Profiles []ModelProfile `json:"profiles"`
}

func loadProfiles(path string) ([]ModelProfile, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read model profiles: %w", err)
	}

	var parsed profilesFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse model profiles: %w", err)
	}
	return parsed.Profiles, nil
}

func saveProfiles(path string, profiles []ModelProfile) error {
	data, err := json.MarshalIndent(profilesFile{Profiles: profiles}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal model profiles: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write model profiles: %w", err)
	}
	return nil
}
