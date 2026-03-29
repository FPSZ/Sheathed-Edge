package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	mux.HandleFunc("/internal/admin/models", s.handleAdminModels)
	mux.HandleFunc("/internal/admin/models/update", s.handleAdminModelUpdate)
	mux.HandleFunc("/internal/admin/modes", s.handleAdminModes)
	mux.HandleFunc("/internal/admin/logs/sessions", s.handleAdminSessionLogs)
	mux.HandleFunc("/internal/admin/logs/tools", s.handleAdminToolLogs)
	mux.HandleFunc("/internal/admin/settings/terminal-paths", s.handleAdminTerminalPaths)
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
			upstreamReq, err := s.orchestrator.PrepareNativeStreamingRequest(req, selectedModel.ModelID)
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
			if err := s.provider.StreamChatCompletion(r.Context(), upstreamReq, selectedModel.ModelID, w, flusher.Flush); err != nil {
				finalSpan.End(false, err.Error())
				return
			}
			finalSpan.End(true, "")
			return
		}

		resp, _, _, err := s.orchestrator.RunNativeToolTurn(r.Context(), requestID, selectedModel.ModelID, req, trace)
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
		finalSpan.End(true, "")
		writeJSON(w, http.StatusOK, resp)
		return
	}

	if req.Stream {
		streamReq, ok, err := s.orchestrator.PrepareStreamingRequest(req, selectedModel.ModelID)
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
			if err := s.provider.StreamChatCompletion(r.Context(), streamReq, selectedModel.ModelID, w, flusher.Flush); err != nil {
				finalSpan.End(false, err.Error())
				return
			}
			finalSpan.End(true, "")
			return
		}
	}

	resp, _, _, err := s.orchestrator.RunTurn(r.Context(), requestID, selectedModel.ModelID, req, trace)
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
	if req.Stream {
		finalSpan.End(true, "")
		writeSSEChatCompletion(w, selectedModel.ModelID, envelope.FirstContent(resp))
		return
	}
	finalSpan.End(true, "")
	writeJSON(w, http.StatusOK, resp)
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
