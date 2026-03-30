package provider

import (
	"bufio"
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
	normalizeResponseMetrics(&parsed)
	return &parsed, nil
}

func (p *Client) StreamChatCompletion(ctx context.Context, reqBody types.ChatCompletionRequest, alias string, w io.Writer, flush func()) error {
	reqBody.Stream = true
	reqBody.StreamOptions = ensureIncludeUsage(reqBody.StreamOptions)

	base, err := p.resolveBaseURL(ctx)
	if err != nil {
		return err
	}
	if modelID, err := p.resolveUpstreamModel(ctx, base); err == nil && modelID != "" {
		reqBody.Model = modelID
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("provider stream failed: %s", strings.TrimSpace(string(body)))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			rewritten := rewriteStreamLine(line, alias)
			if _, writeErr := io.WriteString(w, rewritten); writeErr != nil {
				return writeErr
			}
			flush()
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
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

	base := strings.TrimRight(p.configuredBaseURL, "/")
	if err := p.checkHealth(ctx, base); err == nil {
		p.mu.Lock()
		p.resolvedBaseURL = base
		p.mu.Unlock()
		return base, nil
	} else {
		lastErr := err
		for _, candidate := range p.wslFallbackCandidates(ctx) {
			if err := p.checkHealth(ctx, candidate); err != nil {
				lastErr = err
				continue
			}
			p.mu.Lock()
			p.resolvedBaseURL = candidate
			p.mu.Unlock()
			return candidate, nil
		}
		return "", lastErr
	}
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

func (p *Client) wslFallbackCandidates(ctx context.Context) []string {
	base := strings.TrimRight(p.configuredBaseURL, "/")
	parsed, err := url.Parse(base)
	if err != nil {
		return nil
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "localhost" {
		return nil
	}

	hostIP := detectWSLHostIP(ctx)
	if hostIP == "" {
		return nil
	}

	port := parsed.Port()
	parsed.Host = hostIP
	if port != "" {
		parsed.Host = hostIP + ":" + port
	}
	return dedupeStrings([]string{strings.TrimRight(parsed.String(), "/")})
}

func detectWSLHostIP(parent context.Context) string {
	ctx, cancel := context.WithTimeout(parent, 500*time.Millisecond)
	defer cancel()

	if out, err := exec.CommandContext(ctx, "bash", "-lc", "ip route show default 2>/dev/null | awk '/default/ {print $3; exit}'").Output(); err == nil {
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

func ensureIncludeUsage(options map[string]any) map[string]any {
	if options == nil {
		return map[string]any{"include_usage": true}
	}

	cloned := make(map[string]any, len(options)+1)
	for k, v := range options {
		cloned[k] = v
	}
	if _, ok := cloned["include_usage"]; !ok {
		cloned["include_usage"] = true
	}
	return cloned
}

func rewriteStreamLine(line, alias string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "data: ") {
		return line
	}
	payload := strings.TrimPrefix(trimmed, "data: ")
	if payload == "[DONE]" {
		return line
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return line
	}
	if strings.TrimSpace(alias) != "" {
		data["model"] = alias
	}
	normalizeStreamMetrics(data)
	rewritten, err := json.Marshal(data)
	if err != nil {
		return line
	}
	return "data: " + string(rewritten) + "\n\n"
}

func normalizeResponseMetrics(resp *types.ChatCompletionResponse) {
	if resp == nil {
		return
	}
	resp.Usage = normalizeUsage(resp.Usage, resp.Timings)
}

func normalizeStreamMetrics(data map[string]any) {
	usage, _ := data["usage"].(map[string]any)
	timings, _ := data["timings"].(map[string]any)
	if usage == nil && timings == nil {
		return
	}
	data["usage"] = normalizeUsage(usage, timings)
}

func normalizeUsage(usage map[string]any, timings map[string]any) map[string]any {
	if usage == nil {
		usage = map[string]any{}
	} else {
		usage = cloneMap(usage)
	}

	inputTokens := intFromAny(usage["input_tokens"])
	if inputTokens == 0 {
		inputTokens = intFromAny(usage["prompt_tokens"])
	}
	outputTokens := intFromAny(usage["output_tokens"])
	if outputTokens == 0 {
		outputTokens = intFromAny(usage["completion_tokens"])
	}
	totalTokens := intFromAny(usage["total_tokens"])
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}

	usage["input_tokens"] = inputTokens
	usage["output_tokens"] = outputTokens
	usage["total_tokens"] = totalTokens

	if _, ok := usage["prompt_tokens"]; !ok && inputTokens > 0 {
		usage["prompt_tokens"] = inputTokens
	}
	if _, ok := usage["completion_tokens"]; !ok && outputTokens > 0 {
		usage["completion_tokens"] = outputTokens
	}

	if timings != nil {
		if _, ok := usage["response_token/s"]; !ok {
			if v, ok := floatFromAny(timings["predicted_per_second"]); ok {
				usage["response_token/s"] = roundTo(v, 2)
			}
		}
		if _, ok := usage["prompt_token/s"]; !ok {
			if v, ok := floatFromAny(timings["prompt_per_second"]); ok {
				usage["prompt_token/s"] = roundTo(v, 2)
			}
		}
		if _, ok := usage["approximate_total"]; !ok {
			totalMS := intFromAny(timings["prompt_ms"]) + intFromAny(timings["predicted_ms"])
			if totalMS > 0 {
				usage["approximate_total"] = formatDurationMS(totalMS)
			}
		}
	}

	if _, ok := usage["completion_tokens_details"]; !ok {
		usage["completion_tokens_details"] = map[string]any{
			"reasoning_tokens":           0,
			"accepted_prediction_tokens": 0,
			"rejected_prediction_tokens": 0,
		}
	}

	return usage
}

func cloneMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, err := v.Int64()
		if err == nil {
			return int(i)
		}
		f, err := v.Float64()
		if err == nil {
			return int(f)
		}
	case string:
		if strings.TrimSpace(v) == "" {
			return 0
		}
		var num json.Number = json.Number(v)
		i, err := num.Int64()
		if err == nil {
			return int(i)
		}
		f, err := num.Float64()
		if err == nil {
			return int(f)
		}
	}
	return 0
}

func floatFromAny(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, false
		}
		var num json.Number = json.Number(v)
		f, err := num.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func roundTo(value float64, digits int) float64 {
	pow := 1.0
	for i := 0; i < digits; i++ {
		pow *= 10
	}
	return float64(int(value*pow+0.5)) / pow
}

func formatDurationMS(totalMS int) string {
	if totalMS <= 0 {
		return "0s"
	}
	totalSeconds := totalMS / 1000
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
