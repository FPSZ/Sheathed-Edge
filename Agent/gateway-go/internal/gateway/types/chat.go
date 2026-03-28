package types

type ChatCompletionRequest struct {
	Model       string         `json:"model"`
	Messages    []ChatMessage  `json:"messages"`
	Stream      bool           `json:"stream,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	MaxTokens   *int           `json:"max_tokens,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	XPlugins    []string       `json:"x_plugins,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}
