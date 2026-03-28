package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
	httpServer   *http.Server
}

func NewServer(configPath string) (*Server, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	providerClient := provider.NewClient(cfg)
	sessionLogger := logging.NewSessionLogger(cfg.Logs.SessionLogDir)
	orch := orchestrator.New(
		mode.NewLoader(cfg),
		retrieval.NewService(cfg),
		providerClient,
		toolclient.NewClient(cfg),
		sessionLogger,
		cfg.ProviderModelAlias,
	)

	s := &Server{
		cfg:          cfg,
		provider:     providerClient,
		orchestrator: orch,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)

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
	resp := map[string]any{
		"object": "list",
		"data": []map[string]any{
			{
				"id":       s.cfg.ProviderModelAlias,
				"object":   "model",
				"created":  time.Now().Unix(),
				"owned_by": "local",
			},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req types.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, _, _, err := s.orchestrator.RunTurn(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "provider_error", err.Error())
		return
	}

	resp.Model = s.cfg.ProviderModelAlias
	if req.Stream {
		writeSSEChatCompletion(w, s.cfg.ProviderModelAlias, envelope.FirstContent(resp))
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func writeSSEChatCompletion(w http.ResponseWriter, model, content string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

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
