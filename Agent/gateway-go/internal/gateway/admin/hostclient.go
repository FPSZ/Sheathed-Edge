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

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/config"
)

type HostClient struct {
	baseURL string
	client  *http.Client
}

type hostStatusResponse struct {
	Running         bool   `json:"running"`
	Managed         bool   `json:"managed"`
	PID             int    `json:"pid,omitempty"`
	ActiveProfileID string `json:"active_profile_id,omitempty"`
	Message         string `json:"message,omitempty"`
	ModelPath       string `json:"model_path,omitempty"`
}

func NewHostClient(cfg *config.Config) *HostClient {
	return &HostClient{
		baseURL: strings.TrimRight(cfg.Admin.HostAgentURL, "/"),
		client: &http.Client{
			Timeout: time.Duration(cfg.Admin.TimeoutMS) * time.Millisecond,
		},
	}
}

func (c *HostClient) Health(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodGet, "/healthz", nil)
	return err
}

func (c *HostClient) Status(ctx context.Context) (*hostStatusResponse, error) {
	body, err := c.do(ctx, http.MethodGet, "/internal/host/llama/status", nil)
	if err != nil {
		return nil, err
	}
	var status hostStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (c *HostClient) Start(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodPost, "/internal/host/llama/start", map[string]any{})
	return err
}

func (c *HostClient) Stop(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodPost, "/internal/host/llama/stop", map[string]any{})
	return err
}

func (c *HostClient) Restart(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodPost, "/internal/host/llama/restart", map[string]any{})
	return err
}

func (c *HostClient) Switch(ctx context.Context, profileID string) error {
	_, err := c.do(ctx, http.MethodPost, "/internal/host/llama/switch", map[string]any{
		"profile_id": profileID,
	})
	return err
}

func (c *HostClient) do(ctx context.Context, method, path string, payload any) ([]byte, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("host agent is not configured")
	}

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("host agent %s %s failed: %s", method, path, strings.TrimSpace(string(data)))
	}
	return data, nil
}
