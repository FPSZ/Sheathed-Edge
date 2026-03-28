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
