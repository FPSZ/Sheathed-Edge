package admin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	serviceLlama      = "llama-server"
	serviceGateway    = "gateway"
	serviceToolRouter = "tool-router"
	serviceOpenWebUI  = "open-webui"
	serviceHostAgent  = "host-agent"
)

func defaultControl(name string) ControlState {
	switch name {
	case serviceLlama, serviceToolRouter, serviceOpenWebUI, serviceHostAgent:
		return ControlState{
			CanStart: true,
			CanStop:  true,
		}
	case serviceGateway:
		return ControlState{
			UnsupportedReason: "gateway serves this admin ui and is not self-managed here",
		}
	default:
		return ControlState{
			UnsupportedReason: "service control is not configured",
		}
	}
}

func (s *Service) StartService(ctx context.Context, name string) error {
	switch strings.TrimSpace(name) {
	case serviceLlama:
		return s.host.Start(ctx)
	case serviceToolRouter:
		return s.startDetached(ctx, "tool-router", "/mnt/d/AI/Local/Workflows/wsl/start-tool-router.sh")
	case serviceOpenWebUI:
		return s.startDetached(ctx, "open-webui", "/mnt/d/AI/Local/Workflows/wsl/start-open-webui.sh")
	case serviceGateway:
		return fmt.Errorf("gateway start is not supported from the gateway itself")
	case serviceHostAgent:
		return s.startHostAgent(ctx)
	default:
		return fmt.Errorf("unsupported service: %s", name)
	}
}

func (s *Service) StopService(ctx context.Context, name string) error {
	switch strings.TrimSpace(name) {
	case serviceLlama:
		return s.host.Stop(ctx)
	case serviceToolRouter:
		return s.stopProcess(ctx, "tool-router-rs")
	case serviceOpenWebUI:
		return s.stopProcess(ctx, "open-webui")
	case serviceGateway:
		return fmt.Errorf("gateway stop is not supported from the gateway itself")
	case serviceHostAgent:
		return s.stopHostAgent(ctx)
	default:
		return fmt.Errorf("unsupported service: %s", name)
	}
}

func (s *Service) startDetached(ctx context.Context, logName, scriptPath string) error {
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("start script not found: %s", scriptPath)
	}

	logDir := "/mnt/d/AI/Local/Logs/wsl"
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	stdoutPath := filepath.Join(logDir, logName+".out")
	stderrPath := filepath.Join(logDir, logName+".err")
	command := fmt.Sprintf("nohup %s > %s 2> %s < /dev/null &", shellQuote(scriptPath), shellQuote(stdoutPath), shellQuote(stderrPath))
	cmd := exec.CommandContext(ctx, "/bin/bash", "-lc", command)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start %s: %w: %s", logName, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *Service) stopProcess(ctx context.Context, pattern string) error {
	cmd := exec.CommandContext(ctx, "pkill", "-f", pattern)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
		return nil
	}

	return fmt.Errorf("stop process %s: %w: %s", pattern, err, strings.TrimSpace(string(output)))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func (s *Service) startHostAgent(ctx context.Context) error {
	binary := s.cfg.Admin.HostAgentBinary
	cfgPath := s.cfg.Admin.HostAgentConfig
	if binary == "" {
		return fmt.Errorf("host_agent_binary not configured")
	}
	// Use PowerShell via WSL interop to start the Windows process detached.
	psCmd := fmt.Sprintf(
		`Start-Process -FilePath '%s' -ArgumentList '--config','%s' -WindowStyle Hidden`,
		strings.ReplaceAll(binary, `'`, `''`),
		strings.ReplaceAll(cfgPath, `'`, `''`),
	)
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start host-agent: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *Service) stopHostAgent(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "taskkill.exe", "/IM", "host-control-rs.exe", "/F")
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
		return nil // process not found — already stopped
	}
	return fmt.Errorf("stop host-agent: %w: %s", err, strings.TrimSpace(string(output)))
}
