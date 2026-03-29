package types

type ChatCompletionRequest struct {
	Model             string         `json:"model"`
	Messages          []ChatMessage  `json:"messages"`
	Tools             []ToolSpec     `json:"tools,omitempty"`
	ToolChoice        any            `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool          `json:"parallel_tool_calls,omitempty"`
	Stream            bool           `json:"stream,omitempty"`
	StreamOptions     map[string]any `json:"stream_options,omitempty"`
	Temperature       *float64       `json:"temperature,omitempty"`
	MaxTokens         *int           `json:"max_tokens,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	XPlugins          []string       `json:"x_plugins,omitempty"`
}

type ChatMessage struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	Name             string     `json:"name,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
}

type ToolSpec struct {
	Type     string       `json:"type"`
	Function FunctionSpec `json:"function"`
}

type FunctionSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatCompletionResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
	Choices           []ChatCompletionChoice `json:"choices"`
	Usage             map[string]any         `json:"usage,omitempty"`
	Timings           map[string]any         `json:"timings,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

func (r ChatCompletionRequest) UsesNativeTools() bool {
	if len(r.Tools) > 0 {
		return true
	}
	for _, message := range r.Messages {
		if message.ToolCallID != "" || len(message.ToolCalls) > 0 || message.Role == "tool" {
			return true
		}
	}
	return false
}
