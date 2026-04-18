package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/admin"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/config"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/envelope"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/logging"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/mode"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/orchestrator"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/provider"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/retrieval"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/toolclient"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/types"
)

type Server struct {
	cfg          *config.Config
	provider     *provider.Client
	orchestrator *orchestrator.Orchestrator
	stageLogger  *logging.StageLogger
	admin        *admin.Service
	httpServer   *http.Server
}

func NewServer(configPath string) (*Server, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	providerClient := provider.NewClient(cfg)
	sessionLogger := logging.NewSessionLogger(cfg.Logs.SessionLogDir)
	stageLogger := logging.NewStageLogger(cfg.Logs.AuditLogDir)
	orch := orchestrator.New(
		mode.NewLoader(cfg),
		retrieval.NewService(cfg),
		providerClient,
		toolclient.NewClient(cfg),
		sessionLogger,
	)

	s := &Server{
		cfg:          cfg,
		provider:     providerClient,
		orchestrator: orch,
		stageLogger:  stageLogger,
		admin:        admin.NewService(cfg, providerClient, configPath),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("/internal/admin/overview", s.handleAdminOverview)
	mux.HandleFunc("/internal/admin/services", s.handleAdminServices)
	mux.HandleFunc("/internal/admin/services/start", s.handleAdminServiceStart)
	mux.HandleFunc("/internal/admin/services/stop", s.handleAdminServiceStop)
	mux.HandleFunc("/internal/admin/start-all", s.handleAdminStartAll)
	mux.HandleFunc("/internal/admin/self-check", s.handleAdminSelfCheck)
	mux.HandleFunc("/internal/admin/models", s.handleAdminModels)
	mux.HandleFunc("/internal/admin/models/update", s.handleAdminModelUpdate)
	mux.HandleFunc("/internal/admin/modes", s.handleAdminModes)
	mux.HandleFunc("/internal/admin/agent-layers", s.handleAdminAgentLayers)
	mux.HandleFunc("/internal/admin/users", s.handleAdminUsers)
	mux.HandleFunc("/internal/admin/users/workspace", s.handleAdminUserWorkspace)
	mux.HandleFunc("/internal/admin/logs/sessions", s.handleAdminSessionLogs)
	mux.HandleFunc("/internal/admin/logs/tools", s.handleAdminToolLogs)
	mux.HandleFunc("/internal/admin/settings/terminal-paths", s.handleAdminTerminalPaths)
	mux.HandleFunc("/internal/admin/ssh/hosts", s.handleAdminSSHHosts)
	mux.HandleFunc("/internal/admin/ssh/hosts/test", s.handleAdminSSHHostsTest)
	mux.HandleFunc("/internal/admin/ssh/hosts/confirm-host-key", s.handleAdminSSHHostsConfirmHostKey)
	mux.HandleFunc("/internal/admin/ssh/runtime", s.handleAdminSSHRuntime)
	mux.HandleFunc("/internal/admin/ssh/bindings", s.handleAdminSSHBindings)
	mux.HandleFunc("/internal/admin/mcp/servers", s.handleAdminMCPServers)
	mux.HandleFunc("/internal/admin/mcp/servers/validate", s.handleAdminMCPValidate)
	mux.HandleFunc("/internal/admin/mcp/servers/discover-tools", s.handleAdminMCPDiscoverTools)
	mux.HandleFunc("/internal/admin/mcp/runtime", s.handleAdminMCPRuntime)
	mux.HandleFunc("/internal/admin/mcp/openwebui-preview", s.handleAdminMCPOpenWebUIPreview)
	mux.HandleFunc("/internal/admin/models/switch", s.handleAdminModelSwitch)
	mux.HandleFunc("/internal/admin/llama/start", s.handleAdminLlamaStart)
	mux.HandleFunc("/internal/admin/llama/stop", s.handleAdminLlamaStop)
	mux.HandleFunc("/internal/admin/llama/restart", s.handleAdminLlamaRestart)
	mux.HandleFunc("/internal/admin/host-ips", s.handleAdminHostIPs)
	mux.HandleFunc("/admin", s.handleAdminUI)
	mux.HandleFunc("/admin/", s.handleAdminUI)

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.ListenHost, cfg.ListenPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s, nil
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	status := map[string]any{
		"status":   "ok",
		"provider": "down",
	}
	if err := s.provider.Health(ctx); err == nil {
		status["provider"] = "ok"
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.admin.ExposedModels()
	if err != nil {
		writeError(w, http.StatusBadGateway, "admin_error", err.Error())
		return
	}

	data := make([]map[string]any, 0, len(models))
	for _, model := range models {
		data = append(data, map[string]any{
			"id":       model.ModelID,
			"object":   "model",
			"created":  time.Now().Unix(),
			"owned_by": "local",
		})
	}
	if len(data) == 0 {
		data = append(data, map[string]any{
			"id":       s.cfg.ProviderModelAlias,
			"object":   "model",
			"created":  time.Now().Unix(),
			"owned_by": "local",
		})
	}

	resp := map[string]any{
		"object": "list",
		"data":   data,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	requestID := newRequestID()
	trace := s.stageLogger.NewTrace(requestID)
	finalSpan := trace.Begin("final_writeback")
	defer func() {
		if recovered := recover(); recovered != nil {
			finalSpan.End(false, fmt.Sprintf("panic: %v", recovered))
			writeErrorWithRequestID(w, http.StatusBadGateway, "provider_error", fmt.Sprintf("gateway panic: %v", recovered), requestID)
		}
	}()

	var req types.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		finalSpan.End(false, err.Error())
		writeErrorWithRequestID(w, http.StatusBadRequest, "invalid_request", err.Error(), requestID)
		return
	}
	req.UserEmail = firstNonEmptyUserEmail(
		req.UserEmail,
		r.Header.Get("X-AWDP-User-Email"),
		r.Header.Get("X-User-Email"),
	)
	sanitizeOpenWebUIToolSelectorRequest(&req)
	applyDefaultMaxTokens(&req)
	s.applyDefaultAgentPreset(&req)
	applyDefaultNativeToolFallback(&req)
	trace.Begin("request_received").End(true, summarizeChatRequest(req))

	selectedModel, err := s.admin.EnsureModelReady(r.Context(), req.Model)
	if err != nil {
		finalSpan.End(false, err.Error())
		writeErrorWithRequestID(w, http.StatusBadGateway, "model_switch_failed", err.Error(), requestID)
		return
	}
	req.Model = selectedModel.ModelID

	if req.UsesNativeTools() {
		if req.Stream {
			upstreamReq, active, err := s.orchestrator.PrepareNativeStreamingRequest(req, selectedModel.ModelID)
			if err != nil {
				finalSpan.End(false, err.Error())
				writeErrorWithRequestID(w, http.StatusBadGateway, "provider_error", err.Error(), requestID)
				return
			}

			flusher, ok := w.(http.Flusher)
			if !ok {
				finalSpan.End(false, "response writer does not support streaming")
				writeErrorWithRequestID(w, http.StatusInternalServerError, "stream_unsupported", "response writer does not support streaming", requestID)
				return
			}
			writeSSEHeaders(w)
			writeSSEStatusPreamble(w, flusher, selectedModel.ModelID, buildSkillReadingStatus(active))
			if err := s.provider.StreamChatCompletion(r.Context(), upstreamReq, selectedModel.ModelID, w, flusher.Flush); err != nil {
				finalSpan.End(false, err.Error())
				return
			}
			finalSpan.End(true, "")
			return
		}

		resp, active, _, err := s.orchestrator.RunNativeToolTurn(r.Context(), requestID, selectedModel.ModelID, req, trace)
		if err != nil {
			finalSpan.End(false, err.Error())
			writeErrorWithRequestID(w, http.StatusBadGateway, "provider_error", err.Error(), requestID)
			return
		}
		if err := validateChatResponse(resp); err != nil {
			finalSpan.End(false, err.Error())
			writeErrorWithRequestID(w, http.StatusBadGateway, "provider_error", err.Error(), requestID)
			return
		}

		resp.Model = selectedModel.ModelID
		prependSkillReadingStatus(resp, active)
		finalSpan.End(true, "")
		writeJSON(w, http.StatusOK, resp)
		return
	}

	if req.Stream {
		streamReq, active, ok, err := s.orchestrator.PrepareStreamingRequest(req, selectedModel.ModelID)
		if err != nil {
			finalSpan.End(false, err.Error())
			writeErrorWithRequestID(w, http.StatusBadGateway, "provider_error", err.Error(), requestID)
			return
		}
		if ok {
			flusher, ok := w.(http.Flusher)
			if !ok {
				finalSpan.End(false, "response writer does not support streaming")
				writeErrorWithRequestID(w, http.StatusInternalServerError, "stream_unsupported", "response writer does not support streaming", requestID)
				return
			}
			writeSSEHeaders(w)
			writeSSEStatusPreamble(w, flusher, selectedModel.ModelID, buildSkillReadingStatus(active))
			if err := s.provider.StreamChatCompletion(r.Context(), streamReq, selectedModel.ModelID, w, flusher.Flush); err != nil {
				finalSpan.End(false, err.Error())
				return
			}
			finalSpan.End(true, "")
			return
		}
	}

	resp, active, _, err := s.orchestrator.RunTurn(r.Context(), requestID, selectedModel.ModelID, req, trace)
	if err != nil {
		finalSpan.End(false, err.Error())
		writeErrorWithRequestID(w, http.StatusBadGateway, "provider_error", err.Error(), requestID)
		return
	}
	if err := validateChatResponse(resp); err != nil {
		finalSpan.End(false, err.Error())
		writeErrorWithRequestID(w, http.StatusBadGateway, "provider_error", err.Error(), requestID)
		return
	}

	resp.Model = selectedModel.ModelID
	prependSkillReadingStatus(resp, active)
	if req.Stream {
		finalSpan.End(true, "")
		writeSSEChatCompletion(w, selectedModel.ModelID, envelope.FirstContent(resp))
		return
	}
	finalSpan.End(true, "")
	writeJSON(w, http.StatusOK, resp)
}

func hasToolResultMessages(messages []types.ChatMessage) bool {
	for _, message := range messages {
		if strings.EqualFold(strings.TrimSpace(message.Role), "tool") {
			return true
		}
	}
	return false
}

func applyDefaultMaxTokens(req *types.ChatCompletionRequest) {
	if req == nil || req.MaxTokens != nil {
		return
	}

	maxTokens := 4096
	if req.UsesNativeTools() {
		maxTokens = 3072
	}
	req.MaxTokens = &maxTokens
}

func applyDefaultNativeToolFallback(req *types.ChatCompletionRequest) {
	if req == nil || req.UsesNativeTools() {
		return
	}
	if !hasTechnicalPlugin(req.XPlugins) {
		return
	}
	query := strings.ToLower(strings.TrimSpace(latestUserContent(req.Messages)))
	if query == "" || !looksLikeTechnicalTask(query) {
		return
	}

	req.Tools = []types.ToolSpec{
		{
			Type: "function",
			Function: types.FunctionSpec{
				Name:        "runTerminal",
				Description: "Choose transport per call. If transport is omitted, the server resolves the current user's default execution target automatically. Use local for host scripts, repo operations, binary triage, packaging, and file transfer orchestration. Use ssh with host_id for remote directories, logs, processes, and running tasks on that host.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"command": map[string]any{
							"type":        "string",
							"minLength":   1,
							"description": "Shell command to run on the selected execution target.",
						},
						"transport": map[string]any{
							"type":        "string",
							"enum":        []string{"local", "ssh"},
							"description": "Execution target kind. If omitted, the server selects the current user's preferred target automatically.",
						},
						"host_id": map[string]any{
							"type":        "string",
							"minLength":   1,
							"description": "Required when transport=ssh unless the current user has a default SSH host binding.",
						},
						"shell": map[string]any{
							"type":        "string",
							"enum":        []string{"powershell", "wsl-bash"},
							"description": "Local shell. Only used when transport resolves to local.",
						},
						"remote_shell": map[string]any{
							"type":        "string",
							"enum":        []string{"bash", "powershell"},
							"description": "Remote shell to launch on the SSH host. Only used when transport=ssh.",
						},
						"workdir": map[string]any{
							"type":        "string",
							"minLength":   1,
							"description": "Working directory on the selected target. Keep it inside the target allowed paths.",
						},
						"timeout_ms": map[string]any{
							"type":        "integer",
							"minimum":     1,
							"description": "Overall execution timeout in milliseconds.",
						},
						"user_email": map[string]any{
							"type":        "string",
							"minLength":   3,
							"description": "Current Open WebUI user email. Used to enforce per-user execution target authorization.",
						},
					},
					"required":             []string{"command"},
					"additionalProperties": false,
				},
			},
		},
	}
	req.ToolChoice = "required"
	parallel := false
	req.ParallelToolCalls = &parallel
}

func sanitizeOpenWebUIToolSelectorRequest(req *types.ChatCompletionRequest) {
	if req == nil || len(req.Messages) == 0 {
		return
	}

	foundSyntheticSelector := false
	filtered := make([]types.ChatMessage, 0, len(req.Messages))
	for _, message := range req.Messages {
		if isSyntheticOpenWebUIToolSelectorMessage(message) {
			foundSyntheticSelector = true
			continue
		}
		filtered = append(filtered, message)
	}

	if !foundSyntheticSelector {
		return
	}

	for index := range filtered {
		if !strings.EqualFold(strings.TrimSpace(filtered[index].Role), "user") {
			continue
		}
		content := strings.TrimSpace(filtered[index].Content)
		if len(content) < len("Query:") || !strings.EqualFold(content[:len("Query:")], "Query:") {
			continue
		}
		filtered[index].Content = strings.TrimSpace(content[len("Query:"):])
	}

	req.Messages = filtered
	if len(req.Tools) > 0 {
		req.ToolChoice = nil
		req.ParallelToolCalls = nil
	}
}

func isSyntheticOpenWebUIToolSelectorMessage(message types.ChatMessage) bool {
	if !strings.EqualFold(strings.TrimSpace(message.Role), "system") {
		return false
	}

	content := strings.TrimSpace(message.Content)
	if content == "" {
		return false
	}

	if !strings.Contains(content, "Available Tools:") {
		return false
	}

	markers := []string{
		"Return only the JSON object",
		"tool_calls",
		"If no tools match the query",
		"The format for the JSON response is strictly",
	}
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			return false
		}
	}
	return true
}

func hasTechnicalPlugin(plugins []string) bool {
	for _, plugin := range plugins {
		switch strings.ToLower(strings.TrimSpace(plugin)) {
		case "reverse", "pwn", "web", "awdp-red", "awdp-blue":
			return true
		}
	}
	return false
}

func latestUserContent(messages []types.ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(strings.TrimSpace(messages[i].Role), "user") {
			return messages[i].Content
		}
	}
	return ""
}

func looksLikeTechnicalTask(query string) bool {
	if strings.Contains(query, "\\") || strings.Contains(query, "/") {
		return true
	}
	for _, marker := range []string{
		".exe", ".dll", ".so", ".py", ".php", ".js", ".ts", ".go",
		"ctf", "awdp", "pwn", "reverse", "web", "mcp", "ssh",
		"解题", "分析", "逆向", "漏洞", "修复", "补丁", "目录", "文件", "看看", "运行", "读取",
	} {
		if strings.Contains(query, marker) {
			return true
		}
	}
	return false
}

func prependSkillReadingStatus(resp *types.ChatCompletionResponse, active *mode.Active) {
	if resp == nil || len(resp.Choices) == 0 {
		return
	}
	status := buildSkillReadingStatus(active)
	if status == "" {
		return
	}
	message := resp.Choices[0].Message
	content := strings.TrimSpace(message.Content)
	if content == "" {
		resp.Choices[0].Message.Content = status
		return
	}
	resp.Choices[0].Message.Content = status + "\n\n" + content
}

func buildSkillReadingStatus(active *mode.Active) string {
	if active == nil {
		return ""
	}
	labels := make([]string, 0, len(active.PromptFiles)+len(active.SkillFiles))
	seen := make(map[string]struct{}, len(active.PromptFiles)+len(active.SkillFiles))
	appendLabel := func(path string) {
		base := strings.TrimSpace(filepath.Base(path))
		if base == "" {
			return
		}
		if _, ok := seen[base]; ok {
			return
		}
		seen[base] = struct{}{}
		labels = append(labels, base)
	}
	for _, path := range active.PromptFiles {
		base := strings.ToLower(filepath.Base(path))
		if base == "agent.md" || base == "binary-core.md" || base == "awdp-core.md" {
			appendLabel(path)
		}
	}
	for _, path := range active.SkillFiles {
		appendLabel(path)
	}
	if len(labels) == 0 {
		return ""
	}
	return "正在阅读：`" + strings.Join(labels, "`、`") + "`"
}

func writeSSEStatusPreamble(w io.Writer, flusher http.Flusher, model string, status string) {
	status = strings.TrimSpace(status)
	if status == "" {
		return
	}
	id := fmt.Sprintf("chatcmpl-skill-%d", time.Now().UnixNano())
	created := time.Now().Unix()
	event := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]any{
			{"index": 0, "delta": map[string]any{"content": status + "\n\n"}, "finish_reason": nil},
		},
	}
	data, _ := json.Marshal(event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func (s *Server) applyDefaultAgentPreset(req *types.ChatCompletionRequest) {
	if req == nil {
		return
	}
	if hasExplicitAgentPreset(*req) {
		return
	}
	if s.admin == nil {
		return
	}
	preset, err := s.admin.DefaultAgentLayerPreset()
	if err != nil || preset == nil {
		return
	}

	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	req.Metadata["agent_layers"] = map[string]any{
		"enable_agent_router":   preset.EnableAgentRouter,
		"enable_reverse_skills": preset.EnableReverseSkills,
		"enable_pwn_skills":     preset.EnablePwnSkills,
		"enable_web_skills":     preset.EnableWebSkills,
		"enable_awdp_red":       preset.EnableAWDPRed,
		"enable_awdp_blue":      preset.EnableAWDPBlue,
	}

	if len(req.XPlugins) == 0 {
		var plugins []string
		if preset.EnableReverseSkills {
			plugins = append(plugins, "reverse")
		}
		if preset.EnablePwnSkills {
			plugins = append(plugins, "pwn")
		}
		if preset.EnableWebSkills {
			plugins = append(plugins, "web")
		}
		if preset.EnableAWDPRed {
			plugins = append(plugins, "awdp-red")
		}
		if preset.EnableAWDPBlue {
			plugins = append(plugins, "awdp-blue")
		}
		req.XPlugins = plugins
	}
}

func hasExplicitAgentPreset(req types.ChatCompletionRequest) bool {
	if len(req.XPlugins) > 0 {
		return true
	}
	if req.Metadata == nil {
		return false
	}
	for _, key := range []string{"agent_layers", "preset_id"} {
		if value, ok := req.Metadata[key]; ok && value != nil {
			return true
		}
	}
	if value, ok := req.Metadata["plugins"]; ok && value != nil {
		if items, ok := value.([]any); ok && len(items) > 0 {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeErrorWithRequestID(w, status, code, message, "")
}

func writeErrorWithRequestID(w http.ResponseWriter, status int, code, message, requestID string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":       code,
			"message":    message,
			"request_id": requestID,
		},
	})
}

func writeSSEChatCompletion(w http.ResponseWriter, model, content string) {
	writeSSEHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unsupported", "response writer does not support streaming")
		return
	}

	id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()

	events := []map[string]any{
		{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []map[string]any{
				{"index": 0, "delta": map[string]any{"role": "assistant"}, "finish_reason": nil},
			},
		},
		{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []map[string]any{
				{"index": 0, "delta": map[string]any{"content": content}, "finish_reason": nil},
			},
		},
		{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []map[string]any{
				{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"},
			},
		},
	}

	for _, event := range events {
		data, _ := json.Marshal(event)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func writeSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

func summarizeChatRequest(req types.ChatCompletionRequest) string {
	toolNames := make([]string, 0, len(req.Tools))
	for _, tool := range req.Tools {
		name := strings.TrimSpace(tool.Function.Name)
		if name != "" {
			toolNames = append(toolNames, name)
		}
	}
	sort.Strings(toolNames)

	toolChoice := "unset"
	switch value := req.ToolChoice.(type) {
	case nil:
	case string:
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			toolChoice = fmt.Sprintf("string:%s", trimmed)
		}
	case map[string]any:
		if len(value) == 0 {
			toolChoice = "object:{}"
			break
		}
		if raw, err := json.Marshal(value); err == nil {
			toolChoice = "object:" + string(raw)
		} else {
			toolChoice = "object"
		}
	default:
		if raw, err := json.Marshal(value); err == nil {
			toolChoice = fmt.Sprintf("%T:%s", value, string(raw))
		} else {
			toolChoice = fmt.Sprintf("%T", value)
		}
	}

	parallel := "unset"
	if req.ParallelToolCalls != nil {
		parallel = fmt.Sprintf("%t", *req.ParallelToolCalls)
	}

	return fmt.Sprintf(
		"model=%s stream=%t messages=%d native_tools=%t tools=%d tool_names=%s tool_choice=%s parallel_tool_calls=%s plugins=%s",
		req.Model,
		req.Stream,
		len(req.Messages),
		req.UsesNativeTools(),
		len(req.Tools),
		strings.Join(toolNames, ","),
		toolChoice,
		parallel,
		strings.Join(req.XPlugins, ","),
	)
}

func firstNonEmptyUserEmail(values ...string) string {
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
