package openai

import (
	"fmt"
	"strings"
	"time"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/types"
)

func newRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func validateChatResponse(resp *types.ChatCompletionResponse) error {
	if resp == nil {
		return fmt.Errorf("gateway returned nil response")
	}
	if len(resp.Choices) == 0 {
		return fmt.Errorf("gateway returned no choices")
	}
	message := resp.Choices[0].Message
	if strings.TrimSpace(message.Content) == "" &&
		strings.TrimSpace(message.ReasoningContent) == "" &&
		len(message.ToolCalls) == 0 {
		return fmt.Errorf("gateway returned empty assistant content")
	}
	return nil
}
