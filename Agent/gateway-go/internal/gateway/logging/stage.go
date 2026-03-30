package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type StageLogger struct {
	auditDir string
}

type StageTrace struct {
	logger    *StageLogger
	requestID string

	mu        sync.RWMutex
	mode      string
	plugins   []string
	userEmail string
}

type StageEntry struct {
	Time       string   `json:"time"`
	RequestID  string   `json:"request_id"`
	UserEmail  string   `json:"user_email,omitempty"`
	Mode       string   `json:"mode,omitempty"`
	Plugins    []string `json:"plugins,omitempty"`
	Stage      string   `json:"stage"`
	DurationMS int64    `json:"duration_ms"`
	OK         bool     `json:"ok"`
	Reason     string   `json:"reason,omitempty"`
}

type StageSpan struct {
	trace *StageTrace
	stage string
	start time.Time
}

func NewStageLogger(auditDir string) *StageLogger {
	return &StageLogger{auditDir: auditDir}
}

func (l *StageLogger) NewTrace(requestID string) *StageTrace {
	return &StageTrace{
		logger:    l,
		requestID: requestID,
	}
}

func (t *StageTrace) SetContext(mode string, plugins []string, userEmail string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.mode = mode
	t.plugins = append([]string{}, plugins...)
	t.userEmail = userEmail
}

func (t *StageTrace) Begin(stage string) *StageSpan {
	return &StageSpan{
		trace: t,
		stage: stage,
		start: time.Now(),
	}
}

func (s *StageSpan) End(ok bool, reason string) {
	if s == nil || s.trace == nil || s.trace.logger == nil || s.trace.logger.auditDir == "" {
		return
	}

	s.trace.mu.RLock()
	mode := s.trace.mode
	plugins := append([]string{}, s.trace.plugins...)
	userEmail := s.trace.userEmail
	s.trace.mu.RUnlock()

	entry := StageEntry{
		Time:       time.Now().Format(time.RFC3339),
		RequestID:  s.trace.requestID,
		UserEmail:  userEmail,
		Mode:       mode,
		Plugins:    plugins,
		Stage:      s.stage,
		DurationMS: time.Since(s.start).Milliseconds(),
		OK:         ok,
		Reason:     reason,
	}
	s.trace.logger.append(entry)
}

func (l *StageLogger) append(entry StageEntry) {
	if l == nil || l.auditDir == "" {
		return
	}
	data, _ := json.Marshal(entry)
	logPath := filepath.Join(l.auditDir, time.Now().Format("2006-01-02")+".jsonl")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}
