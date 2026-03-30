package admin

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
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
	case serviceLlama, serviceToolRouter, serviceOpenWebUI:
		return ControlState{
			CanStart: true,
			CanStop:  true,
		}
	case serviceHostAgent:
		return ControlState{
			CanStart:          true,
			UnsupportedReason: "host-agent stop is disabled to avoid stale port locks on Windows",
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
		return s.startToolRouter(ctx)
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
		return s.stopToolRouter(ctx)
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

func (s *Service) startToolRouter(ctx context.Context) error {
	configPath := strings.TrimSpace(s.toolRouterConfigPath)
	projectDir := strings.TrimSpace(s.toolRouterProjectDir)
	if configPath == "" {
		return fmt.Errorf("tool-router config path is not configured")
	}
	if projectDir == "" {
		return fmt.Errorf("tool-router project directory is not configured")
	}

	exePath := firstExistingPath(
		filepath.Join(projectDir, "target", "release", "tool-router-rs.exe"),
		filepath.Join(projectDir, "target", "debug", "tool-router-rs.exe"),
	)
	logDir := filepath.Join(filepath.Dir(filepath.Dir(projectDir)), "Logs", "startup")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("create startup log dir: %w", err)
	}
	stdoutPath := filepath.Join(logDir, "tool-router.out.log")
	stderrPath := filepath.Join(logDir, "tool-router.err.log")

	var command string
	if exePath != "" {
		command = fmt.Sprintf(
			"cd %s && nohup %s --config %s > %s 2> %s < /dev/null &",
			shellQuote(projectDir),
			shellQuote(exePath),
			shellQuote(normalizeWindowsPath(configPath)),
			shellQuote(stdoutPath),
			shellQuote(stderrPath),
		)
	} else {
		command = fmt.Sprintf(
			"cd %s && nohup cargo run -- --config %s > %s 2> %s < /dev/null &",
			shellQuote(projectDir),
			shellQuote(configPath),
			shellQuote(stdoutPath),
			shellQuote(stderrPath),
		)
	}
	cmd := exec.CommandContext(ctx, "/bin/bash", "-lc", command)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start tool-router: %w", err)
	}
	_ = cmd.Process.Release()
	return nil
}

func (s *Service) stopToolRouter(ctx context.Context) error {
	powershellPath, err := resolveWindowsCommand("powershell.exe", "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe")
	if err != nil {
		return err
	}
	psCmd := `[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; ` +
		`$targets = Get-CimInstance Win32_Process | Where-Object { ` +
		`$_.Name -match 'tool-router-rs(\.exe)?|cargo(\.exe)?' -and ($_.CommandLine -like '*tool-router-rs*' -or $_.CommandLine -like '*tool-router.config.json*') }; ` +
		`foreach ($proc in $targets) { Stop-Process -Id $proc.ProcessId -Force -ErrorAction SilentlyContinue }`
	cmd := exec.CommandContext(ctx, powershellPath, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("stop tool-router: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
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

func firstExistingPath(paths ...string) string {
	for _, candidate := range paths {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func (s *Service) startHostAgent(ctx context.Context) error {
	powershellPath, err := resolveWindowsCommand("powershell.exe", "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe")
	if err != nil {
		return err
	}
	binary := s.cfg.Admin.HostAgentBinary
	cfgPath := s.cfg.Admin.HostAgentConfig
	if binary == "" {
		return fmt.Errorf("host_agent_binary not configured")
	}
	binary = normalizeWindowsPath(binary)
	cfgPath = normalizeWindowsPath(cfgPath)
	hostAgentPort := "8101"
	hostAgentHealthURL := "http://127.0.0.1:8101/healthz"
	if parsed, err := url.Parse(strings.TrimSpace(s.cfg.Admin.HostAgentURL)); err == nil {
		if port := parsed.Port(); port != "" {
			hostAgentPort = port
		}
		if strings.TrimSpace(parsed.Scheme) != "" && strings.TrimSpace(parsed.Host) != "" {
			hostAgentHealthURL = strings.TrimRight(parsed.String(), "/") + "/healthz"
		}
	}

	clearStaleCmd := `[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; ` +
		fmt.Sprintf(`$listener = Get-NetTCPConnection -LocalPort %s -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1; `, hostAgentPort) +
		`if ($listener) { ` +
		`  try { ` +
		fmt.Sprintf(`    Invoke-WebRequest -UseBasicParsing -Uri '%s' -TimeoutSec 2 | Out-Null; `, strings.ReplaceAll(hostAgentHealthURL, `'`, `''`)) +
		`  } catch { ` +
		`    Stop-Process -Id $listener.OwningProcess -Force -ErrorAction SilentlyContinue; ` +
		`    Start-Sleep -Milliseconds 800; ` +
		`  } ` +
		`}`
	clearCmd := exec.CommandContext(ctx, powershellPath, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", clearStaleCmd)
	if output, err := clearCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clear stale host-agent listener: %w: %s", err, strings.TrimSpace(string(output)))
	}

	// Use PowerShell via WSL interop to start the Windows process detached.
	psCmd := fmt.Sprintf(
		`Start-Process -FilePath '%s' -ArgumentList '--config','%s' -WindowStyle Hidden`,
		strings.ReplaceAll(binary, `'`, `''`),
		strings.ReplaceAll(cfgPath, `'`, `''`),
	)
	cmd := exec.CommandContext(ctx, powershellPath, "-NoProfile", "-NonInteractive", "-Command", psCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start host-agent: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *Service) stopHostAgent(ctx context.Context) error {
	taskkillPath, err := resolveWindowsCommand("taskkill.exe", "/mnt/c/Windows/System32/taskkill.exe")
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, taskkillPath, "/IM", "host-control-rs.exe", "/F")
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
		return nil // process not found — already stopped
	}
	return fmt.Errorf("stop host-agent: %w: %s", err, strings.TrimSpace(string(output)))
}

func resolveWindowsCommand(command string, fallbacks ...string) (string, error) {
	if trimmed := strings.TrimSpace(command); trimmed != "" {
		if resolved, err := exec.LookPath(trimmed); err == nil {
			return resolved, nil
		}
	}

	for _, fallback := range fallbacks {
		fallback = strings.TrimSpace(fallback)
		if fallback == "" {
			continue
		}
		if _, err := os.Stat(fallback); err == nil {
			return fallback, nil
		}
	}

	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("windows command is not configured")
	}
	return "", fmt.Errorf("unable to resolve windows command: %s", command)
}

func normalizeWindowsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "/mnt/") && len(path) >= 7 && unicode.IsLetter(rune(path[5])) && path[6] == '/' {
		drive := unicode.ToUpper(rune(path[5]))
		rest := strings.ReplaceAll(path[7:], "/", `\`)
		return fmt.Sprintf("%c:\\%s", drive, rest)
	}

	return path
}
