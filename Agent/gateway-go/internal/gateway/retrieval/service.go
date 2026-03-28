package retrieval

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/config"
	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/pathutil"
)

type Fragment struct {
	Source string `json:"source"`
	Text   string `json:"text"`
}

type Service struct {
	command          string
	maxFragments     int
	maxFragmentChars int
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		command:          cfg.Retrieval.FallbackCommand,
		maxFragments:     cfg.Retrieval.MaxFragments,
		maxFragmentChars: cfg.Retrieval.MaxFragmentChars,
	}
}

func (r *Service) Search(ctx context.Context, query string, roots []string) ([]Fragment, error) {
	query = strings.TrimSpace(query)
	if query == "" || len(roots) == 0 {
		return nil, nil
	}

	args := []string{
		"-n",
		"-S",
		"--hidden",
		"--max-count", fmt.Sprintf("%d", r.maxFragments),
		query,
	}
	for _, root := range roots {
		args = append(args, pathutil.NormalizeRuntimePath(root))
	}

	cmd := exec.CommandContext(ctx, r.command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("retrieval command failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	var out []Fragment
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		text := line
		if len(text) > r.maxFragmentChars {
			text = text[:r.maxFragmentChars]
		}
		parts := strings.SplitN(line, ":", 3)
		source := line
		if len(parts) >= 2 {
			source = parts[0] + ":" + parts[1]
		}
		out = append(out, Fragment{
			Source: source,
			Text:   text,
		})
	}

	return out, nil
}
