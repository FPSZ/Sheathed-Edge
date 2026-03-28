package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/config"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/types"
)

type Client struct {
	configuredBaseURL string
	client            *http.Client
	mu                sync.RWMutex
	resolvedBaseURL   string
	upstreamModelID   string
}

func NewClient(cfg *config.Config) *Client {
	base := strings.TrimRight(cfg.LlamaServer.BaseURL, "/")
	return &Client{
		configuredBaseURL: base,
		client: &http.Client{
			Timeout: time.Duration(cfg.LlamaServer.TimeoutMS) * time.Millisecond,
		},
	}
}

func (p *Client) Health(ctx context.Context) error {
	base, err := p.resolveBaseURL(ctx)
	if err != nil {
		return err
	}
	return p.checkHealth(ctx, base)
}

func (p *Client) ChatCompletion(ctx context.Context, reqBody types.ChatCompletionRequest) (*types.ChatCompletionResponse, error) {
	reqBody.Stream = false

	base, err := p.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	if modelID, err := p.resolveUpstreamModel(ctx, base); err == nil && modelID != "" {
		reqBody.Model = modelID
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider chat failed: %s", strings.TrimSpace(string(body)))
	}

	var parsed types.ChatCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode provider response: %w", err)
	}
	return &parsed, nil
}

func (p *Client) resolveBaseURL(ctx context.Context) (string, error) {
	p.mu.RLock()
	cached := p.resolvedBaseURL
	p.mu.RUnlock()
	if cached != "" {
		if err := p.checkHealth(ctx, cached); err == nil {
			return cached, nil
		}
	}

	var lastErr error
	for _, candidate := range p.baseURLCandidates() {
		if err := p.checkHealth(ctx, candidate); err != nil {
			lastErr = err
			continue
		}
		p.mu.Lock()
		p.resolvedBaseURL = candidate
		p.mu.Unlock()
		return candidate, nil
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("no reachable llama-server base url")
}

func (p *Client) checkHealth(ctx context.Context, baseURL string) error {
	healthBase := strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/v1")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthBase+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("provider health failed: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

func (p *Client) resolveUpstreamModel(ctx context.Context, baseURL string) (string, error) {
	p.mu.RLock()
	cached := p.upstreamModelID
	p.mu.RUnlock()
	if cached != "" {
		return cached, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return "", err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("provider models failed: %s", strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Data) == 0 || parsed.Data[0].ID == "" {
		return "", fmt.Errorf("provider returned no model ids")
	}

	p.mu.Lock()
	p.upstreamModelID = parsed.Data[0].ID
	p.mu.Unlock()
	return parsed.Data[0].ID, nil
}

func (p *Client) baseURLCandidates() []string {
	base := strings.TrimRight(p.configuredBaseURL, "/")
	candidates := []string{base}

	parsed, err := url.Parse(base)
	if err != nil {
		return dedupeStrings(candidates)
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "localhost" {
		return dedupeStrings(candidates)
	}

	hostIP := detectWSLHostIP()
	if hostIP == "" {
		return dedupeStrings(candidates)
	}

	port := parsed.Port()
	parsed.Host = hostIP
	if port != "" {
		parsed.Host = hostIP + ":" + port
	}
	candidates = append(candidates, strings.TrimRight(parsed.String(), "/"))
	return dedupeStrings(candidates)
}

func detectWSLHostIP() string {
	if out, err := exec.Command("bash", "-lc", "ip route show default 2>/dev/null | awk '/default/ {print $3; exit}'").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
