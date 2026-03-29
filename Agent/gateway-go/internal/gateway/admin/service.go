package admin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/config"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/provider"
)

type Service struct {
	cfg                  *config.Config
	provider             *provider.Client
	host                 *HostClient
	client               *http.Client
	toolRouterConfigPath string
	toolRouterProjectDir string
}

func NewService(cfg *config.Config, providerClient *provider.Client, gatewayConfigPath string) *Service {
	toolRouterConfigPath := strings.TrimSpace(cfg.Admin.ToolRouterConfig)
	if toolRouterConfigPath == "" && strings.TrimSpace(gatewayConfigPath) != "" {
		toolRouterConfigPath = config.ResolveSiblingPath(gatewayConfigPath, "tool-router.config.json")
	}
	toolRouterProjectDir := ""
	if strings.TrimSpace(gatewayConfigPath) != "" {
		toolRouterProjectDir = config.ResolveSiblingPath(gatewayConfigPath, "tool-router-rs")
	}
	return &Service{
		cfg:      cfg,
		provider: providerClient,
		host:     NewHostClient(cfg),
		client: &http.Client{
			Timeout: time.Duration(cfg.Admin.TimeoutMS) * time.Millisecond,
		},
		toolRouterConfigPath: toolRouterConfigPath,
		toolRouterProjectDir: toolRouterProjectDir,
	}
}

func (s *Service) Overview(ctx context.Context) (*OverviewResponse, error) {
	var (
		services []ServiceStatus
		models   *ModelsResponse
		sessions []map[string]any
		tools    []map[string]any
		failures []map[string]any
	)

	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		services, _ = s.Services(ctx)
	}()

	go func() {
		defer wg.Done()
		models, _ = s.Models(ctx)
	}()

	go func() {
		defer wg.Done()
		sessions, _ = readRecentEntries(s.cfg.Logs.SessionLogDir, 10, nil)
	}()

	go func() {
		defer wg.Done()
		tools, _ = readRecentEntries(s.cfg.Admin.ToolLogDir, 10, nil)
	}()

	go func() {
		defer wg.Done()
		failures, _ = readRecentEntries(s.cfg.Logs.AuditLogDir, 10, func(item map[string]any) bool {
			ok, exists := item["ok"].(bool)
			return exists && !ok
		})
	}()

	wg.Wait()

	resp := &OverviewResponse{
		Services:               services,
		RecentSessionSummaries: sessions,
		RecentToolSummaries:    tools,
		RecentFailures:         failures,
	}
	if models != nil {
		resp.ActiveModel = models.ActiveModel
		resp.AvailableProfiles = models.Profiles
	}
	return resp, nil
}

func (s *Service) Services(ctx context.Context) ([]ServiceStatus, error) {
	now := time.Now().Format(time.RFC3339)

	var (
		hostStatus       *hostStatusResponse
		hostErr          error
		providerStatus   ServiceStatus
		hostAgentStatus  ServiceStatus
		toolRouterStatus ServiceStatus
		openWebUIStatus  ServiceStatus
	)

	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		checkCtx, cancel := s.newCheckContext(ctx)
		defer cancel()
		hostStatus, hostErr = s.host.Status(checkCtx)
	}()

	go func() {
		defer wg.Done()
		providerStatus = s.checkStatus(ctx, "llama-server", strings.TrimRight(s.cfg.LlamaServer.BaseURL, "/"), s.provider.Health)
	}()

	go func() {
		defer wg.Done()
		hostAgentStatus = s.checkStatus(ctx, "host-agent", s.cfg.Admin.HostAgentURL, s.host.Health)
	}()

	go func() {
		defer wg.Done()
		toolRouterStatus = s.checkHTTPStatus(ctx, "tool-router", strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/")+"/healthz")
	}()

	go func() {
		defer wg.Done()
		openWebUIStatus = s.checkHTTPStatus(ctx, "open-webui", strings.TrimRight(s.cfg.Admin.OpenWebUIURL, "/")+"/health")
	}()

	wg.Wait()

	gatewayStatus := ServiceStatus{
		Name:        "gateway",
		Status:      "ok",
		Address:     fmt.Sprintf("http://%s:%d", s.cfg.ListenHost, s.cfg.ListenPort),
		LastCheckAt: now,
		Message:     "gateway is running",
		Control:     defaultControl(serviceGateway),
	}

	if hostErr == nil && hostStatus != nil && hostStatus.Running {
		providerStatus.Message = strings.TrimSpace(strings.Join([]string{
			providerStatus.Message,
			fmt.Sprintf("active profile: %s", hostStatus.ActiveProfileID),
		}, "; "))
	}

	return []ServiceStatus{
		providerStatus,
		gatewayStatus,
		toolRouterStatus,
		openWebUIStatus,
		hostAgentStatus,
	}, nil
}

func (s *Service) Models(ctx context.Context) (*ModelsResponse, error) {
	profiles, err := loadProfiles(s.cfg.Admin.ModelProfilesPath)
	if err != nil {
		return nil, err
	}

	checkCtx, cancel := s.newCheckContext(ctx)
	defer cancel()

	status, err := s.host.Status(checkCtx)
	if err != nil {
		return &ModelsResponse{Profiles: profiles}, nil
	}

	active := ActiveModel{
		ProfileID: status.ActiveProfileID,
		ModelPath: status.ModelPath,
		Running:   status.Running,
		PID:       status.PID,
		Managed:   status.Managed,
		Message:   status.Message,
	}
	for _, profile := range profiles {
		if profile.ID == status.ActiveProfileID {
			active.Label = profile.Label
			active.Quant = profile.Quant
			if active.ModelPath == "" {
				active.ModelPath = profile.ModelPath
			}
			break
		}
	}

	return &ModelsResponse{
		ActiveProfileID: status.ActiveProfileID,
		ActiveModel:     active,
		Profiles:        profiles,
	}, nil
}

func (s *Service) Modes(ctx context.Context) (*ModesResponse, error) {
	return loadModes(s.cfg)
}

func (s *Service) SessionLogs(limit int) ([]map[string]any, error) {
	return readRecentEntries(s.cfg.Logs.SessionLogDir, limit, nil)
}

func (s *Service) ToolLogs(limit int) ([]map[string]any, error) {
	return readRecentEntries(s.cfg.Admin.ToolLogDir, limit, nil)
}

func (s *Service) SwitchModel(ctx context.Context, profileID string) error {
	status, err := s.host.Status(ctx)
	if err != nil {
		return err
	}
	if status.Running {
		if err := s.host.Stop(ctx); err != nil {
			return err
		}
		if err := s.host.Switch(ctx, profileID); err != nil {
			return err
		}
		return s.host.Start(ctx)
	}
	return s.host.Switch(ctx, profileID)
}

func (s *Service) StartModel(ctx context.Context) error {
	return s.host.Start(ctx)
}

func (s *Service) StopModel(ctx context.Context) error {
	return s.host.Stop(ctx)
}

func (s *Service) RestartModel(ctx context.Context) error {
	return s.host.Restart(ctx)
}

func (s *Service) UpdateModelProfile(ctx context.Context, profile ModelProfile, applyNow bool) error {
	if strings.TrimSpace(profile.ID) == "" {
		return fmt.Errorf("profile id is required")
	}
	if strings.TrimSpace(profile.ModelPath) == "" {
		return fmt.Errorf("model path is required")
	}
	if profile.CtxSize <= 0 {
		return fmt.Errorf("ctx_size must be greater than 0")
	}
	if profile.Threads <= 0 {
		return fmt.Errorf("threads must be greater than 0")
	}
	if profile.Parallel <= 0 {
		return fmt.Errorf("parallel must be greater than 0")
	}
	if profile.NGPULayers < 0 {
		return fmt.Errorf("n_gpu_layers must be 0 or greater")
	}

	if err := s.host.UpdateProfile(ctx, profile); err != nil {
		return err
	}

	if !applyNow {
		return nil
	}

	status, err := s.host.Status(ctx)
	if err != nil {
		return err
	}
	if status.ActiveProfileID == profile.ID && status.Running {
		return s.host.Restart(ctx)
	}
	return nil
}

func (s *Service) HostIPs() (*HostIPsResponse, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("enumerate interfaces: %w", err)
	}

	var ips []string
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() || ip.To4() == nil {
			continue
		}
		ips = append(ips, ip.String())
	}

	port := s.cfg.Admin.WebUISharePort
	urls := make([]string, len(ips))
	for i, ip := range ips {
		urls[i] = fmt.Sprintf("http://%s:%d", ip, port)
	}

	return &HostIPsResponse{
		IPs:       ips,
		SharePort: port,
		ShareURLs: urls,
	}, nil
}

func (s *Service) checkStatus(ctx context.Context, name, address string, fn func(context.Context) error) ServiceStatus {
	status := ServiceStatus{
		Name:        name,
		Address:     address,
		LastCheckAt: time.Now().Format(time.RFC3339),
		Status:      "down",
		Control:     defaultControl(name),
	}
	checkCtx, cancel := s.newCheckContext(ctx)
	defer cancel()

	if err := fn(checkCtx); err != nil {
		status.Message = err.Error()
		return status
	}
	status.Status = "ok"
	status.Message = "healthy"
	return status
}

func (s *Service) checkHTTPStatus(ctx context.Context, name, address string) ServiceStatus {
	status := ServiceStatus{
		Name:        name,
		Address:     address,
		LastCheckAt: time.Now().Format(time.RFC3339),
		Status:      "down",
		Control:     defaultControl(name),
	}

	checkCtx, cancel := s.newCheckContext(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, address, nil)
	if err != nil {
		status.Message = err.Error()
		return status
	}
	resp, err := s.client.Do(req)
	if err != nil {
		status.Message = err.Error()
		return status
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		status.Message = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
		return status
	}
	status.Status = "ok"
	status.Message = "healthy"
	return status
}

func (s *Service) newCheckContext(parent context.Context) (context.Context, context.CancelFunc) {
	timeout := 1500 * time.Millisecond
	if configured := time.Duration(s.cfg.Admin.TimeoutMS) * time.Millisecond; configured > 0 && configured < timeout {
		timeout = configured
	}

	return context.WithTimeout(parent, timeout)
}
