package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
}

func New(modeLoader *mode.Loader, retrievalSvc *retrieval.Service, providerClient *provider.Client, toolClient *toolclient.Client, logger *logging.SessionLogger) *Orchestrator {
	return &Orchestrator{
		modeLoader: modeLoader,
		retrieval:  retrievalSvc,
		provider:   providerClient,
		toolClient: toolClient,
		logger:     logger,
	}
}

func (o *Orchestrator) PrepareStreamingRequest(req types.ChatCompletionRequest, responseModel string) (types.ChatCompletionRequest, bool, error) {
	plugins := extractPlugins(req)
	active, err := o.modeLoader.Load(plugins)
	if err != nil {
		return types.ChatCompletionRequest{}, false, err
	}

	query := latestUserMessage(req.Messages)
	if classifyTurn(query) != turnKindConversation {
		return types.ChatCompletionRequest{}, false, nil
	}

	upstreamReq := req
	upstreamReq.Model = responseModel
	upstreamReq.Stream = true
	upstreamReq.StreamOptions = mergeStreamOptions(req.StreamOptions, map[string]any{
		"include_usage": true,
	})
	upstreamReq.Messages = prependSystemContext(req.Messages, buildConversationPrompt(active.ConversationPrompt), nil)

	return upstreamReq, true, nil
}

func (o *Orchestrator) RunTurn(ctx context.Context, requestID string, responseModel string, req types.ChatCompletionRequest, trace *logging.StageTrace) (*types.ChatCompletionResponse, *mode.Active, []retrieval.Fragment, error) {
	var (
		active        *mode.Active
		fragments     []retrieval.Fragment
		finalResp     *types.ChatCompletionResponse
		runErr        error
		answerPreview string
	)
	defer func() {
		if o.logger == nil {
			return
		}
		status := "ok"
		failure := ""
		if runErr != nil {
			status = "failed"
			failure = runErr.Error()
		}
		if finalResp != nil {
			answerPreview = envelope.FirstContent(finalResp)
		}
		o.logger.Append(logging.NewSessionEntry(requestID, active, req, fragments, answerPreview, status, failure))
	}()

	plugins := extractPlugins(req)
	var err error
	active, err = o.modeLoader.Load(plugins)
	if err != nil {
		runErr = err
		return nil, nil, nil, err
	}
	if trace != nil {
		trace.SetContext(mode.BuildLabel(active), active.Plugins)
	}

	query := latestUserMessage(req.Messages)
	turnKind := classifyTurn(query)
	turnPrompt := active.SystemPrompt
	if turnKind == turnKindConversation {
		turnPrompt = buildConversationPrompt(active.ConversationPrompt)
	} else {
		retrievalCtx, cancel := context.WithCancel(ctx)
		fragments, _ = o.retrieval.Search(retrievalCtx, query, active.RetrievalRoots)
		cancel()
	}

	upstreamReq := req
	upstreamReq.Model = responseModel
	upstreamReq.Messages = prependSystemContext(req.Messages, turnPrompt, fragments)
	upstreamReq.Stream = false

	firstSpan := trace.Begin("provider_first")
	result, err := o.provider.ChatCompletion(ctx, upstreamReq)
	if err != nil {
		firstSpan.End(false, err.Error())
		runErr = err
		return nil, active, fragments, err
	}
	firstSpan.End(true, "")

	parseSpan := trace.Begin("envelope_parse")
	finalResp, err = o.resolveEnvelope(ctx, responseModel, req, active, turnPrompt, result, trace)
	if err != nil {
		parseSpan.End(false, err.Error())
		runErr = err
		return nil, active, fragments, err
	}
	parseSpan.End(true, "")

	finalResp.Model = responseModel
	return finalResp, active, fragments, nil
}

func (o *Orchestrator) resolveEnvelope(ctx context.Context, responseModel string, originalReq types.ChatCompletionRequest, active *mode.Active, turnPrompt string, providerResp *types.ChatCompletionResponse, trace *logging.StageTrace) (*types.ChatCompletionResponse, error) {
	content := envelope.FirstContent(providerResp)
	if env, ok := envelope.Parse(content); ok {
		switch env.Type {
		case "answer":
			return envelope.UnwrapAnswer(providerResp, env), nil
		case "tool_call":
			return o.handleToolCall(ctx, responseModel, originalReq, active, turnPrompt, env, providerResp, trace)
		default:
			return nil, fmt.Errorf("unsupported envelope type: %s", env.Type)
		}
	}
	if envelope.LooksLikeJSONObject(content) {
		return o.repairEnvelope(ctx, responseModel, originalReq, turnPrompt, providerResp)
	}
	return providerResp, nil
}

func (o *Orchestrator) handleToolCall(ctx context.Context, responseModel string, originalReq types.ChatCompletionRequest, active *mode.Active, turnPrompt string, env envelope.Action, providerResp *types.ChatCompletionResponse, trace *logging.StageTrace) (*types.ChatCompletionResponse, error) {
	sessionID := providerResp.ID
	resolveSpan := trace.Begin("tool_resolve")
	resolveResp, err := o.toolClient.Resolve(ctx, toolclient.ResolveRequest{
		SessionID: sessionID,
		Mode:      mode.BuildLabel(active),
		Tool:      env.Tool,
		Arguments: env.Arguments,
	})
	if err != nil {
		resolveSpan.End(false, err.Error())
		return o.fallbackWithoutTools(ctx, responseModel, originalReq, turnPrompt, providerResp, env.Tool, fmt.Sprintf("tool resolve failed: %v", err), trace)
	}
	if !resolveResp.Allowed {
		reason := strings.TrimSpace(resolveResp.Reason)
		if reason == "" {
			reason = "tool resolve denied"
		}
		resolveSpan.End(false, reason)
		return o.fallbackWithoutTools(ctx, responseModel, originalReq, turnPrompt, providerResp, env.Tool, reason, trace)
	}
	resolveSpan.End(true, "")

	execSpan := trace.Begin("tool_execute")
	execResp, err := o.toolClient.Execute(ctx, toolclient.ExecuteRequest{
		SessionID: sessionID,
		Mode:      mode.BuildLabel(active),
		Tool:      env.Tool,
		Arguments: resolveResp.NormalizedArguments,
	})
	if err != nil {
		execSpan.End(false, err.Error())
		return o.fallbackWithoutTools(ctx, responseModel, originalReq, turnPrompt, providerResp, env.Tool, fmt.Sprintf("tool execute failed: %v", err), trace)
	}
	if !execResp.OK {
		reason := "tool execute failed"
		if execResp.Error != nil && execResp.Error.Message != "" {
			reason = execResp.Error.Message
		}
		execSpan.End(false, reason)
		return o.fallbackWithoutTools(ctx, responseModel, originalReq, turnPrompt, providerResp, env.Tool, reason, trace)
	}
	execSpan.End(true, "")

	nextReq := originalReq
	nextReq.Model = responseModel
	nextReq.Stream = false
	nextReq.Messages = append(
		prependSystemContext(originalReq.Messages, turnPrompt, nil),
		types.ChatMessage{Role: "assistant", Content: envelope.FirstContent(providerResp)},
		types.ChatMessage{Role: "system", Content: buildToolResultBlock(execResp)},
		types.ChatMessage{Role: "system", Content: "You have exactly one tool result block. Do not request more tools. Return only a final answer envelope JSON with type=answer. Use the tool result to answer the latest user request directly. Do not echo raw tool JSON unless the user explicitly asked for raw JSON. If the user asked for a summary, produce concise natural-language summary points instead of copying the result object."},
	)

	secondSpan := trace.Begin("provider_second")
	resp, err := o.provider.ChatCompletion(ctx, nextReq)
	if err != nil {
		secondSpan.End(false, err.Error())
		return o.fallbackWithoutTools(ctx, responseModel, originalReq, turnPrompt, providerResp, env.Tool, fmt.Sprintf("provider second pass failed: %v", err), trace)
	}
	secondSpan.End(true, "")

	secondEnv, ok := envelope.Parse(envelope.FirstContent(resp))
	if !ok {
		return nil, fmt.Errorf("provider second pass returned invalid final envelope")
	}
	if secondEnv.Type == "tool_call" {
		return failClosedToolResponse(responseModel, env.Tool, execResp.Summary), nil
	}
	if secondEnv.Type != "answer" {
		return nil, fmt.Errorf("provider second pass returned unsupported envelope type: %s", secondEnv.Type)
	}
	return envelope.UnwrapAnswer(resp, secondEnv), nil
}

func (o *Orchestrator) fallbackWithoutTools(ctx context.Context, responseModel string, originalReq types.ChatCompletionRequest, turnPrompt string, providerResp *types.ChatCompletionResponse, toolName, cause string, trace *logging.StageTrace) (*types.ChatCompletionResponse, error) {
	req := originalReq
	req.Model = responseModel
	req.Stream = false
	req.Messages = append(
		prependSystemContext(originalReq.Messages, turnPrompt, nil),
		types.ChatMessage{Role: "assistant", Content: envelope.FirstContent(providerResp)},
		types.ChatMessage{Role: "system", Content: buildFailClosedBlock(toolName, cause)},
		types.ChatMessage{Role: "system", Content: "Do not call any more tools. Return only a final answer envelope JSON with type=answer. Answer the latest user request directly in natural language. Do not echo raw failure JSON. If evidence is insufficient, say so explicitly and answer conservatively."},
	)

	secondSpan := trace.Begin("provider_second")
	resp, err := o.provider.ChatCompletion(ctx, req)
	if err != nil {
		secondSpan.End(false, err.Error())
		return nil, err
	}
	secondSpan.End(true, "")

	env, ok := envelope.Parse(envelope.FirstContent(resp))
	if !ok {
		return failClosedToolResponse(responseModel, toolName, cause), nil
	}
	if env.Type == "answer" {
		return envelope.UnwrapAnswer(resp, env), nil
	}
	return failClosedToolResponse(responseModel, toolName, cause), nil
}

func (o *Orchestrator) repairEnvelope(ctx context.Context, responseModel string, originalReq types.ChatCompletionRequest, turnPrompt string, providerResp *types.ChatCompletionResponse) (*types.ChatCompletionResponse, error) {
	req := originalReq
	req.Model = responseModel
	req.Stream = false
	req.Messages = append(
		prependSystemContext(originalReq.Messages, turnPrompt, nil),
		types.ChatMessage{Role: "assistant", Content: envelope.FirstContent(providerResp)},
		types.ChatMessage{Role: "system", Content: "Your previous output looked like malformed JSON. If you need a tool, emit a valid action envelope JSON. Otherwise answer normally under the output contract without partial JSON."},
	)
	resp, err := o.provider.ChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}
	return o.normalizeEnvelopeResponse(resp)
}

func (o *Orchestrator) normalizeEnvelopeResponse(resp *types.ChatCompletionResponse) (*types.ChatCompletionResponse, error) {
	env, ok := envelope.Parse(envelope.FirstContent(resp))
	if !ok {
		return nil, fmt.Errorf("provider returned malformed envelope after repair")
	}
	if env.Type == "answer" {
		return envelope.UnwrapAnswer(resp, env), nil
	}
	if env.Type == "tool_call" {
		return resp, nil
	}
	return nil, fmt.Errorf("provider returned unsupported envelope type: %s", env.Type)
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

func mergeStreamOptions(current map[string]any, defaults map[string]any) map[string]any {
	if len(defaults) == 0 && len(current) == 0 {
		return nil
	}

	merged := make(map[string]any, len(current)+len(defaults))
	for k, v := range current {
		merged[k] = v
	}
	for k, v := range defaults {
		if _, ok := merged[k]; ok {
			continue
		}
		merged[k] = v
	}
	return merged
}

type turnClassification int

const (
	turnKindConversation turnClassification = iota
	turnKindTask
)

func classifyTurn(query string) turnClassification {
	normalized := normalizeUserText(query)
	if normalized == "" {
		return turnKindConversation
	}
	if isTaskOrAnalysisTurn(normalized) {
		return turnKindTask
	}
	return turnKindConversation
}

func normalizeUserText(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

func isTaskOrAnalysisTurn(normalized string) bool {
	taskHints := []string{
		"awdp", "ctf", "web", "pwn", "mcp", "tool", "retrieval", "radare2", "checksec", "payload", "patch", "exploit",
		"漏洞", "攻击", "分析", "审计", "修复", "补丁", "题目", "赛题", "靶场", "复盘", "写wp", "wp", "writeup",
		"帮我", "解题", "实现", "设计", "搭建", "配置", "接入", "调试", "排查", "测试", "日志", "代码", "脚本",
		"help me", "solve", "analyze", "debug", "fix", "implement", "design", "plan", "build", "review", "inspect",
		"gateway", "router", "server", "plugin", "mode", "schema", "config",
	}
	for _, hint := range taskHints {
		if strings.Contains(normalized, hint) {
			return true
		}
	}

	technicalMarkers := []string{
		"```", "/mnt/", "d:\\", "127.0.0.1", "http://", "https://", ".json", ".md", ".go", ".rs", ".py",
	}
	for _, marker := range technicalMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}

	return false
}

func conversationSystemPrompt() string {
	return strings.TrimSpace(`
You are replying in normal conversation mode.

Rules for this turn:
- Reply naturally in the user's language.
- Be helpful, direct, and human.
- AWDP is part of your expertise, not a mandatory output format.
- If the user asks what you do, who you are, or what you are good at, answer concretely as a local security assistant focused on AWDP, web security, pwn, patching, writeups, and tool-assisted analysis.
- Do not force security-analysis headings, audit structure, or JSON.
- Do not call tools unless the user explicitly asks for tool-driven work.
`)
}

func buildConversationPrompt(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return conversationSystemPrompt()
	}
	return base + "\n\n" + conversationSystemPrompt()
}

func buildToolResultBlock(resp *toolclient.ExecuteResponse) string {
	payload := map[string]any{
		"tool":    resp.Tool,
		"ok":      resp.OK,
		"summary": resp.Summary,
		"result":  resp.Result,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "Tool execution result is unavailable."
	}
	return "Tool execution result block:\n" + string(data)
}

func buildFailClosedBlock(toolName, cause string) string {
	payload := map[string]any{
		"tool":   toolName,
		"ok":     false,
		"reason": strings.TrimSpace(cause),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "Tool execution is unavailable. Answer conservatively without tools."
	}
	return "Tool execution failure block:\n" + string(data)
}

func failClosedToolResponse(modelAlias, toolName, summary string) *types.ChatCompletionResponse {
	content := "The tool path did not return a valid final answer envelope, so the gateway stopped further tool recursion and returned a conservative result."
	if strings.TrimSpace(toolName) != "" {
		content += "\nTool: " + toolName
	}
	if strings.TrimSpace(summary) != "" {
		content += "\nSummary: " + summary
	}
	return &types.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-failclosed-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelAlias,
		Choices: []types.ChatCompletionChoice{
			{
				Index:        0,
				Message:      types.ChatMessage{Role: "assistant", Content: content},
				FinishReason: "stop",
			},
		},
	}
}
