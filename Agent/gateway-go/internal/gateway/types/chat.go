package types

type ChatCompletionRequest struct {
	Model         string         `json:"model"`
	Messages      []ChatMessage  `json:"messages"`
	Stream        bool           `json:"stream,omitempty"`
	StreamOptions map[string]any `json:"stream_options,omitempty"`
	Temperature   *float64       `json:"temperature,omitempty"`
	MaxTokens     *int           `json:"max_tokens,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	XPlugins      []string       `json:"x_plugins,omitempty"`
}

type ChatMessage struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
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
