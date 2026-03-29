package orchestrator

import "testing"

func TestClassifyTurnTreatsLocalMachineActionsAsTasks(t *testing.T) {
	cases := []string{
		"打开我的电脑的计算器",
		"帮我启动 powershell",
		"open calculator on this computer",
		"restart the local service",
	}

	for _, query := range cases {
		if got := classifyTurn(query); got != turnKindTask {
			t.Fatalf("classifyTurn(%q) = %v, want task", query, got)
		}
	}
}

func TestClassifyTurnKeepsCasualConversationAsConversation(t *testing.T) {
	cases := []string{
		"你好",
		"你是谁",
		"今天天气怎么样",
	}

	for _, query := range cases {
		if got := classifyTurn(query); got != turnKindConversation {
			t.Fatalf("classifyTurn(%q) = %v, want conversation", query, got)
		}
	}
}
