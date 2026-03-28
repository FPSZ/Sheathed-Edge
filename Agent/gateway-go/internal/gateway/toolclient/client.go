package toolclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/config"
)

type Client struct {
	baseURL string
	client  *http.Client
}

type ResolveRequest struct {
	SessionID string         `json:"session_id"`
	Mode      string         `json:"mode"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

type ResolveResponse struct {
	Allowed             bool           `json:"allowed"`
	Tool                string         `json:"tool"`
	Reason              string         `json:"reason"`
	NormalizedArguments map[string]any `json:"normalized_arguments"`
}

type ExecuteRequest struct {
	SessionID string         `json:"session_id"`
	Mode      string         `json:"mode"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ExecuteResponse struct {
	OK        bool           `json:"ok"`
	Tool      string         `json:"tool"`
	Result    map[string]any `json:"result"`
	Summary   string         `json:"summary"`
	Truncated bool           `json:"truncated"`
	Error     *Error         `json:"error,omitempty"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL: cfg.ToolRouter.BaseURL,
		client: &http.Client{
			Timeout: time.Duration(cfg.ToolRouter.TimeoutMS) * time.Millisecond,
		},
	}
}

func (c *Client) Resolve(ctx context.Context, reqBody ResolveRequest) (*ResolveResponse, error) {
	var resp ResolveResponse
	if err := c.postJSON(ctx, "/internal/tools/resolve", reqBody, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Execute(ctx context.Context, reqBody ExecuteRequest) (*ExecuteResponse, error) {
	var resp ExecuteResponse
	if err := c.postJSON(ctx, "/internal/tools/execute", reqBody, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) postJSON(ctx context.Context, path string, request any, out any) error {
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("tool router error: %s", string(body))
	}
	return json.Unmarshal(body, out)
}
