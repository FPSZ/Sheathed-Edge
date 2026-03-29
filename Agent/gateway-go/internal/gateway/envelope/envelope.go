package envelope

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

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
	original := content
	content = normalizePayload(content)
	if !LooksLikeJSONObject(content) {
		return Action{}, false
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return Action{}, false
	}
	env := normalizeAction(raw, content, original)
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
	content = strings.TrimSpace(stripThinkingBlocks(content))
	if fenced, ok := extractFencedBlock(content); ok {
		content = fenced
	}

	if object, ok := extractJSONObject(content); ok {
		return object
	}
	return content
}

// stripThinkingBlocks removes <think>...</think> blocks so that JSON envelope
// extraction is not confused by reasoning prefixes emitted by the model.
func stripThinkingBlocks(content string) string {
	for {
		start := strings.Index(content, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(content[start:], "</think>")
		if end == -1 {
			content = content[:start]
			break
		}
		content = content[:start] + content[start+end+len("</think>"):]
	}
	return content
}

func renderContent(content any) string {
	switch v := content.(type) {
	case nil:
		return ""
	case string:
		return renderStringContent(v)
	case map[string]any:
		if rendered, ok := renderStructuredAnswer(v); ok {
			return rendered
		}
		return prettyJSON(v)
	case []any:
		return prettyJSON(v)
	default:
		return prettyJSON(v)
	}
}

func extractFencedBlock(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	var body []string
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				inFence = true
				body = body[:0]
				continue
			}
			return strings.TrimSpace(strings.Join(body, "\n")), true
		}
		if inFence {
			body = append(body, line)
		}
	}
	return "", false
}

func extractJSONObject(content string) (string, bool) {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "{")
	if start == -1 {
		return "", false
	}
	content = content[start:]

	depth := 0
	inString := false
	escaped := false
	for idx, r := range content {
		switch {
		case escaped:
			escaped = false
		case r == '\\' && inString:
			escaped = true
		case r == '"':
			inString = !inString
		case !inString && r == '{':
			depth++
		case !inString && r == '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(content[:idx+utf8.RuneLen(r)]), true
			}
		}
	}
	return "", false
}

func normalizeAction(raw map[string]any, normalizedContent string, originalContent string) Action {
	env := Action{
		Type:      getString(raw["type"]),
		Tool:      getString(raw["tool"]),
		Arguments: firstNonNilMap(raw["arguments"], raw["params"], raw["parameters"]),
		Reason:    getString(raw["reason"]),
		Content:   raw["content"],
	}
	if env.Tool == "" {
		env.Tool = getString(raw["name"])
	}

	if action, ok := raw["action"].(map[string]any); ok {
		if env.Tool == "" {
			env.Tool = firstNonEmptyString(action["name"], action["tool"], action["action"])
		}
		if len(env.Arguments) == 0 {
			env.Arguments = firstNonNilMap(action["arguments"], action["params"], action["parameters"])
		}
	}
	if function, ok := raw["function"].(map[string]any); ok {
		if env.Tool == "" {
			env.Tool = firstNonEmptyString(function["name"], function["tool"])
		}
		if len(env.Arguments) == 0 {
			env.Arguments = firstNonNilMap(function["arguments"], function["params"], function["parameters"])
		}
	}

	if env.Type == "" && env.Tool != "" {
		env.Type = "tool_call"
	}
	if env.Tool == "" && looksLikeTerminalArguments(env.Arguments) {
		env.Tool = "terminal"
		if env.Type == "" {
			env.Type = "tool_call"
		}
	}
	if env.Tool == "" && env.Type == "tool_call" && mentionsTerminal(normalizedContent, originalContent) {
		env.Tool = "terminal"
	}

	if env.Arguments == nil {
		env.Arguments = map[string]any{}
	}
	return env
}

func getString(value any) string {
	str, _ := value.(string)
	return str
}

func getMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	if mapped, ok := value.(map[string]any); ok {
		return mapped
	}
	return nil
}

func firstNonEmptyString(values ...any) string {
	for _, value := range values {
		if str := getString(value); str != "" {
			return str
		}
	}
	return ""
}

func firstNonNilMap(values ...any) map[string]any {
	for _, value := range values {
		if mapped := getMap(value); mapped != nil {
			return mapped
		}
	}
	return nil
}

func looksLikeTerminalArguments(arguments map[string]any) bool {
	if len(arguments) == 0 {
		return false
	}
	terminalKeys := []string{"command", "shell", "workdir", "timeout_ms"}
	for _, key := range terminalKeys {
		if _, ok := arguments[key]; ok {
			return true
		}
	}
	return false
}

func mentionsTerminal(contents ...string) bool {
	for _, content := range contents {
		lowered := strings.ToLower(content)
		if strings.Contains(lowered, "\"tool\": \"terminal\"") ||
			strings.Contains(lowered, "\"name\": \"terminal\"") ||
			strings.Contains(lowered, "<tool_call>") && strings.Contains(lowered, "terminal") ||
			strings.Contains(lowered, "`terminal`") {
			return true
		}
	}
	return false
}

func renderStructuredAnswer(content map[string]any) (string, bool) {
	ordered := []struct {
		key   string
		label string
	}{
		{key: "attack_surface", label: "Attack Surface"},
		{key: "evidence", label: "Evidence"},
		{key: "recommended_action", label: "Recommended Action"},
		{key: "patch_plan", label: "Patch Plan"},
		{key: "regression_risks", label: "Regression Risks"},
		{key: "next_needed_inputs", label: "Next Needed Inputs"},
	}

	requiredKeys := 0
	for _, item := range ordered {
		if _, ok := content[item.key]; ok {
			requiredKeys++
		}
	}
	if requiredKeys < 3 {
		return "", false
	}

	var sections []string
	for _, item := range ordered {
		value, ok := content[item.key]
		if !ok {
			continue
		}
		rendered := renderSectionValue(value)
		if rendered == "" {
			continue
		}
		sections = append(sections, fmt.Sprintf("%s:\n%s", item.label, rendered))
	}
	if len(sections) == 0 {
		return "", false
	}
	return strings.Join(sections, "\n\n"), true
}

func renderSectionValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []any:
		var lines []string
		for _, item := range v {
			text := strings.TrimSpace(renderSectionValue(item))
			if text == "" {
				continue
			}
			lines = append(lines, "- "+text)
		}
		return strings.Join(lines, "\n")
	default:
		return prettyJSON(v)
	}
}

func prettyJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

func renderStringContent(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if LooksLikeJSONObject(trimmed) {
		var parsed any
		if err := json.Unmarshal([]byte(normalizePayload(trimmed)), &parsed); err == nil {
			switch v := parsed.(type) {
			case map[string]any:
				if rendered, ok := renderStructuredAnswer(v); ok {
					return rendered
				}
				return prettyJSON(v)
			case []any:
				return prettyJSON(v)
			}
		}
	}
	return value
}
