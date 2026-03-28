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

type SessionEntry struct {
	RequestID        string               `json:"request_id,omitempty"`
	Time             string               `json:"time"`
	Mode             string               `json:"mode"`
	Plugins          []string             `json:"plugins,omitempty"`
	Status           string               `json:"status"`
	Failure          string               `json:"failure,omitempty"`
	RetrievalSources []retrieval.Fragment `json:"retrieval_sources,omitempty"`
	AnswerPreview    string               `json:"answer_preview,omitempty"`
	Messages         []types.ChatMessage  `json:"messages,omitempty"`
}

func NewSessionLogger(sessionDir string) *SessionLogger {
	return &SessionLogger{sessionDir: sessionDir}
}

func (l *SessionLogger) Append(entry SessionEntry) {
	if l == nil || l.sessionDir == "" {
		return
	}
	if entry.Time == "" {
		entry.Time = time.Now().Format(time.RFC3339)
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

func NewSessionEntry(requestID string, active *mode.Active, req types.ChatCompletionRequest, fragments []retrieval.Fragment, answerPreview, status, failure string) SessionEntry {
	entry := SessionEntry{
		RequestID:        requestID,
		Status:           status,
		Failure:          failure,
		RetrievalSources: fragments,
		AnswerPreview:    answerPreview,
		Messages:         req.Messages,
	}
	if active != nil {
		entry.Mode = mode.BuildLabel(active)
		entry.Plugins = append([]string{}, active.Plugins...)
	}
	return entry
}
