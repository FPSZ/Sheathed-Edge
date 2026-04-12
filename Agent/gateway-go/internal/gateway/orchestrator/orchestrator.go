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
	active, err := o.modeLoader.Load(plugins, extractAgentLayers(req))
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

func (o *Orchestrator) PrepareNativeStreamingRequest(req types.ChatCompletionRequest, responseModel string) (types.ChatCompletionRequest, error) {
	active, fragments, turnPrompt, err := o.loadTurnContext(context.Background(), req)
	if err != nil {
		return types.ChatCompletionRequest{}, err
	}

	upstreamReq := req
	upstreamReq.Model = responseModel
	upstreamReq.Stream = true
	upstreamReq.StreamOptions = mergeStreamOptions(req.StreamOptions, map[string]any{
		"include_usage": true,
	})
	upstreamReq.Messages = prependSystemContext(req.Messages, buildNativeToolPrompt(turnPrompt), fragments)

	_ = active
	return upstreamReq, nil
}

func (o *Orchestrator) RunTurn(ctx context.Context, requestID string, responseModel string, req types.ChatCompletionRequest, trace *logging.StageTrace) (*types.ChatCompletionResponse, *mode.Active, []retrieval.Fragment, error) {
	return o.executeTurnLogic(ctx, requestID, responseModel, req, false, trace)
}

func (o *Orchestrator) RunNativeToolTurn(ctx context.Context, requestID string, responseModel string, req types.ChatCompletionRequest, trace *logging.StageTrace) (*types.ChatCompletionResponse, *mode.Active, []retrieval.Fragment, error) {
	return o.executeTurnLogic(ctx, requestID, responseModel, req, true, trace)
}

func (o *Orchestrator) executeTurnLogic(ctx context.Context, requestID string, responseModel string, originalReq types.ChatCompletionRequest, isNativeTools bool, trace *logging.StageTrace) (*types.ChatCompletionResponse, *mode.Active, []retrieval.Fragment, error) {
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
			if isNativeTools {
				if len(finalResp.Choices) > 0 {
					answerPreview = finalResp.Choices[0].Message.Content
				}
			} else {
				answerPreview = envelope.FirstContent(finalResp)
			}
		}
		o.logger.Append(logging.NewSessionEntry(requestID, active, originalReq, fragments, answerPreview, status, failure))
	}()

	var turnPrompt string
	var err error
	active, fragments, turnPrompt, err = o.loadTurnContext(ctx, originalReq)
	if err != nil {
		runErr = err
		return nil, nil, nil, err
	}
	if trace != nil {
		trace.SetContext(mode.BuildLabel(active), active.Plugins, originalReq.UserEmail, active.AgentLayers)
	}

	req := originalReq
	req.Model = responseModel
	req.Stream = false
	
	if isNativeTools {
		req.Messages = prependSystemContext(originalReq.Messages, buildNativeToolPrompt(turnPrompt), fragments)
	} else {
		req.Messages = prependSystemContext(originalReq.Messages, turnPrompt, fragments)
	}

	const maxTurns = 10

	for turn := 0; turn < maxTurns; turn++ {
		span := trace.Begin(fmt.Sprintf("provider_turn_%d", turn))
		resp, err := o.provider.ChatCompletion(ctx, req)
		if err != nil {
			span.End(false, err.Error())
			runErr = err
			return nil, active, fragments, err
		}
		span.End(true, "")

		if isNativeTools {
			if len(resp.Choices) == 0 {
				finalResp = resp
				return finalResp, active, fragments, nil
			}
			msg := resp.Choices[0].Message
			
			if len(msg.ToolCalls) == 0 {
				finalResp = resp
				return finalResp, active, fragments, nil
			}

			req.Messages = append(req.Messages, msg)
			hasToolError := false

			for _, tc := range msg.ToolCalls {
				if tc.Function.Name == "terminal" {
					req.Messages = append(req.Messages, types.ChatMessage{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    "terminal is handled by the Open WebUI external OpenAPI tool path, not the Gateway legacy tool path",
					})
					hasToolError = true
					continue
				}

				var args map[string]any
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					req.Messages = append(req.Messages, types.ChatMessage{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    fmt.Sprintf("invalid arguments json: %v", err),
					})
					hasToolError = true
					continue
				}

				resolveSpan := trace.Begin("tool_resolve")
				resolveResp, err := o.toolClient.Resolve(ctx, toolclient.ResolveRequest{
					SessionID: resp.ID,
					Mode:      mode.BuildLabel(active),
					Tool:      tc.Function.Name,
					UserEmail: originalReq.UserEmail,
					Arguments: args,
				})
				if err != nil || !resolveResp.Allowed {
					reason := "tool resolve denied"
					if err != nil {
						reason = fmt.Sprintf("tool resolve failed: %v", err)
					} else if resolveResp != nil && resolveResp.Reason != "" {
						reason = resolveResp.Reason
					}
					resolveSpan.End(false, reason)
					req.Messages = append(req.Messages, types.ChatMessage{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    reason,
					})
					hasToolError = true
					continue
				}
				resolveSpan.End(true, "")

				execSpan := trace.Begin(fmt.Sprintf("tool_execute_%s", tc.Function.Name))
				execResp, err := o.toolClient.Execute(ctx, toolclient.ExecuteRequest{
					SessionID: resp.ID,
					Mode:      mode.BuildLabel(active),
					Tool:      tc.Function.Name,
					UserEmail: originalReq.UserEmail,
					Arguments: resolveResp.NormalizedArguments,
				})
				if err != nil || !execResp.OK {
					reason := "tool execute failed"
					if err != nil {
						reason = err.Error()
					} else if execResp != nil && execResp.Error != nil && execResp.Error.Message != "" {
						reason = execResp.Error.Message
					}
					execSpan.End(false, reason)
					req.Messages = append(req.Messages, types.ChatMessage{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    reason,
					})
					hasToolError = true
					continue
				}
				execSpan.End(true, "")

				req.Messages = append(req.Messages, types.ChatMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    buildToolResultBlock(execResp),
				})
			}
			
			if hasToolError {
				req.Messages = append(req.Messages, types.ChatMessage{
					Role:    "user",
					Content: "Some tools failed. Please analyze the errors and either fix your arguments or answer directly without using tools.",
				})
			}
			continue
		} else {
			content := envelope.FirstContent(resp)
			if env, ok := envelope.Parse(content); ok {
				if env.Type == "answer" {
					finalResp = envelope.UnwrapAnswer(resp, env)
					return finalResp, active, fragments, nil
				}
				if env.Type == "tool_call" {
					if env.Tool == "terminal" {
						req.Messages = append(req.Messages, 
							types.ChatMessage{Role: "assistant", Content: content},
							types.ChatMessage{Role: "user", Content: buildFailClosedBlock(env.Tool, "terminal is handled by the Open WebUI external OpenAPI tool path, not the Gateway legacy tool path")},
							types.ChatMessage{Role: "user", Content: "Do not call any more tools. Return only a final answer envelope JSON with type=answer."},
						)
						continue
					}

					resolveSpan := trace.Begin("tool_resolve")
					resolveResp, err := o.toolClient.Resolve(ctx, toolclient.ResolveRequest{
						SessionID: resp.ID, Mode: mode.BuildLabel(active), Tool: env.Tool, UserEmail: originalReq.UserEmail, Arguments: env.Arguments,
					})
					if err != nil || !resolveResp.Allowed {
						reason := "tool resolve denied"
						if err != nil {
							reason = fmt.Sprintf("tool resolve failed: %v", err)
						} else if resolveResp != nil && resolveResp.Reason != "" {
							reason = resolveResp.Reason
						}
						resolveSpan.End(false, reason)
						req.Messages = append(req.Messages,
							types.ChatMessage{Role: "assistant", Content: content},
							types.ChatMessage{Role: "user", Content: buildFailClosedBlock(env.Tool, reason)},
							types.ChatMessage{Role: "user", Content: "Do not call any more tools. Return only a final answer envelope JSON with type=answer."},
						)
						continue
					}
					resolveSpan.End(true, "")

					execSpan := trace.Begin("tool_execute")
					execResp, err := o.toolClient.Execute(ctx, toolclient.ExecuteRequest{
						SessionID: resp.ID, Mode: mode.BuildLabel(active), Tool: env.Tool, UserEmail: originalReq.UserEmail, Arguments: resolveResp.NormalizedArguments,
					})
					if err != nil || !execResp.OK {
						reason := "tool execute failed"
						if err != nil {
							reason = err.Error()
						} else if execResp != nil && execResp.Error != nil && execResp.Error.Message != "" {
							reason = execResp.Error.Message
						}
						execSpan.End(false, reason)
						req.Messages = append(req.Messages,
							types.ChatMessage{Role: "assistant", Content: content},
							types.ChatMessage{Role: "user", Content: buildFailClosedBlock(env.Tool, reason)},
							types.ChatMessage{Role: "user", Content: "Do not call any more tools. Return only a final answer envelope JSON with type=answer."},
						)
						continue
					}
					execSpan.End(true, "")

					req.Messages = append(req.Messages,
						types.ChatMessage{Role: "assistant", Content: content},
						types.ChatMessage{Role: "user", Content: buildToolResultBlock(execResp)},
						types.ChatMessage{Role: "user", Content: "Use the tool result to answer the latest user request directly. Return only a final answer envelope JSON with type=answer."},
					)
					continue
				}
			}

			if envelope.LooksLikeJSONObject(content) {
				req.Messages = append(req.Messages,
					types.ChatMessage{Role: "assistant", Content: content},
					types.ChatMessage{Role: "user", Content: "Your previous output looked like malformed JSON. If you need a tool, emit a valid action envelope JSON. Otherwise answer normally under the output contract without partial JSON."},
				)
				continue
			}

			finalResp = resp
			return finalResp, active, fragments, nil
		}
	}

	finalResp = failClosedToolResponse(responseModel, "", "Exceeded maximum query loops (N=10) without returning a final answer.")
	return finalResp, active, fragments, nil
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

func (o *Orchestrator) loadTurnContext(ctx context.Context, req types.ChatCompletionRequest) (*mode.Active, []retrieval.Fragment, string, error) {
	plugins := extractPlugins(req)
	active, err := o.modeLoader.Load(plugins, extractAgentLayers(req))
	if err != nil {
		return nil, nil, "", err
	}

	query := latestUserMessage(req.Messages)
	turnKind := classifyTurn(query)
	turnPrompt := active.SystemPrompt
	var fragments []retrieval.Fragment
	if turnKind == turnKindConversation {
		turnPrompt = buildConversationPrompt(active.ConversationPrompt)
	} else {
		retrievalCtx, cancel := context.WithCancel(ctx)
		fragments, _ = o.retrieval.Search(retrievalCtx, query, active.RetrievalRoots)
		cancel()
	}

	return active, fragments, turnPrompt, nil
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
	out = uniqueStrings(nil, out)
	layers := extractAgentLayers(req)
	if layers != nil {
		out = applyLayerPluginFilter(out, *layers)
	}
	return out
}

func extractAgentLayers(req types.ChatCompletionRequest) *mode.SessionAgentLayers {
	if req.Metadata == nil {
		return nil
	}
	raw, ok := req.Metadata["agent_layers"]
	if !ok || raw == nil {
		return nil
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var parsed mode.SessionAgentLayers
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil
	}
	return &parsed
}

func applyLayerPluginFilter(plugins []string, layers mode.SessionAgentLayers) []string {
	out := make([]string, 0, len(plugins)+2)
	seen := make(map[string]struct{}, len(plugins))
	for _, plugin := range plugins {
		switch plugin {
		case "pwn":
			if !layers.EnablePwnSkills {
				continue
			}
		case "web":
			if !layers.EnableWebSkills {
				continue
			}
		}
		if _, exists := seen[plugin]; exists {
			continue
		}
		seen[plugin] = struct{}{}
		out = append(out, plugin)
	}
	if layers.EnablePwnSkills {
		if _, exists := seen["pwn"]; !exists {
			out = append(out, "pwn")
			seen["pwn"] = struct{}{}
		}
	}
	if layers.EnableWebSkills {
		if _, exists := seen["web"]; !exists {
			out = append(out, "web")
		}
	}
	return out
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
	if isTaskOrAnalysisTurn(normalized) || isLocalActionTurn(normalized) {
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

func isLocalActionTurn(normalized string) bool {
	actionHints := []string{
		"open", "launch", "start", "stop", "restart", "run", "execute",
		"打开", "启动", "运行", "关闭", "停止", "重启", "执行",
	}
	targetHints := []string{
		"calculator", "calc", "notepad", "terminal", "powershell", "cmd", "explorer", "task manager", "service", "process", "port", "local", "computer",
		"计算器", "记事本", "终端", "powershell", "cmd", "文件夹", "目录", "资源管理器", "任务管理器", "程序", "应用", "服务", "端口", "进程", "本地", "电脑",
	}

	hasAction := false
	for _, hint := range actionHints {
		if strings.Contains(normalized, hint) {
			hasAction = true
			break
		}
	}
	if !hasAction {
		return false
	}

	for _, hint := range targetHints {
		if strings.Contains(normalized, hint) {
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
- Do not call tools for casual chat, greetings, or meta questions.
- If the user asks you to operate the local machine or inspect local state, that is tool-driven work.
`)
}

func buildConversationPrompt(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return conversationSystemPrompt()
	}
	return base + "\n\n" + conversationSystemPrompt()
}

func buildNativeToolPrompt(base string) string {
	const nativeToolDirective = "Native OpenAI tools are available in this request. Do not emit the legacy action-envelope JSON. When a tool is needed, use the native tool calling interface provided by the client. Do not wrap tool calls in markdown or code fences. After receiving tool result messages, answer normally in natural language."

	base = strings.TrimSpace(base)
	if base == "" {
		return nativeToolDirective
	}
	return base + "\n\n" + nativeToolDirective
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
