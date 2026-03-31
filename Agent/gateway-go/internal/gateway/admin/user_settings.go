package admin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const primaryLocalTerminalUser = "3223659402@qq.com"

func (s *Service) Users() (*UsersResponse, error) {
	settings, err := s.readUserSettings()
	if err != nil {
		return nil, err
	}
	legacyBindings, err := s.readSSHBindings()
	if err != nil {
		return nil, err
	}
	observed, err := s.collectObservedUsers()
	if err != nil {
		return nil, err
	}

	byEmail := make(map[string]UserSummary)
	for _, workspace := range settings {
		email := normalizeEmail(workspace.UserEmail)
		if email == "" {
			continue
		}
		byEmail[email] = UserSummary{
			UserEmail:   email,
			Label:       firstNonEmptyString(strings.TrimSpace(workspace.Label), labelFromEmail(email)),
			LastSeenAt:  observed[email],
			HasSettings: true,
		}
	}
	for _, binding := range legacyBindings {
		email := normalizeEmail(binding.UserEmail)
		if email == "" {
			continue
		}
		item := byEmail[email]
		item.UserEmail = email
		item.Label = firstNonEmptyString(item.Label, labelFromEmail(email))
		item.LastSeenAt = firstNonEmptyString(item.LastSeenAt, observed[email])
		item.HasLegacy = true
		byEmail[email] = item
	}
	for email, lastSeenAt := range observed {
		item := byEmail[email]
		item.UserEmail = email
		item.Label = firstNonEmptyString(item.Label, labelFromEmail(email))
		item.LastSeenAt = firstNonEmptyString(item.LastSeenAt, lastSeenAt)
		byEmail[email] = item
	}

	users := make([]UserSummary, 0, len(byEmail))
	for _, item := range byEmail {
		users = append(users, item)
	}
	slices.SortFunc(users, func(a, b UserSummary) int {
		if cmp := strings.Compare(a.UserEmail, b.UserEmail); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Label, b.Label)
	})

	return &UsersResponse{
		Users:      users,
		ConfigPath: s.userSettingsPath,
	}, nil
}

func (s *Service) UserWorkspace(userEmail string) (*UserWorkspaceResponse, error) {
	globalAllowedPaths, err := s.globalAllowedPaths()
	if err != nil {
		return nil, err
	}
	hosts, err := s.readSSHHosts()
	if err != nil {
		return nil, err
	}

	workspace, err := s.resolveUserWorkspace(userEmail, globalAllowedPaths, hosts)
	if err != nil {
		return nil, err
	}

	return &UserWorkspaceResponse{
		Workspace:                 workspace,
		ConfigPath:                s.userSettingsPath,
		GlobalAllowedPaths:        globalAllowedPaths,
		AvailableExecutionTargets: buildAvailableExecutionTargets(workspace, hosts),
		RestartRequired:           true,
		LegacyBindingsPath:        s.sshBindingsPath,
	}, nil
}

func (s *Service) UpdateUserWorkspace(workspace UserWorkspace) (*UserWorkspaceResponse, error) {
	globalAllowedPaths, err := s.globalAllowedPaths()
	if err != nil {
		return nil, err
	}

	cleaned, err := sanitizeUserWorkspace(workspace, globalAllowedPaths)
	if err != nil {
		return nil, err
	}

	hosts, err := s.readSSHHosts()
	if err != nil {
		return nil, err
	}
	if cleaned.DefaultSSHHostID != "" && !hasSSHHostID(hosts, cleaned.DefaultSSHHostID) {
		return nil, fmt.Errorf("unknown default_ssh_host_id: %s", cleaned.DefaultSSHHostID)
	}
	cleaned.EnabledExecutionTargets, err = sanitizeExecutionTargets(
		cleaned.EnabledExecutionTargets,
		hosts,
	)
	if err != nil {
		return nil, err
	}

	settings, err := s.readUserSettings()
	if err != nil {
		return nil, err
	}

	next := make([]UserWorkspace, 0, len(settings)+1)
	replaced := false
	for _, item := range settings {
		if normalizeEmail(item.UserEmail) == cleaned.UserEmail {
			next = append(next, cleaned)
			replaced = true
			continue
		}
		next = append(next, item)
	}
	if !replaced {
		next = append(next, cleaned)
	}
	slices.SortFunc(next, func(a, b UserWorkspace) int {
		return strings.Compare(normalizeEmail(a.UserEmail), normalizeEmail(b.UserEmail))
	})

	if err := s.writeUserSettings(next); err != nil {
		return nil, err
	}
	if err := s.syncLegacyBindingsFromUserSettings(next); err != nil {
		return nil, err
	}

	return s.UserWorkspace(cleaned.UserEmail)
}

func (s *Service) readUserSettings() ([]UserWorkspace, error) {
	if strings.TrimSpace(s.userSettingsPath) == "" {
		return []UserWorkspace{}, nil
	}

	data, err := os.ReadFile(s.userSettingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []UserWorkspace{}, nil
		}
		return nil, fmt.Errorf("read user settings: %w", err)
	}

	var workspaces []UserWorkspace
	if err := json.Unmarshal(data, &workspaces); err == nil {
		return sanitizeUserWorkspaceList(workspaces), nil
	}

	var wrapped struct {
		Users []UserWorkspace `json:"users"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, fmt.Errorf("parse user settings: %w", err)
	}
	return sanitizeUserWorkspaceList(wrapped.Users), nil
}

func (s *Service) writeUserSettings(items []UserWorkspace) error {
	if strings.TrimSpace(s.userSettingsPath) == "" {
		return fmt.Errorf("user settings path is not configured")
	}
	return writeJSONFile(s.userSettingsPath, sanitizeUserWorkspaceList(items))
}

func (s *Service) resolveUserWorkspace(
	userEmail string,
	globalAllowedPaths []string,
	hosts []SSHHostProfile,
) (UserWorkspace, error) {
	email := normalizeEmail(userEmail)
	if email == "" {
		return UserWorkspace{}, fmt.Errorf("user_email is required")
	}

	settings, err := s.readUserSettings()
	if err != nil {
		return UserWorkspace{}, err
	}
	for _, item := range settings {
		if normalizeEmail(item.UserEmail) != email {
			continue
		}
		if len(item.TerminalAllowedPaths) == 0 {
			item.TerminalAllowedPaths = append([]string{}, globalAllowedPaths...)
		}
		item.UserEmail = email
		item.Label = firstNonEmptyString(strings.TrimSpace(item.Label), labelFromEmail(email))
		item.EnabledExecutionTargets = hydrateExecutionTargets(item, hosts)
		return item, nil
	}

	legacyBindings, err := s.readSSHBindings()
	if err != nil {
		return UserWorkspace{}, err
	}
	defaultSSHHostID := ""
	for _, binding := range legacyBindings {
		if normalizeEmail(binding.UserEmail) == email {
			defaultSSHHostID = strings.TrimSpace(binding.DefaultHostID)
			break
		}
	}

	workspace := UserWorkspace{
		UserEmail:            email,
		Label:                labelFromEmail(email),
		TerminalAllowedPaths: append([]string{}, globalAllowedPaths...),
		DefaultSSHHostID:     defaultSSHHostID,
	}
	workspace.EnabledExecutionTargets = hydrateExecutionTargets(workspace, hosts)
	return workspace, nil
}

func (s *Service) globalAllowedPaths() ([]string, error) {
	cfgPath := strings.TrimSpace(s.toolRouterConfigPath)
	if cfgPath == "" {
		return nil, fmt.Errorf("tool router config path is not configured")
	}
	return s.readAllowedPaths(cfgPath)
}

func (s *Service) syncLegacyBindingsFromUserSettings(settings []UserWorkspace) error {
	bindings := make([]SSHUserBinding, 0, len(settings))
	for _, item := range settings {
		if strings.TrimSpace(item.DefaultSSHHostID) == "" {
			continue
		}
		bindings = append(bindings, SSHUserBinding{
			UserEmail:     normalizeEmail(item.UserEmail),
			DefaultHostID: strings.TrimSpace(item.DefaultSSHHostID),
		})
	}
	cleaned, err := sanitizeSSHBindings(bindings)
	if err != nil {
		return err
	}
	return writeJSONFile(s.sshBindingsPath, cleaned)
}

func (s *Service) collectObservedUsers() (map[string]string, error) {
	out := make(map[string]string)
	for _, dir := range []string{s.cfg.Logs.SessionLogDir, s.cfg.Admin.ToolLogDir, s.cfg.Logs.AuditLogDir} {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		if err != nil {
			return nil, err
		}
		for _, path := range files {
			items, err := readJSONLFile(path, nil)
			if err != nil {
				continue
			}
			for _, item := range items {
				email := normalizeEmail(stringField(item, "user_email"))
				if email == "" {
					continue
				}
				timeValue := stringField(item, "time")
				if out[email] == "" || strings.Compare(timeValue, out[email]) > 0 {
					out[email] = timeValue
				}
			}
		}
	}
	return out, nil
}

func sanitizeUserWorkspaceList(items []UserWorkspace) []UserWorkspace {
	seen := make(map[string]struct{}, len(items))
	cleaned := make([]UserWorkspace, 0, len(items))
	for _, item := range items {
		email := normalizeEmail(item.UserEmail)
		if email == "" {
			continue
		}
		if _, exists := seen[email]; exists {
			continue
		}
		seen[email] = struct{}{}
		item.UserEmail = email
		item.Label = strings.TrimSpace(item.Label)
		item.DefaultLocalWorkdir = strings.TrimSpace(item.DefaultLocalWorkdir)
		item.DefaultSSHHostID = strings.TrimSpace(item.DefaultSSHHostID)
		item.EnabledExecutionTargets = sanitizeUserStringList(item.EnabledExecutionTargets)
		item.EnabledMCPServerIDs = sanitizeUserStringList(item.EnabledMCPServerIDs)
		item.TerminalAllowedPaths = sanitizePathList(item.TerminalAllowedPaths)
		if item.DisabledMCPToolsByServer == nil {
			item.DisabledMCPToolsByServer = map[string][]string{}
		}
		for key, values := range item.DisabledMCPToolsByServer {
			item.DisabledMCPToolsByServer[strings.TrimSpace(key)] = sanitizeUserStringList(values)
			if strings.TrimSpace(key) == "" {
				delete(item.DisabledMCPToolsByServer, key)
			}
		}
		cleaned = append(cleaned, item)
	}
	slices.SortFunc(cleaned, func(a, b UserWorkspace) int {
		return strings.Compare(a.UserEmail, b.UserEmail)
	})
	return cleaned
}

func sanitizeUserWorkspace(raw UserWorkspace, globalAllowedPaths []string) (UserWorkspace, error) {
	item := raw
	item.UserEmail = normalizeEmail(item.UserEmail)
	item.Label = strings.TrimSpace(item.Label)
	item.DefaultLocalWorkdir = cleanPortablePath(item.DefaultLocalWorkdir)
	if item.DefaultLocalWorkdir == "." {
		item.DefaultLocalWorkdir = ""
	}
	item.DefaultSSHHostID = strings.TrimSpace(item.DefaultSSHHostID)
	item.EnabledMCPServerIDs = sanitizeUserStringList(item.EnabledMCPServerIDs)
	item.TerminalAllowedPaths = sanitizePathList(item.TerminalAllowedPaths)
	if len(item.TerminalAllowedPaths) == 0 {
		item.TerminalAllowedPaths = append([]string{}, globalAllowedPaths...)
	}

	if item.UserEmail == "" {
		return UserWorkspace{}, fmt.Errorf("user_email is required")
	}
	if item.Label == "" {
		item.Label = labelFromEmail(item.UserEmail)
	}
	for _, path := range item.TerminalAllowedPaths {
		if !isAllowedUserPath(path, globalAllowedPaths) {
			return UserWorkspace{}, fmt.Errorf("user path must stay inside system allowed paths: %s", path)
		}
	}
	if item.DefaultLocalWorkdir != "" && !isAllowedUserPath(item.DefaultLocalWorkdir, item.TerminalAllowedPaths) {
		return UserWorkspace{}, fmt.Errorf("default_local_workdir must stay inside user terminal_allowed_paths: %s", item.DefaultLocalWorkdir)
	}
	item.EnabledExecutionTargets = sanitizeUserStringList(item.EnabledExecutionTargets)
	return item, nil
}

func buildAvailableExecutionTargets(
	workspace UserWorkspace,
	hosts []SSHHostProfile,
) []ExecutionTargetSummary {
	allowed := make([]ExecutionTargetSummary, 0, len(workspace.EnabledExecutionTargets))
	hostByID := make(map[string]SSHHostProfile, len(hosts))
	for _, host := range hosts {
		hostByID[strings.ToLower(strings.TrimSpace(host.ID))] = host
	}

	for _, target := range workspace.EnabledExecutionTargets {
		switch {
		case strings.EqualFold(target, "local"):
			workdir := workspace.DefaultLocalWorkdir
			if workdir == "" && len(workspace.TerminalAllowedPaths) > 0 {
				workdir = workspace.TerminalAllowedPaths[0]
			}
			allowed = append(allowed, ExecutionTargetSummary{
				TargetID:       "local",
				Kind:           "local",
				Label:          "本机 / Local",
				Shells:         []string{"powershell", "wsl-bash"},
				DefaultWorkdir: workdir,
				AllowedPaths:   append([]string{}, workspace.TerminalAllowedPaths...),
				RecommendedUse: "适合本机脚本、仓库操作、文件打包与传输编排",
			})
		case strings.HasPrefix(strings.ToLower(target), "ssh:"):
			hostID := strings.TrimSpace(target[4:])
			host, ok := hostByID[strings.ToLower(hostID)]
			if !ok || !host.Enabled {
				continue
			}
			workdir := strings.TrimSpace(host.DefaultWorkdir)
			if workdir == "" && len(host.AllowedPaths) > 0 {
				workdir = host.AllowedPaths[0]
			}
			shells := []string{host.RemoteShellDefault}
			if host.RemoteShellDefault == "bash" {
				shells = []string{"bash", "powershell"}
			} else {
				shells = []string{"powershell", "bash"}
			}
			allowed = append(allowed, ExecutionTargetSummary{
				TargetID:       "ssh:" + host.ID,
				Kind:           "ssh",
				Label:          firstNonEmptyString(host.Label, host.ID),
				Shells:         shells,
				DefaultWorkdir: workdir,
				AllowedPaths:   append([]string{}, host.AllowedPaths...),
				RecommendedUse: "适合远端目录检查、运行题目、查看远端日志与进程",
			})
		}
	}

	return allowed
}

func hydrateExecutionTargets(workspace UserWorkspace, hosts []SSHHostProfile) []string {
	targets := sanitizeUserStringList(workspace.EnabledExecutionTargets)
	if len(targets) > 0 {
		return targets
	}

	if normalizeEmail(workspace.UserEmail) == primaryLocalTerminalUser {
		targets = append(targets, "local")
	}
	if strings.TrimSpace(workspace.DefaultSSHHostID) != "" && hasSSHHostID(hosts, workspace.DefaultSSHHostID) {
		targets = append(targets, "ssh:"+strings.TrimSpace(workspace.DefaultSSHHostID))
	}
	return sanitizeUserStringList(targets)
}

func sanitizeExecutionTargets(targets []string, hosts []SSHHostProfile) ([]string, error) {
	cleaned := sanitizeUserStringList(targets)
	for _, target := range cleaned {
		switch {
		case strings.EqualFold(target, "local"):
		case strings.HasPrefix(strings.ToLower(target), "ssh:"):
			hostID := strings.TrimSpace(target[4:])
			if hostID == "" {
				return nil, fmt.Errorf("ssh execution target must include host id")
			}
			if !hasSSHHostID(hosts, hostID) {
				return nil, fmt.Errorf("unknown ssh execution target: %s", hostID)
			}
		default:
			return nil, fmt.Errorf("unsupported execution target: %s", target)
		}
	}
	return cleaned, nil
}

func isAllowedUserPath(path string, roots []string) bool {
	cleanPath := normalizedPathKey(path)
	for _, root := range roots {
		cleanRoot := normalizedPathKey(root)
		if cleanPath == cleanRoot || strings.HasPrefix(cleanPath, cleanRoot+"/") {
			return true
		}
	}
	return false
}

func hasSSHHostID(hosts []SSHHostProfile, hostID string) bool {
	key := strings.ToLower(strings.TrimSpace(hostID))
	for _, host := range hosts {
		if strings.ToLower(strings.TrimSpace(host.ID)) == key {
			return true
		}
	}
	return false
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sanitizeUserStringList(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	cleaned := make([]string, 0, len(items))
	for _, raw := range items {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, item)
	}
	return cleaned
}

func normalizedPathKey(value string) string {
	return normalizedPortablePathKey(value)
}

func labelFromEmail(email string) string {
	if email == "" {
		return ""
	}
	local := email
	if pivot := strings.Index(email, "@"); pivot > 0 {
		local = email[:pivot]
	}
	local = strings.TrimSpace(strings.ReplaceAll(local, ".", " "))
	local = strings.ReplaceAll(local, "_", " ")
	local = strings.ReplaceAll(local, "-", " ")
	if local == "" {
		return email
	}
	return local
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
