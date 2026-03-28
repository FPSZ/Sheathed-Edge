package admin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/config"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/provider"
)

type Service struct {
	cfg      *config.Config
	provider *provider.Client
	host     *HostClient
	client   *http.Client
}

func NewService(cfg *config.Config, providerClient *provider.Client) *Service {
	return &Service{
		cfg:      cfg,
		provider: providerClient,
		host:     NewHostClient(cfg),
		client: &http.Client{
			Timeout: time.Duration(cfg.Admin.TimeoutMS) * time.Millisecond,
		},
	}
}

func (s *Service) Overview(ctx context.Context) (*OverviewResponse, error) {
	services, _ := s.Services(ctx)
	models, _ := s.Models(ctx)
	sessions, _ := readRecentEntries(s.cfg.Logs.SessionLogDir, 10, nil)
	tools, _ := readRecentEntries(s.cfg.Admin.ToolLogDir, 10, nil)
	failures, _ := readRecentEntries(s.cfg.Logs.AuditLogDir, 10, func(item map[string]any) bool {
		ok, exists := item["ok"].(bool)
		return exists && !ok
	})

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

	hostStatus, hostErr := s.host.Status(ctx)
	providerStatus := s.checkStatus(ctx, "llama-server", strings.TrimRight(s.cfg.LlamaServer.BaseURL, "/"), s.provider.Health)
	hostAgentStatus := s.checkStatus(ctx, "host-agent", s.cfg.Admin.HostAgentURL, s.host.Health)
	toolRouterStatus := s.checkHTTPStatus(ctx, "tool-router", strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/")+"/healthz")
	openWebUIStatus := s.checkHTTPStatus(ctx, "open-webui", strings.TrimRight(s.cfg.Admin.OpenWebUIURL, "/")+"/health")

	gatewayStatus := ServiceStatus{
		Name:        "gateway",
		Status:      "ok",
		Address:     fmt.Sprintf("http://%s:%d", s.cfg.ListenHost, s.cfg.ListenPort),
		LastCheckAt: now,
		Message:     "gateway is running",
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

	status, err := s.host.Status(ctx)
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
	}
	if err := fn(ctx); err != nil {
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
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
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
