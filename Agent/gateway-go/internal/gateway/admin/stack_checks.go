package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *Service) StartAll(ctx context.Context) (*StartAllResponse, error) {
	results := make([]ServiceActionResult, 0, 5)

	steps := []struct {
		name    string
		check   func(context.Context) error
		start   func(context.Context) error
		timeout time.Duration
	}{
		{
			name:    serviceHostAgent,
			check:   s.host.Health,
			start:   s.startHostAgent,
			timeout: 12 * time.Second,
		},
		{
			name: serviceToolRouter,
			check: func(ctx context.Context) error {
				return s.simpleGET(ctx, strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/")+"/healthz")
			},
			start:   s.startToolRouter,
			timeout: 15 * time.Second,
		},
		{
			name: serviceOpenWebUI,
			check: func(ctx context.Context) error {
				return s.simpleGET(ctx, strings.TrimRight(s.cfg.Admin.OpenWebUIURL, "/")+"/health")
			},
			start: func(ctx context.Context) error {
				return s.startDetached(ctx, "open-webui", "/mnt/d/AI/Local/Workflows/wsl/start-open-webui.sh")
			},
			timeout: 45 * time.Second,
		},
		{
			name:    serviceLlama,
			check:   s.provider.Health,
			start:   s.host.Start,
			timeout: 45 * time.Second,
		},
	}

	for _, step := range steps {
		results = append(results, s.ensureNamedService(ctx, step.name, step.check, step.start, step.timeout))
	}

	ok := true
	for _, item := range results {
		if !item.OK {
			ok = false
			break
		}
	}
	return &StartAllResponse{OK: ok, Results: results}, nil
}

func (s *Service) SelfCheck(ctx context.Context) (*SelfCheckResponse, error) {
	checks := []SelfCheckItem{
		s.runCheck(ctx, "gateway.health", "Gateway health", func(checkCtx context.Context) (string, map[string]any, error) {
			url := s.localGatewayBaseURL() + "/healthz"
			if err := s.simpleGET(checkCtx, url); err != nil {
				return "", nil, err
			}
			return "gateway /healthz ok", map[string]any{"url": url}, nil
		}),
		s.runCheck(ctx, "host-agent.health", "Host agent", func(checkCtx context.Context) (string, map[string]any, error) {
			if err := s.host.Health(checkCtx); err != nil {
				return "", nil, err
			}
			return "host-agent reachable", map[string]any{"url": strings.TrimRight(s.cfg.Admin.HostAgentURL, "/") + "/healthz"}, nil
		}),
		s.runCheck(ctx, "tool-router.health", "Tool router", func(checkCtx context.Context) (string, map[string]any, error) {
			url := strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/") + "/healthz"
			if err := s.simpleGET(checkCtx, url); err != nil {
				return "", nil, err
			}
			return "tool-router /healthz ok", map[string]any{"url": url}, nil
		}),
		s.runCheck(ctx, "open-webui.health", "Open WebUI", func(checkCtx context.Context) (string, map[string]any, error) {
			url := strings.TrimRight(s.cfg.Admin.OpenWebUIURL, "/") + "/health"
			if err := s.simpleGET(checkCtx, url); err != nil {
				return "", nil, err
			}
			return "open-webui /health ok", map[string]any{"url": url}, nil
		}),
		s.runCheck(ctx, "llama.ready", "Llama server", func(checkCtx context.Context) (string, map[string]any, error) {
			if _, err := s.EnsureModelReady(checkCtx, s.selfCheckModelID(ctx)); err != nil {
				return "", nil, err
			}
			deadline := time.Now().Add(2 * time.Minute)
			for time.Now().Before(deadline) {
				if err := s.provider.Health(checkCtx); err == nil {
					return "provider health ok", map[string]any{"base_url": s.cfg.LlamaServer.BaseURL}, nil
				}
				time.Sleep(2 * time.Second)
			}
			return "", nil, fmt.Errorf("provider did not become healthy in time")
		}),
		s.runCheck(ctx, "gateway.models", "Model list", func(checkCtx context.Context) (string, map[string]any, error) {
			var payload struct {
				Data []map[string]any `json:"data"`
			}
			if err := s.getJSON(checkCtx, s.localGatewayBaseURL()+"/v1/models", &payload); err != nil {
				return "", nil, err
			}
			return fmt.Sprintf("returned %d models", len(payload.Data)), map[string]any{"count": len(payload.Data)}, nil
		}),
		s.runCheck(ctx, "gateway.chat", "Chat", func(checkCtx context.Context) (string, map[string]any, error) {
			body := map[string]any{
				"model":      s.selfCheckModelID(ctx),
				"messages":   []map[string]any{{"role": "user", "content": "reply with ok"}},
				"stream":     false,
				"user_email": "admin-self-check@local",
			}
			var payload map[string]any
			if err := s.postJSON(checkCtx, s.localGatewayBaseURL()+"/v1/chat/completions", body, &payload); err != nil {
				return "", nil, err
			}
			return "chat completed", nil, nil
		}),
		s.runCheck(ctx, "gateway.stream", "Streaming chat", func(checkCtx context.Context) (string, map[string]any, error) {
			body := map[string]any{
				"model":      s.selfCheckModelID(ctx),
				"messages":   []map[string]any{{"role": "user", "content": "reply with ok"}},
				"stream":     true,
				"user_email": "admin-self-check@local",
			}
			reqBody, _ := json.Marshal(body)
			req, err := http.NewRequestWithContext(checkCtx, http.MethodPost, s.localGatewayBaseURL()+"/v1/chat/completions", bytes.NewReader(reqBody))
			if err != nil {
				return "", nil, err
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "text/event-stream")
			resp, err := s.selfCheckHTTPClient().Do(req)
			if err != nil {
				return "", nil, err
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 300 {
				data, _ := io.ReadAll(resp.Body)
				return "", nil, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
			}
			data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
			if err != nil {
				return "", nil, err
			}
			if !strings.Contains(string(data), "data:") {
				return "", nil, fmt.Errorf("stream response missing SSE frames")
			}
			return "stream response returned SSE frames", nil, nil
		}),
		s.runCheck(ctx, "terminal.local", "Local terminal", func(checkCtx context.Context) (string, map[string]any, error) {
			body := map[string]any{
				"command":    "Get-Location | Select-Object -ExpandProperty Path",
				"shell":      "powershell",
				"workdir":    "D:\\AI\\Local",
				"timeout_ms": 10000,
				"user_email": "admin-self-check@local",
			}
			var payload map[string]any
			if err := s.postJSON(checkCtx, strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/")+"/api/tools/terminal", body, &payload); err != nil {
				return "", nil, err
			}
			if ok, _ := payload["ok"].(bool); !ok {
				return "", payload, fmt.Errorf("%v", payload["summary"])
			}
			return "local terminal execution ok", map[string]any{"summary": payload["summary"]}, nil
		}),
		s.selfCheckSSHTerminal(ctx),
	}

	checks = append(checks, s.selfCheckMCPServers(ctx)...)

	ok := true
	for _, item := range checks {
		if item.Status == "failed" {
			ok = false
			break
		}
	}
	return &SelfCheckResponse{OK: ok, Checks: checks}, nil
}

func (s *Service) selfCheckMCPServers(ctx context.Context) []SelfCheckItem {
	servers, err := s.readMCPServers()
	if err != nil {
		return []SelfCheckItem{{
			ID:      "mcp.config",
			Label:   "MCP config",
			Status:  "failed",
			Message: err.Error(),
		}}
	}

	items := make([]SelfCheckItem, 0, len(servers))
	for _, server := range servers {
		if !server.Enabled {
			continue
		}
		serverCopy := server
		items = append(items, s.runCheck(ctx, "mcp."+serverCopy.ID, "MCP "+serverCopy.Label, func(checkCtx context.Context) (string, map[string]any, error) {
			resp, err := s.ValidateMCPServer(checkCtx, serverCopy)
			if err != nil {
				return "", nil, err
			}
			if !resp.OK {
				return "", map[string]any{"error": resp.Error}, fmt.Errorf("%s", resp.Summary)
			}
			return resp.Summary, map[string]any{
				"type": resp.EffectiveOpenWebUIType,
				"url":  resp.EffectiveConnectionURL,
			}, nil
		}))
	}

	if len(items) == 0 {
		return []SelfCheckItem{{
			ID:      "mcp.none",
			Label:   "MCP",
			Status:  "skipped",
			Message: "no enabled MCP servers configured",
		}}
	}
	return items
}

func (s *Service) selfCheckSSHTerminal(ctx context.Context) SelfCheckItem {
	settings, err := s.readUserSettings()
	if err != nil {
		return SelfCheckItem{ID: "terminal.ssh", Label: "SSH terminal", Status: "failed", Message: err.Error()}
	}
	hosts, err := s.readSSHHosts()
	if err != nil {
		return SelfCheckItem{ID: "terminal.ssh", Label: "SSH terminal", Status: "failed", Message: err.Error()}
	}

	hostByID := make(map[string]SSHHostProfile, len(hosts))
	for _, host := range hosts {
		hostByID[strings.ToLower(strings.TrimSpace(host.ID))] = host
	}

	for _, workspace := range settings {
		hostID := strings.TrimSpace(workspace.DefaultSSHHostID)
		if hostID == "" {
			continue
		}
		host, ok := hostByID[strings.ToLower(hostID)]
		if !ok || !host.Enabled {
			continue
		}

		command := "pwd"
		if host.RemoteShellDefault == "powershell" {
			command = "Get-Location | Select-Object -ExpandProperty Path"
		}
		workdir := host.DefaultWorkdir
		if strings.TrimSpace(workdir) == "" && len(host.AllowedPaths) > 0 {
			workdir = host.AllowedPaths[0]
		}

		return s.runCheck(ctx, "terminal.ssh", "SSH terminal", func(checkCtx context.Context) (string, map[string]any, error) {
			body := map[string]any{
				"transport":    "ssh",
				"host_id":      host.ID,
				"remote_shell": host.RemoteShellDefault,
				"command":      command,
				"workdir":      workdir,
				"timeout_ms":   10000,
				"user_email":   workspace.UserEmail,
			}
			var payload map[string]any
			if err := s.postJSON(checkCtx, strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/")+"/api/tools/terminal", body, &payload); err != nil {
				return "", nil, err
			}
			if ok, _ := payload["ok"].(bool); !ok {
				return "", payload, fmt.Errorf("%v", payload["summary"])
			}
			return "ssh terminal execution ok", map[string]any{
				"host_id":    host.ID,
				"user_email": workspace.UserEmail,
			}, nil
		})
	}

	return SelfCheckItem{
		ID:      "terminal.ssh",
		Label:   "SSH terminal",
		Status:  "skipped",
		Message: "no default SSH host binding available for self-check",
	}
}

func (s *Service) ensureNamedService(
	ctx context.Context,
	name string,
	check func(context.Context) error,
	start func(context.Context) error,
	timeout time.Duration,
) ServiceActionResult {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	err := check(checkCtx)
	cancel()
	if err == nil {
		return ServiceActionResult{Name: name, OK: true, Message: "already running"}
	}

	if err := start(ctx); err != nil {
		return ServiceActionResult{Name: name, OK: false, Message: err.Error()}
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err = check(checkCtx)
		cancel()
		if err == nil {
			return ServiceActionResult{Name: name, OK: true, Message: "started and healthy"}
		}
		time.Sleep(1500 * time.Millisecond)
	}

	return ServiceActionResult{Name: name, OK: false, Message: "start timed out waiting for health"}
}

func (s *Service) runCheck(
	ctx context.Context,
	id, label string,
	fn func(context.Context) (string, map[string]any, error),
) SelfCheckItem {
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	message, details, err := fn(checkCtx)
	if err != nil {
		return SelfCheckItem{
			ID:      id,
			Label:   label,
			Status:  "failed",
			Message: err.Error(),
			Details: details,
		}
	}
	return SelfCheckItem{
		ID:      id,
		Label:   label,
		Status:  "ok",
		Message: message,
		Details: details,
	}
}

func (s *Service) localGatewayBaseURL() string {
	host := strings.TrimSpace(s.cfg.ListenHost)
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, s.cfg.ListenPort)
}

func (s *Service) selfCheckModelID(ctx context.Context) string {
	models, err := s.ExposedModels()
	if err == nil && len(models) > 0 {
		return models[0].ModelID
	}
	return s.cfg.ProviderModelAlias
}

func (s *Service) simpleGET(ctx context.Context, url string) error {
	client := s.selfCheckHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return nil
}

func (s *Service) getJSON(ctx context.Context, url string, target any) error {
	client := s.selfCheckHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (s *Service) postJSON(ctx context.Context, url string, payload any, target any) error {
	client := s.selfCheckHTTPClient()
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (s *Service) selfCheckHTTPClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Minute}
}
