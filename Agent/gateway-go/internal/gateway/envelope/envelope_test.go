package envelope

import "testing"

func TestParseAcceptsFencedToolCallWithLegacyActionShape(t *testing.T) {
	content := "正在帮你打开计算器...\n\n```json\n{\n  \"type\": \"tool_call\",\n  \"action\": {\n    \"name\": \"terminal\",\n    \"arguments\": {\n      \"command\": \"start calc\"\n    }\n  }\n}\n```"

	env, ok := Parse(content)
	if !ok {
		t.Fatal("Parse returned false")
	}
	if env.Type != "tool_call" {
		t.Fatalf("env.Type = %q, want tool_call", env.Type)
	}
	if env.Tool != "terminal" {
		t.Fatalf("env.Tool = %q, want terminal", env.Tool)
	}
	if env.Arguments["command"] != "start calc" {
		t.Fatalf("env.Arguments[command] = %v, want start calc", env.Arguments["command"])
	}
}

func TestParseAcceptsEmbeddedJSONObjectAnswer(t *testing.T) {
	content := "prefix text\n{\n  \"type\": \"answer\",\n  \"tool\": \"\",\n  \"arguments\": {},\n  \"reason\": \"done\",\n  \"content\": \"ok\"\n}\nextra text"

	env, ok := Parse(content)
	if !ok {
		t.Fatal("Parse returned false")
	}
	if env.Type != "answer" {
		t.Fatalf("env.Type = %q, want answer", env.Type)
	}
}

func TestParseInfersToolCallFromTopLevelNameShape(t *testing.T) {
	content := "{\n  \"name\": \"terminal\",\n  \"parameters\": {\n    \"command\": \"start calc\"\n  }\n}"

	env, ok := Parse(content)
	if !ok {
		t.Fatal("Parse returned false")
	}
	if env.Type != "tool_call" {
		t.Fatalf("env.Type = %q, want tool_call", env.Type)
	}
	if env.Tool != "terminal" {
		t.Fatalf("env.Tool = %q, want terminal", env.Tool)
	}
	if env.Arguments["command"] != "start calc" {
		t.Fatalf("env.Arguments[command] = %v, want start calc", env.Arguments["command"])
	}
}

func TestParseDefaultsToTerminalForShellLikeArguments(t *testing.T) {
	content := "{\n  \"type\": \"tool_call\",\n  \"arguments\": {\n    \"command\": \"calc\",\n    \"shell\": \"powershell\"\n  }\n}"

	env, ok := Parse(content)
	if !ok {
		t.Fatal("Parse returned false")
	}
	if env.Tool != "terminal" {
		t.Fatalf("env.Tool = %q, want terminal", env.Tool)
	}
}

func TestParseInfersTerminalFromTaggedContent(t *testing.T) {
	content := "好的，我来帮你打开计算器。\n\n<tool_call>\n{\"type\": \"tool_call\", \"tool\": \"terminal\", \"arguments\": {\"command\": \"gnome-calculator &\"}}\n"

	env, ok := Parse(content)
	if !ok {
		t.Fatal("Parse returned false")
	}
	if env.Tool != "terminal" {
		t.Fatalf("env.Tool = %q, want terminal", env.Tool)
	}
}
