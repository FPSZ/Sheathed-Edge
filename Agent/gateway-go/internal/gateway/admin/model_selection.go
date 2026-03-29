package admin

import (
	"context"
	"fmt"
	"strings"
)

type ResolvedModel struct {
	ModelID string
	Profile *ModelProfile
}

func (s *Service) ExposedModels() ([]ResolvedModel, error) {
	profiles, err := s.EnabledProfiles()
	if err != nil {
		return nil, err
	}

	models := make([]ResolvedModel, 0, len(profiles))
	seen := make(map[string]struct{}, len(profiles))
	for i := range profiles {
		modelID := exposedModelID(profiles[i])
		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}
		models = append(models, ResolvedModel{
			ModelID: modelID,
			Profile: &profiles[i],
		})
	}
	return models, nil
}

func (s *Service) EnabledProfiles() ([]ModelProfile, error) {
	profiles, err := loadProfiles(s.cfg.Admin.ModelProfilesPath)
	if err != nil {
		return nil, err
	}

	out := make([]ModelProfile, 0, len(profiles))
	for _, profile := range profiles {
		if !profile.Enabled {
			continue
		}
		out = append(out, profile)
	}
	return out, nil
}

func (s *Service) ResolveModel(ctx context.Context, requestedModel string) (*ResolvedModel, error) {
	profiles, err := s.EnabledProfiles()
	if err != nil {
		return nil, err
	}

	modelID := strings.TrimSpace(requestedModel)
	if modelID == "" {
		modelID = s.defaultModelID(ctx, profiles)
	}
	if modelID == "" {
		modelID = strings.TrimSpace(s.cfg.ProviderModelAlias)
	}
	if modelID == "" {
		return nil, fmt.Errorf("no model is configured")
	}

	for i := range profiles {
		if matchesRequestedModel(profiles[i], modelID) {
			return &ResolvedModel{
				ModelID: exposedModelID(profiles[i]),
				Profile: &profiles[i],
			}, nil
		}
	}

	if requestedModel == "" {
		return &ResolvedModel{ModelID: modelID}, nil
	}
	return nil, fmt.Errorf("unsupported model: %s", requestedModel)
}

func (s *Service) EnsureModelReady(ctx context.Context, requestedModel string) (*ResolvedModel, error) {
	resolved, err := s.ResolveModel(ctx, requestedModel)
	if err != nil {
		return nil, err
	}
	if resolved.Profile == nil {
		return resolved, nil
	}

	status, err := s.host.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("host agent status unavailable: %w", err)
	}

	targetID := resolved.Profile.ID
	switch {
	case status.ActiveProfileID == targetID && status.Running:
		return resolved, nil
	case status.ActiveProfileID == targetID && !status.Running:
		if err := s.host.Start(ctx); err != nil {
			return nil, err
		}
	case status.ActiveProfileID != targetID && status.Running:
		if err := s.host.Stop(ctx); err != nil {
			return nil, err
		}
		if err := s.host.Switch(ctx, targetID); err != nil {
			return nil, err
		}
		if err := s.host.Start(ctx); err != nil {
			return nil, err
		}
	default:
		if err := s.host.Switch(ctx, targetID); err != nil {
			return nil, err
		}
	}

	return resolved, nil
}

func (s *Service) defaultModelID(ctx context.Context, profiles []ModelProfile) string {
	status, err := s.host.Status(ctx)
	if err == nil && strings.TrimSpace(status.ActiveProfileID) != "" {
		for _, profile := range profiles {
			if profile.ID == strings.TrimSpace(status.ActiveProfileID) {
				return exposedModelID(profile)
			}
		}
		return strings.TrimSpace(status.ActiveProfileID)
	}

	alias := strings.TrimSpace(s.cfg.ProviderModelAlias)
	for _, profile := range profiles {
		if profile.ID == alias || matchesRequestedModel(profile, alias) {
			return exposedModelID(profile)
		}
	}
	if len(profiles) > 0 {
		return exposedModelID(profiles[0])
	}
	return alias
}

func matchesRequestedModel(profile ModelProfile, requested string) bool {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return false
	}

	for _, alias := range aliasesForProfile(profile) {
		if alias == requested {
			return true
		}
	}
	return false
}

func exposedModelID(profile ModelProfile) string {
	aliases := aliasesForProfile(profile)
	if len(aliases) == 0 {
		return profile.ID
	}
	return aliases[0]
}

func aliasesForProfile(profile ModelProfile) []string {
	var aliases []string
	for _, suffix := range []string{"-experimental", "-competition"} {
		if strings.HasSuffix(profile.ID, suffix) {
			aliases = append(aliases, strings.TrimSuffix(profile.ID, suffix))
			break
		}
	}

	aliases = append(aliases, strings.TrimSpace(profile.ID))
	if strings.HasPrefix(profile.ID, "deepseek-r1-70b") {
		aliases = append(aliases, "awdp-r1-70b")
	}

	ordered := make([]string, 0, len(aliases))
	seen := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		ordered = append(ordered, alias)
	}
	return ordered
}
