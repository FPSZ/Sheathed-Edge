package admin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type mcpToolCacheFile struct {
	Servers []mcpToolCacheEntry `json:"servers"`
}

type mcpToolCacheEntry struct {
	ServerID         string          `json:"server_id"`
	Tools            []mcpCachedTool `json:"tools"`
	LastDiscoveredAt string          `json:"last_discovered_at,omitempty"`
	LastError        string          `json:"last_error,omitempty"`
}

type mcpCachedTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type toolRouterMCPValidateResponse struct {
	OK                     bool         `json:"ok"`
	Summary                string       `json:"summary"`
	EffectiveOpenWebUIType string       `json:"effective_openwebui_type,omitempty"`
	EffectiveConnectionURL string       `json:"effective_connection_url,omitempty"`
	Error                  *ErrorDetail `json:"error,omitempty"`
}

type toolRouterMCPDiscoverResponse struct {
	OK                     bool                `json:"ok"`
	Summary                string              `json:"summary"`
	ServerID               string              `json:"server_id"`
	Tools                  []MCPDiscoveredTool `json:"tools"`
	LastDiscoveredAt       string              `json:"last_discovered_at,omitempty"`
	EffectiveOpenWebUIType string              `json:"effective_openwebui_type,omitempty"`
	EffectiveConnectionURL string              `json:"effective_connection_url,omitempty"`
	Error                  *ErrorDetail        `json:"error,omitempty"`
}

func (s *Service) MCPServers() (*MCPServersResponse, error) {
	servers, err := s.readMCPServers()
	if err != nil {
		return nil, err
	}
	cache, err := s.readMCPToolCache()
	if err != nil {
		return nil, err
	}
	runtime, err := s.MCPRuntimeStatus(context.Background())
	if err != nil {
		runtime = &MCPRuntimeStatusResponse{Servers: []MCPRuntimeEntry{}}
	}
	runtimeByID := make(map[string]MCPRuntimeEntry, len(runtime.Servers))
	for _, item := range runtime.Servers {
		runtimeByID[strings.ToLower(item.ServerID)] = item
	}
	cacheByID := make(map[string]mcpToolCacheEntry, len(cache.Servers))
	for _, item := range cache.Servers {
		cacheByID[strings.ToLower(strings.TrimSpace(item.ServerID))] = item
	}

	states := make([]MCPServerState, 0, len(servers))
	for _, profile := range servers {
		key := strings.ToLower(profile.ID)
		cacheEntry := cacheByID[key]
		tools := make([]MCPDiscoveredTool, 0, len(cacheEntry.Tools))
		for _, tool := range cacheEntry.Tools {
			tools = append(tools, MCPDiscoveredTool{
				Name:        tool.Name,
				Description: tool.Description,
				Disabled:    slices.Contains(profile.DisabledTools, tool.Name),
			})
		}
		runtimeEntry, ok := runtimeByID[key]
		if !ok {
			runtimeEntry = MCPRuntimeEntry{
				ServerID: profile.ID,
				Label:    profile.Label,
				Enabled:  profile.Enabled,
				Kind:     profile.Kind,
				Status:   "unknown",
			}
		}
		states = append(states, MCPServerState{
			Profile:                profile,
			DiscoveredTools:        tools,
			LastDiscoveredAt:       cacheEntry.LastDiscoveredAt,
			RuntimeStatus:          runtimeEntry,
			EffectiveOpenWebUIType: runtimeEntry.EffectiveOpenWebUIType,
			EffectiveConnectionURL: runtimeEntry.EffectiveConnectionURL,
			LastError:              firstNonEmpty(runtimeEntry.LastError, cacheEntry.LastError),
		})
	}

	return &MCPServersResponse{
		Servers:        states,
		ConfigPath:     s.mcpServersPath,
		ToolCachePath:  s.mcpToolCachePath,
		ToolRouterBase: strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/"),
	}, nil
}

func (s *Service) UpdateMCPServers(ctx context.Context, servers []MCPServerProfile) (*MCPServersResponse, error) {
	_ = ctx
	cleaned, err := sanitizeMCPServers(servers)
	if err != nil {
		return nil, err
	}
	if err := writeJSONFile(s.mcpServersPath, cleaned); err != nil {
		return nil, err
	}
	return s.MCPServers()
}

func (s *Service) ValidateMCPServer(ctx context.Context, server MCPServerProfile) (*MCPValidateResponse, error) {
	cleaned, err := sanitizeMCPServer(server)
	if err != nil {
		return nil, err
	}
	var resp toolRouterMCPValidateResponse
	if err := s.callToolRouterJSON(ctx, http.MethodPost, "/internal/mcp/validate", map[string]any{
		"server": cleaned,
	}, &resp); err != nil {
		return nil, err
	}
	return &MCPValidateResponse{
		OK:                     resp.OK,
		Summary:                resp.Summary,
		EffectiveOpenWebUIType: resp.EffectiveOpenWebUIType,
		EffectiveConnectionURL: resp.EffectiveConnectionURL,
		Error:                  resp.Error,
	}, nil
}

func (s *Service) DiscoverMCPTools(ctx context.Context, serverID string) (*MCPDiscoverToolsResponse, error) {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return nil, fmt.Errorf("server_id is required")
	}
	var resp toolRouterMCPDiscoverResponse
	if err := s.callToolRouterJSON(ctx, http.MethodPost, "/internal/mcp/discover-tools", map[string]any{
		"server_id": serverID,
	}, &resp); err != nil {
		return nil, err
	}
	return &MCPDiscoverToolsResponse{
		OK:                     resp.OK,
		Summary:                resp.Summary,
		ServerID:               resp.ServerID,
		Tools:                  resp.Tools,
		LastDiscoveredAt:       resp.LastDiscoveredAt,
		EffectiveOpenWebUIType: resp.EffectiveOpenWebUIType,
		EffectiveConnectionURL: resp.EffectiveConnectionURL,
		Error:                  resp.Error,
	}, nil
}

func (s *Service) MCPRuntimeStatus(ctx context.Context) (*MCPRuntimeStatusResponse, error) {
	var resp MCPRuntimeStatusResponse
	if err := s.callToolRouterJSON(ctx, http.MethodGet, "/internal/mcp/runtime", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) MCPOpenWebUIPreview() (*MCPOpenWebUIPreviewResponse, error) {
	servers, err := s.readMCPServers()
	if err != nil {
		return nil, err
	}
	runtime, err := s.MCPRuntimeStatus(context.Background())
	if err != nil {
		runtime = &MCPRuntimeStatusResponse{Servers: []MCPRuntimeEntry{}}
	}
	runtimeByID := make(map[string]MCPRuntimeEntry, len(runtime.Servers))
	for _, item := range runtime.Servers {
		runtimeByID[strings.ToLower(item.ServerID)] = item
	}

	connections := make([]OpenWebUIToolConnection, 0, len(servers))
	for _, server := range servers {
		if !server.Enabled {
			continue
		}
		runtimeEntry := runtimeByID[strings.ToLower(server.ID)]
		connection := buildOpenWebUIConnection(server, runtimeEntry)
		connections = append(connections, connection)
	}
	data, err := json.MarshalIndent(connections, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode openwebui preview: %w", err)
	}
	data = append(data, '\n')
	return &MCPOpenWebUIPreviewResponse{
		Connections:               connections,
		ToolServerConnectionsJSON: string(data),
		RestartRequired:           true,
	}, nil
}

func (s *Service) readMCPServers() ([]MCPServerProfile, error) {
	if strings.TrimSpace(s.mcpServersPath) == "" {
		return nil, fmt.Errorf("mcp servers path is not configured")
	}
	data, err := os.ReadFile(s.mcpServersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []MCPServerProfile{}, nil
		}
		return nil, fmt.Errorf("read mcp servers: %w", err)
	}
	var servers []MCPServerProfile
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("parse mcp servers: %w", err)
	}
	return sanitizeMCPServers(servers)
}

func (s *Service) readMCPToolCache() (*mcpToolCacheFile, error) {
	if strings.TrimSpace(s.mcpToolCachePath) == "" {
		return nil, fmt.Errorf("mcp tool cache path is not configured")
	}
	data, err := os.ReadFile(s.mcpToolCachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &mcpToolCacheFile{Servers: []mcpToolCacheEntry{}}, nil
		}
		return nil, fmt.Errorf("read mcp tool cache: %w", err)
	}
	var payload mcpToolCacheFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse mcp tool cache: %w", err)
	}
	return &payload, nil
}

func (s *Service) callToolRouterJSON(ctx context.Context, method, path string, payload any, target any) error {
	timeoutMS := s.cfg.ToolRouter.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = s.cfg.Admin.TimeoutMS
	}
	if timeoutMS <= 0 {
		timeoutMS = 5000
	}

	var bodyBytes []byte
	var err error
	if payload != nil {
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode tool-router request: %w", err)
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS+3000)*time.Millisecond)
	defer cancel()

	url := strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(reqCtx, method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build tool-router request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: time.Duration(timeoutMS+3000) * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call tool-router %s: %w", path, err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode tool-router response %s: %w", path, err)
	}
	return nil
}

func sanitizeMCPServers(servers []MCPServerProfile) ([]MCPServerProfile, error) {
	cleaned := make([]MCPServerProfile, 0, len(servers))
	seen := make(map[string]struct{}, len(servers))
	for _, raw := range servers {
		server, err := sanitizeMCPServer(raw)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(server.ID)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate mcp server id: %s", server.ID)
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, server)
	}
	slices.SortFunc(cleaned, func(a, b MCPServerProfile) int {
		return strings.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
	})
	return cleaned, nil
}

func sanitizeMCPServer(raw MCPServerProfile) (MCPServerProfile, error) {
	server := raw
	server.ID = strings.TrimSpace(server.ID)
	server.Label = strings.TrimSpace(server.Label)
	server.Kind = strings.TrimSpace(server.Kind)
	server.Description = strings.TrimSpace(server.Description)
	server.AuthType = strings.TrimSpace(server.AuthType)
	server.Notes = strings.TrimSpace(server.Notes)
	server.URL = strings.TrimSpace(server.URL)
	server.Workdir = filepath.Clean(strings.TrimSpace(server.Workdir))
	if server.Workdir == "." {
		server.Workdir = ""
	}
	if server.TimeoutMS <= 0 {
		server.TimeoutMS = 30000
	}
	if !raw.VerifyTLS {
		server.VerifyTLS = false
	} else {
		server.VerifyTLS = true
	}
	server.PluginScope = sanitizeEnumList(server.PluginScope)
	server.DisabledTools = sanitizePathList(server.DisabledTools)
	server.Command = sanitizeStringList(server.Command)
	server.Env = sanitizeStringMap(server.Env)
	server.Headers = sanitizeStringMap(server.Headers)
	server.AuthPayload = sanitizeStringMap(server.AuthPayload)

	if server.ID == "" {
		return MCPServerProfile{}, fmt.Errorf("mcp server id is required")
	}
	if server.Label == "" {
		return MCPServerProfile{}, fmt.Errorf("mcp server label is required")
	}
	switch server.Kind {
	case "native_streamable_http", "mcpo_stdio", "mcpo_sse":
	default:
		return MCPServerProfile{}, fmt.Errorf("unsupported mcp kind: %s", server.Kind)
	}
	switch server.AuthType {
	case "", "none":
		server.AuthType = "none"
	case "bearer", "basic", "header":
	default:
		return MCPServerProfile{}, fmt.Errorf("unsupported mcp auth_type: %s", server.AuthType)
	}
	if len(server.PluginScope) == 0 {
		return MCPServerProfile{}, fmt.Errorf("at least one plugin_scope is required")
	}
	for _, scope := range server.PluginScope {
		if scope != "awdp" && scope != "web" && scope != "pwn" {
			return MCPServerProfile{}, fmt.Errorf("unsupported plugin_scope: %s", scope)
		}
	}

	switch server.Kind {
	case "native_streamable_http", "mcpo_sse":
		if server.URL == "" {
			return MCPServerProfile{}, fmt.Errorf("url is required for %s", server.Kind)
		}
	case "mcpo_stdio":
		if len(server.Command) == 0 {
			return MCPServerProfile{}, fmt.Errorf("command is required for mcpo_stdio")
		}
		if server.Workdir == "" {
			return MCPServerProfile{}, fmt.Errorf("workdir is required for mcpo_stdio")
		}
		if !filepath.IsAbs(server.Workdir) {
			return MCPServerProfile{}, fmt.Errorf("workdir must be absolute for mcpo_stdio")
		}
	}
	if server.AuthType == "bearer" && strings.TrimSpace(server.AuthPayload["token"]) == "" {
		return MCPServerProfile{}, fmt.Errorf("auth_payload.token is required for bearer auth")
	}
	if server.AuthType == "basic" {
		if strings.TrimSpace(server.AuthPayload["username"]) == "" || strings.TrimSpace(server.AuthPayload["password"]) == "" {
			return MCPServerProfile{}, fmt.Errorf("auth_payload.username and auth_payload.password are required for basic auth")
		}
	}
	if server.AuthType == "header" {
		if strings.TrimSpace(server.AuthPayload["name"]) == "" || strings.TrimSpace(server.AuthPayload["value"]) == "" {
			return MCPServerProfile{}, fmt.Errorf("auth_payload.name and auth_payload.value are required for header auth")
		}
	}

	return server, nil
}

func sanitizeStringList(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	cleaned := make([]string, 0, len(items))
	for _, raw := range items {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func sanitizeEnumList(items []string) []string {
	cleaned := sanitizeStringList(items)
	slices.Sort(cleaned)
	return cleaned
}

func sanitizeStringMap(items map[string]string) map[string]string {
	if len(items) == 0 {
		return map[string]string{}
	}
	cleaned := make(map[string]string, len(items))
	for rawKey, rawValue := range items {
		key := strings.TrimSpace(rawKey)
		value := strings.TrimSpace(rawValue)
		if key == "" || value == "" {
			continue
		}
		cleaned[key] = value
	}
	return cleaned
}

func buildOpenWebUIConnection(server MCPServerProfile, runtime MCPRuntimeEntry) OpenWebUIToolConnection {
	connection := OpenWebUIToolConnection{
		ID:       fmt.Sprintf("mcp-%s", server.ID),
		Name:     server.Label,
		Enabled:  server.Enabled,
		AuthType: normalizeOpenWebUIAuthType(server.AuthType),
		Config: map[string]any{
			"access_grants": []map[string]string{
				{
					"principal_type": "user",
					"principal_id":   "*",
					"permission":     "read",
				},
			},
		},
		Info: map[string]any{
			"id":          fmt.Sprintf("mcp-%s", server.ID),
			"name":        server.Label,
			"description": firstNonEmpty(server.Description, server.Kind),
		},
	}
	if len(server.DisabledTools) > 0 {
		connection.FunctionNameFilterList = strings.Join(server.DisabledTools, ",")
	}
	switch server.Kind {
	case "native_streamable_http":
		connection.Type = "mcp"
		connection.URL = server.URL
	case "mcpo_stdio", "mcpo_sse":
		connection.Type = "openapi"
		connection.SpecType = "url"
		connection.URL = mcpoBaseURLFromRuntime(runtime)
		if connection.URL == "" {
			connection.URL = runtime.EffectiveConnectionURL
		}
		connection.Path = fmt.Sprintf("/%s/openapi.json", server.ID)
		if connection.URL == "" {
			connection.URL = "http://127.0.0.1:8092"
		}
	}

	switch server.AuthType {
	case "bearer":
		connection.Key = server.AuthPayload["token"]
	case "header":
		connection.Headers = map[string]string{
			server.AuthPayload["name"]: server.AuthPayload["value"],
		}
	case "basic":
		connection.Headers = map[string]string{
			"Authorization": basicAuthHeader(server.AuthPayload["username"], server.AuthPayload["password"]),
		}
	}
	return connection
}

func normalizeOpenWebUIAuthType(authType string) string {
	switch authType {
	case "bearer":
		return "bearer"
	case "basic", "header":
		return "custom"
	default:
		return "none"
	}
}

func mcpoBaseURLFromRuntime(runtime MCPRuntimeEntry) string {
	if runtime.BridgePort <= 0 {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d", runtime.BridgePort)
}

func basicAuthHeader(username, password string) string {
	token := []byte(strings.TrimSpace(username) + ":" + strings.TrimSpace(password))
	return "Basic " + basicAuthEncoder(token)
}

func basicAuthEncoder(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
