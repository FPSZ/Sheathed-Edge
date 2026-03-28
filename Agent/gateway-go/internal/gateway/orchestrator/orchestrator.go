package orchestrator

import (
	"context"
	"strings"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/envelope"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/logging"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/mode"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/provider"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/retrieval"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/toolclient"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/types"
)

type Orchestrator struct {
	modeLoader *mode.Loader
	retrieval  *retrieval.Service
	provider   *provider.Client
	toolClient *toolclient.Client
	logger     *logging.SessionLogger
	modelAlias string
}

func New(modeLoader *mode.Loader, retrievalSvc *retrieval.Service, providerClient *provider.Client, toolClient *toolclient.Client, logger *logging.SessionLogger, modelAlias string) *Orchestrator {
	return &Orchestrator{
		modeLoader: modeLoader,
		retrieval:  retrievalSvc,
		provider:   providerClient,
		toolClient: toolClient,
		logger:     logger,
		modelAlias: modelAlias,
	}
}

func (o *Orchestrator) RunTurn(ctx context.Context, req types.ChatCompletionRequest) (*types.ChatCompletionResponse, *mode.Active, []retrieval.Fragment, error) {
	plugins := extractPlugins(req)
	active, err := o.modeLoader.Load(plugins)
	if err != nil {
		return nil, nil, nil, err
	}

	query := latestUserMessage(req.Messages)
	retrievalCtx, cancel := context.WithCancel(ctx)
	fragments, _ := o.retrieval.Search(retrievalCtx, query, active.RetrievalRoots)
	cancel()

	upstreamReq := req
	upstreamReq.Model = o.modelAlias
	upstreamReq.Messages = prependSystemContext(req.Messages, active.SystemPrompt, fragments)
	upstreamReq.Stream = false

	result, err := o.provider.ChatCompletion(ctx, upstreamReq)
	if err != nil {
		return nil, active, fragments, err
	}

	finalResp := result
	if env, ok := envelope.Parse(envelope.FirstContent(result)); ok && env.Type == "tool_call" {
		finalResp = o.normalizeEnvelopeResponse(o.handleToolCall(ctx, req, active, env, result))
	} else if env, ok := envelope.Parse(envelope.FirstContent(result)); ok && env.Type == "answer" {
		finalResp = envelope.UnwrapAnswer(result, env)
	} else if envelope.LooksLikeJSONObject(envelope.FirstContent(result)) {
		finalResp = o.repairEnvelope(ctx, req, active, result)
	}

	finalResp.Model = o.modelAlias
	o.logger.Append(active, req, fragments, envelope.FirstContent(finalResp))
	return finalResp, active, fragments, nil
}

func (o *Orchestrator) handleToolCall(ctx context.Context, originalReq types.ChatCompletionRequest, active *mode.Active, env envelope.Action, providerResp *types.ChatCompletionResponse) *types.ChatCompletionResponse {
	sessionID := providerResp.ID
	resolveResp, err := o.toolClient.Resolve(ctx, toolclient.ResolveRequest{
		SessionID: sessionID,
		Mode:      mode.BuildLabel(active),
		Tool:      env.Tool,
		Arguments: env.Arguments,
	})
	if err != nil || !resolveResp.Allowed {
		return o.fallbackWithoutTools(ctx, originalReq, active, providerResp, err)
	}

	execResp, err := o.toolClient.Execute(ctx, toolclient.ExecuteRequest{
		SessionID: sessionID,
		Mode:      mode.BuildLabel(active),
		Tool:      env.Tool,
		Arguments: resolveResp.NormalizedArguments,
	})
	if err != nil || !execResp.OK {
		return o.fallbackWithoutTools(ctx, originalReq, active, providerResp, err)
	}

	nextReq := originalReq
	nextReq.Stream = false
	nextReq.Messages = append(
		prependSystemContext(originalReq.Messages, active.SystemPrompt, nil),
		types.ChatMessage{Role: "assistant", Content: envelope.FirstContent(providerResp)},
		types.ChatMessage{Role: "system", Content: "Tool result summary:\n" + execResp.Summary},
	)

	resp, err := o.provider.ChatCompletion(ctx, nextReq)
	if err != nil {
		return o.fallbackWithoutTools(ctx, originalReq, active, providerResp, err)
	}
	return resp
}

func (o *Orchestrator) fallbackWithoutTools(ctx context.Context, originalReq types.ChatCompletionRequest, active *mode.Active, providerResp *types.ChatCompletionResponse, cause error) *types.ChatCompletionResponse {
	note := "Tool execution is unavailable. Answer conservatively without tools."
	if cause != nil {
		note += "\nFailure: " + cause.Error()
	}
	req := originalReq
	req.Stream = false
	req.Messages = append(
		prependSystemContext(originalReq.Messages, active.SystemPrompt, nil),
		types.ChatMessage{Role: "assistant", Content: envelope.FirstContent(providerResp)},
		types.ChatMessage{Role: "system", Content: note},
	)
	resp, err := o.provider.ChatCompletion(ctx, req)
	if err != nil {
		return providerResp
	}
	return resp
}

func (o *Orchestrator) repairEnvelope(ctx context.Context, originalReq types.ChatCompletionRequest, active *mode.Active, providerResp *types.ChatCompletionResponse) *types.ChatCompletionResponse {
	req := originalReq
	req.Stream = false
	req.Messages = append(
		prependSystemContext(originalReq.Messages, active.SystemPrompt, nil),
		types.ChatMessage{Role: "assistant", Content: envelope.FirstContent(providerResp)},
		types.ChatMessage{Role: "system", Content: "Your previous output looked like malformed JSON. If you need a tool, emit a valid action envelope JSON. Otherwise answer normally under the output contract without partial JSON."},
	)
	resp, err := o.provider.ChatCompletion(ctx, req)
	if err != nil {
		return providerResp
	}
	return o.normalizeEnvelopeResponse(resp)
}

func (o *Orchestrator) normalizeEnvelopeResponse(resp *types.ChatCompletionResponse) *types.ChatCompletionResponse {
	env, ok := envelope.Parse(envelope.FirstContent(resp))
	if !ok {
		return resp
	}
	if env.Type == "answer" {
		return envelope.UnwrapAnswer(resp, env)
	}
	return resp
}

func prependSystemContext(messages []types.ChatMessage, systemPrompt string, fragments []retrieval.Fragment) []types.ChatMessage {
	var parts []string
	if strings.TrimSpace(systemPrompt) != "" {
		parts = append(parts, strings.TrimSpace(systemPrompt))
	}
	if len(fragments) > 0 {
		var b strings.Builder
		b.WriteString("Retrieved local context:\n")
		for _, frag := range fragments {
			b.WriteString("- ")
			b.WriteString(frag.Source)
			b.WriteString("\n")
			b.WriteString(frag.Text)
			b.WriteString("\n")
		}
		parts = append(parts, strings.TrimSpace(b.String()))
	}

	if len(parts) == 0 {
		return messages
	}
	system := types.ChatMessage{
		Role:    "system",
		Content: strings.Join(parts, "\n\n"),
	}
	return append([]types.ChatMessage{system}, messages...)
}

func extractPlugins(req types.ChatCompletionRequest) []string {
	out := append([]string{}, req.XPlugins...)
	if plugins, ok := req.Metadata["plugins"].([]any); ok {
		for _, item := range plugins {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
	}
	return uniqueStrings(nil, out)
}

func latestUserMessage(messages []types.ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return ""
}

func uniqueStrings(base []string, add []string) []string {
	seen := make(map[string]struct{}, len(base))
	for _, item := range base {
		seen[item] = struct{}{}
	}
	for _, item := range add {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		base = append(base, item)
	}
	return base
}
