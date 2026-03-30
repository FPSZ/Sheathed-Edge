package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func (s *Service) SSHHosts() (*SSHHostsResponse, error) {
	hosts, err := s.readSSHHosts()
	if err != nil {
		return nil, err
	}
	return &SSHHostsResponse{
		Hosts:      sanitizeSSHHostsForResponse(hosts),
		ConfigPath: s.sshHostsPath,
		ToolRouter: strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/"),
	}, nil
}

func (s *Service) UpdateSSHHosts(ctx context.Context, hosts []SSHHostProfile) (*SSHHostsResponse, error) {
	_ = ctx
	merged, err := s.mergeSSHHosts(hosts)
	if err != nil {
		return nil, err
	}
	if err := s.writeSSHHosts(merged); err != nil {
		return nil, err
	}
	return s.SSHHosts()
}

func (s *Service) TestSSHHost(ctx context.Context, req SSHHostTestRequest) (*SSHHostTestResponse, error) {
	host, err := s.mergeSSHHostSecrets(req.Host)
	if err != nil {
		return nil, err
	}
	timeoutMS := req.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 10_000
	}

	payload := map[string]any{
		"host":       host,
		"timeout_ms": timeoutMS,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode ssh test request: %w", err)
	}

	testCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS+2_000)*time.Millisecond)
	defer cancel()

	url := strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/") + "/internal/ssh/test"
	httpReq, err := http.NewRequestWithContext(testCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build ssh test request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: time.Duration(timeoutMS+2_000) * time.Millisecond}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call tool-router ssh test: %w", err)
	}
	defer resp.Body.Close()

	var parsed SSHHostTestResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode ssh test response: %w", err)
	}
	return &parsed, nil
}

func (s *Service) ConfirmSSHHostKey(ctx context.Context, req ConfirmSSHHostKeyRequest) (*SSHHostsResponse, error) {
	hostID := strings.TrimSpace(req.HostID)
	fingerprint := strings.TrimSpace(req.Fingerprint)
	if hostID == "" {
		return nil, fmt.Errorf("host_id is required")
	}
	if fingerprint == "" {
		return nil, fmt.Errorf("fingerprint is required")
	}

	hosts, err := s.readSSHHosts()
	if err != nil {
		return nil, err
	}

	found := false
	for i := range hosts {
		if !strings.EqualFold(hosts[i].ID, hostID) {
			continue
		}
		hosts[i].HostKeyStatus = "trusted"
		hosts[i].HostKeyFingerprint = fingerprint
		found = true
		break
	}
	if !found {
		return nil, fmt.Errorf("unknown ssh host: %s", hostID)
	}
	if err := s.writeSSHHosts(hosts); err != nil {
		return nil, err
	}
	return s.SSHHosts()
}

func (s *Service) SSHBindings() (*SSHBindingsResponse, error) {
	settings, err := s.readUserSettings()
	if err != nil {
		return nil, err
	}
	legacyBindings, err := s.readSSHBindings()
	if err != nil {
		return nil, err
	}
	byEmail := make(map[string]SSHUserBinding)
	for _, item := range settings {
		if strings.TrimSpace(item.DefaultSSHHostID) == "" {
			continue
		}
		byEmail[normalizeEmail(item.UserEmail)] = SSHUserBinding{
			UserEmail:     normalizeEmail(item.UserEmail),
			DefaultHostID: strings.TrimSpace(item.DefaultSSHHostID),
		}
	}
	for _, binding := range legacyBindings {
		email := normalizeEmail(binding.UserEmail)
		if email == "" {
			continue
		}
		if _, exists := byEmail[email]; exists {
			continue
		}
		byEmail[email] = SSHUserBinding{
			UserEmail:     email,
			DefaultHostID: strings.TrimSpace(binding.DefaultHostID),
		}
	}
	bindings := make([]SSHUserBinding, 0, len(byEmail))
	for _, binding := range byEmail {
		bindings = append(bindings, binding)
	}
	slices.SortFunc(bindings, func(a, b SSHUserBinding) int {
		return strings.Compare(a.UserEmail, b.UserEmail)
	})
	return &SSHBindingsResponse{
		Bindings:   bindings,
		ConfigPath: s.sshBindingsPath,
	}, nil
}

func (s *Service) UpdateSSHBindings(ctx context.Context, bindings []SSHUserBinding) (*SSHBindingsResponse, error) {
	_ = ctx
	cleaned, err := sanitizeSSHBindings(bindings)
	if err != nil {
		return nil, err
	}

	hostIDs := make(map[string]struct{})
	hosts, err := s.readSSHHosts()
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		hostIDs[strings.ToLower(host.ID)] = struct{}{}
	}
	for _, binding := range cleaned {
		if _, ok := hostIDs[strings.ToLower(binding.DefaultHostID)]; !ok {
			return nil, fmt.Errorf("unknown default_host_id: %s", binding.DefaultHostID)
		}
	}

	settings, err := s.readUserSettings()
	if err != nil {
		return nil, err
	}
	settingsByEmail := make(map[string]UserWorkspace, len(settings))
	for _, item := range settings {
		settingsByEmail[normalizeEmail(item.UserEmail)] = item
	}
	for _, binding := range cleaned {
		email := normalizeEmail(binding.UserEmail)
		item, exists := settingsByEmail[email]
		if !exists {
			globalAllowedPaths, pathErr := s.globalAllowedPaths()
			if pathErr != nil {
				return nil, pathErr
			}
			item = UserWorkspace{
				UserEmail:            email,
				Label:                labelFromEmail(email),
				TerminalAllowedPaths: append([]string{}, globalAllowedPaths...),
			}
		}
		item.DefaultSSHHostID = binding.DefaultHostID
		settingsByEmail[email] = item
	}
	nextSettings := make([]UserWorkspace, 0, len(settingsByEmail))
	for _, item := range settingsByEmail {
		nextSettings = append(nextSettings, item)
	}
	if err := s.writeUserSettings(nextSettings); err != nil {
		return nil, err
	}
	if err := writeJSONFile(s.sshBindingsPath, cleaned); err != nil {
		return nil, err
	}
	return s.SSHBindings()
}

func (s *Service) readSSHHosts() ([]SSHHostProfile, error) {
	if strings.TrimSpace(s.sshHostsPath) == "" {
		return nil, fmt.Errorf("ssh hosts path is not configured")
	}

	data, err := os.ReadFile(s.sshHostsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []SSHHostProfile{}, nil
		}
		return nil, fmt.Errorf("read ssh hosts: %w", err)
	}

	var hosts []SSHHostProfile
	if err := json.Unmarshal(data, &hosts); err != nil {
		return nil, fmt.Errorf("parse ssh hosts: %w", err)
	}
	return hosts, nil
}

func (s *Service) writeSSHHosts(hosts []SSHHostProfile) error {
	if strings.TrimSpace(s.sshHostsPath) == "" {
		return fmt.Errorf("ssh hosts path is not configured")
	}
	return writeJSONFile(s.sshHostsPath, hosts)
}

func (s *Service) readSSHBindings() ([]SSHUserBinding, error) {
	if strings.TrimSpace(s.sshBindingsPath) == "" {
		return nil, fmt.Errorf("ssh bindings path is not configured")
	}

	data, err := os.ReadFile(s.sshBindingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []SSHUserBinding{}, nil
		}
		return nil, fmt.Errorf("read ssh bindings: %w", err)
	}

	var bindings []SSHUserBinding
	if err := json.Unmarshal(data, &bindings); err != nil {
		return nil, fmt.Errorf("parse ssh bindings: %w", err)
	}
	return bindings, nil
}

func (s *Service) mergeSSHHosts(incoming []SSHHostProfile) ([]SSHHostProfile, error) {
	existing, err := s.readSSHHosts()
	if err != nil {
		return nil, err
	}
	existingByID := make(map[string]SSHHostProfile, len(existing))
	for _, host := range existing {
		existingByID[strings.ToLower(host.ID)] = host
	}

	cleaned := make([]SSHHostProfile, 0, len(incoming))
	seen := make(map[string]struct{}, len(incoming))
	for _, raw := range incoming {
		host, err := sanitizeSSHHost(raw)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(host.ID)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate ssh host id: %s", host.ID)
		}
		seen[key] = struct{}{}

		if previous, ok := existingByID[key]; ok {
			if strings.TrimSpace(host.Password) == "" {
				host.Password = previous.Password
			}
			if strings.TrimSpace(host.PrivateKey) == "" {
				host.PrivateKey = previous.PrivateKey
			}
			if strings.TrimSpace(host.Passphrase) == "" {
				host.Passphrase = previous.Passphrase
			}
			if strings.TrimSpace(host.HostKeyStatus) == "" {
				host.HostKeyStatus = previous.HostKeyStatus
			}
			if strings.TrimSpace(host.HostKeyFingerprint) == "" {
				host.HostKeyFingerprint = previous.HostKeyFingerprint
			}
		}

		if host.AuthType == "password" && strings.TrimSpace(host.Password) == "" {
			return nil, fmt.Errorf("password is required for ssh host: %s", host.ID)
		}
		if host.AuthType == "private_key" && strings.TrimSpace(host.PrivateKey) == "" {
			return nil, fmt.Errorf("private_key is required for ssh host: %s", host.ID)
		}
		cleaned = append(cleaned, host)
	}
	return cleaned, nil
}

func (s *Service) mergeSSHHostSecrets(raw SSHHostProfile) (SSHHostProfile, error) {
	host, err := sanitizeSSHHost(raw)
	if err != nil {
		return SSHHostProfile{}, err
	}

	if strings.TrimSpace(host.Password) != "" || strings.TrimSpace(host.PrivateKey) != "" {
		return host, nil
	}

	existing, err := s.readSSHHosts()
	if err != nil {
		return SSHHostProfile{}, err
	}
	for _, current := range existing {
		if !strings.EqualFold(current.ID, host.ID) {
			continue
		}
		if strings.TrimSpace(host.Password) == "" {
			host.Password = current.Password
		}
		if strings.TrimSpace(host.PrivateKey) == "" {
			host.PrivateKey = current.PrivateKey
		}
		if strings.TrimSpace(host.Passphrase) == "" {
			host.Passphrase = current.Passphrase
		}
		if strings.TrimSpace(host.HostKeyStatus) == "" {
			host.HostKeyStatus = current.HostKeyStatus
		}
		if strings.TrimSpace(host.HostKeyFingerprint) == "" {
			host.HostKeyFingerprint = current.HostKeyFingerprint
		}
		break
	}
	return host, nil
}

func sanitizeSSHHost(raw SSHHostProfile) (SSHHostProfile, error) {
	host := raw
	host.ID = strings.TrimSpace(host.ID)
	host.Label = strings.TrimSpace(host.Label)
	host.Host = strings.TrimSpace(host.Host)
	host.Username = strings.TrimSpace(host.Username)
	host.AuthType = strings.TrimSpace(host.AuthType)
	host.RemoteShellDefault = strings.TrimSpace(host.RemoteShellDefault)
	host.DefaultWorkdir = strings.TrimSpace(host.DefaultWorkdir)
	host.HostKeyStatus = strings.TrimSpace(host.HostKeyStatus)
	host.HostKeyFingerprint = strings.TrimSpace(host.HostKeyFingerprint)

	if host.Port <= 0 {
		host.Port = 22
	}
	if host.RemoteShellDefault == "" {
		host.RemoteShellDefault = "bash"
	}
	if host.HostKeyStatus == "" {
		host.HostKeyStatus = "unknown"
	}

	host.AllowedPaths = sanitizePathList(host.AllowedPaths)
	if host.ID == "" {
		return SSHHostProfile{}, fmt.Errorf("ssh host id is required")
	}
	if host.Label == "" {
		return SSHHostProfile{}, fmt.Errorf("ssh host label is required")
	}
	if host.Host == "" {
		return SSHHostProfile{}, fmt.Errorf("ssh host address is required")
	}
	if host.Username == "" {
		return SSHHostProfile{}, fmt.Errorf("ssh username is required")
	}
	if host.AuthType != "password" && host.AuthType != "private_key" {
		return SSHHostProfile{}, fmt.Errorf("ssh auth_type must be password or private_key")
	}
	if host.RemoteShellDefault != "bash" && host.RemoteShellDefault != "powershell" {
		return SSHHostProfile{}, fmt.Errorf("remote_shell_default must be bash or powershell")
	}
	if host.HostKeyStatus != "unknown" && host.HostKeyStatus != "trusted" {
		return SSHHostProfile{}, fmt.Errorf("host_key_status must be unknown or trusted")
	}
	if len(host.AllowedPaths) == 0 {
		return SSHHostProfile{}, fmt.Errorf("at least one allowed_path is required")
	}
	return host, nil
}

func sanitizePathList(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	cleaned := make([]string, 0, len(paths))
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		key := strings.ToLower(path)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, path)
	}
	return cleaned
}

func sanitizeSSHBindings(bindings []SSHUserBinding) ([]SSHUserBinding, error) {
	cleaned := make([]SSHUserBinding, 0, len(bindings))
	seen := make(map[string]struct{}, len(bindings))
	for _, raw := range bindings {
		binding := SSHUserBinding{
			UserEmail:     strings.ToLower(strings.TrimSpace(raw.UserEmail)),
			DefaultHostID: strings.TrimSpace(raw.DefaultHostID),
		}
		if binding.UserEmail == "" || binding.DefaultHostID == "" {
			return nil, fmt.Errorf("user_email and default_host_id are required")
		}
		if _, exists := seen[binding.UserEmail]; exists {
			return nil, fmt.Errorf("duplicate user_email binding: %s", binding.UserEmail)
		}
		seen[binding.UserEmail] = struct{}{}
		cleaned = append(cleaned, binding)
	}
	slices.SortFunc(cleaned, func(a, b SSHUserBinding) int {
		return strings.Compare(a.UserEmail, b.UserEmail)
	})
	return cleaned, nil
}

func sanitizeSSHHostsForResponse(hosts []SSHHostProfile) []SSHHostProfile {
	resp := make([]SSHHostProfile, 0, len(hosts))
	for _, host := range hosts {
		item := host
		item.HasPassword = strings.TrimSpace(item.Password) != ""
		item.HasPrivateKey = strings.TrimSpace(item.PrivateKey) != ""
		item.Password = ""
		item.PrivateKey = ""
		item.Passphrase = ""
		resp = append(resp, item)
	}
	return resp
}

func writeJSONFile(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
