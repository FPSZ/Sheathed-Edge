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
	if strings.HasPrefix(content, "```") {
		if fenced, ok := extractFencedBlock(content); ok {
			content = fenced
		}
	}

	if object, ok := extractLeadingJSONObject(content); ok {
		return object
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
	if len(lines) < 2 || !strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
		return "", false
	}

	var body []string
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "```" {
			return strings.TrimSpace(strings.Join(body, "\n")), true
		}
		body = append(body, line)
	}
	return "", false
}

func extractLeadingJSONObject(content string) (string, bool) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "{") {
		return "", false
	}

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
