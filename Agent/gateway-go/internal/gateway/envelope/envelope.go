package envelope

import (
	"encoding/json"
	"strings"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/types"
)

type Action struct {
	Type      string         `json:"type"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
	Reason    string         `json:"reason"`
	Content   any            `json:"content"`
}

func Parse(content string) (Action, bool) {
	content = normalizePayload(content)
	if !LooksLikeJSONObject(content) {
		return Action{}, false
	}
	var env Action
	if err := json.Unmarshal([]byte(content), &env); err != nil {
		return Action{}, false
	}
	if env.Type == "" {
		return Action{}, false
	}
	return env, true
}

func LooksLikeJSONObject(content string) bool {
	content = normalizePayload(content)
	return strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}")
}

func FirstContent(resp *types.ChatCompletionResponse) string {
	if resp == nil || len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Content
}

func UnwrapAnswer(resp *types.ChatCompletionResponse, env Action) *types.ChatCompletionResponse {
	if resp == nil || len(resp.Choices) == 0 {
		return resp
	}
	cloned := *resp
	cloned.Choices = append([]types.ChatCompletionChoice(nil), resp.Choices...)
	cloned.Choices[0].Message.Content = renderContent(env.Content)
	cloned.Choices[0].FinishReason = "stop"
	return &cloned
}

func normalizePayload(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "```") {
		return content
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return content
	}
	if strings.HasPrefix(lines[0], "```") {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func renderContent(content any) string {
	switch v := content.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(data)
	}
}
