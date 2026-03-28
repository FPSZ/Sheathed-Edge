package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/mode"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/retrieval"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/types"
)

type SessionLogger struct {
	sessionDir string
}

func NewSessionLogger(sessionDir string) *SessionLogger {
	return &SessionLogger{sessionDir: sessionDir}
}

func (l *SessionLogger) Append(active *mode.Active, req types.ChatCompletionRequest, fragments []retrieval.Fragment, answerPreview string) {
	if l == nil || l.sessionDir == "" {
		return
	}
	entry := map[string]any{
		"time":              time.Now().Format(time.RFC3339),
		"mode":              mode.BuildLabel(active),
		"plugins":           active.Plugins,
		"retrieval_sources": fragments,
		"answer_preview":    answerPreview,
		"messages":          req.Messages,
	}
	data, _ := json.Marshal(entry)
	logPath := filepath.Join(l.sessionDir, time.Now().Format("2006-01-02")+".jsonl")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}
