package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/FPSZ/Sheathed-Edge/Agent/gateway-go/internal/gateway/pathutil"
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
		return s.gracefulStopHostAgent(ctx)
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
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("tool-router config not found: %s", configPath)
	}
	if projectDir == "" {
		return fmt.Errorf("tool-router project directory is not configured")
	}
	if _, err := os.Stat(projectDir); err != nil {
		return fmt.Errorf("tool-router project directory not found: %s", projectDir)
	}

	toolRouterPort, err := readToolRouterListenPort(configPath)
	if err != nil {
		return err
	}
	toolRouterHealthURL := strings.TrimRight(s.cfg.ToolRouter.BaseURL, "/") + "/healthz"
	portCandidates := buildToolRouterPortCandidates(toolRouterPort, 8001, 8002, 8003, 8004, 8005)
	selectedPort := ""
	selectedHealthURL := ""
	for _, candidate := range portCandidates {
		candidateHealthURL := fmt.Sprintf("http://127.0.0.1:%s/healthz", candidate)
		if candidate == toolRouterPort && isHTTPHealthy(ctx, candidateHealthURL, 1500*time.Millisecond) {
			s.cfg.ToolRouter.BaseURL = fmt.Sprintf("http://127.0.0.1:%s", candidate)
			return nil
		}
		if err := s.clearStaleToolRouterListener(ctx, candidate, candidateHealthURL); err != nil {
			continue
		}
		if s.waitForTCPPortClosed(candidate, 1500*time.Millisecond) {
			selectedPort = candidate
			selectedHealthURL = candidateHealthURL
			break
		}
	}
	if selectedPort == "" {
		return fmt.Errorf("no usable tool-router port available in candidates: %s", strings.Join(portCandidates, ", "))
	}
	if selectedPort != toolRouterPort {
		if err := s.updateToolRouterPort(selectedPort); err != nil {
			return err
		}
		toolRouterPort = selectedPort
		toolRouterHealthURL = selectedHealthURL
	} else {
		toolRouterHealthURL = selectedHealthURL
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

	powershellPath, err := resolveWindowsCommand("powershell.exe", "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe")
	if err != nil {
		return err
	}

	var psCmd string
	if exePath != "" {
		psCmd = fmt.Sprintf(
			`Start-Process -FilePath '%s' -ArgumentList '--config','%s' -WorkingDirectory '%s' -RedirectStandardOutput '%s' -RedirectStandardError '%s' -WindowStyle Hidden`,
			strings.ReplaceAll(normalizeWindowsPath(exePath), `'`, `''`),
			strings.ReplaceAll(normalizeWindowsPath(configPath), `'`, `''`),
			strings.ReplaceAll(normalizeWindowsPath(filepath.Dir(exePath)), `'`, `''`),
			strings.ReplaceAll(normalizeWindowsPath(stdoutPath), `'`, `''`),
			strings.ReplaceAll(normalizeWindowsPath(stderrPath), `'`, `''`),
		)
	} else {
		psCmd = fmt.Sprintf(
			`Start-Process -FilePath 'cargo.exe' -ArgumentList 'run','--','--config','%s' -WorkingDirectory '%s' -RedirectStandardOutput '%s' -RedirectStandardError '%s' -WindowStyle Hidden`,
			strings.ReplaceAll(normalizeWindowsPath(configPath), `'`, `''`),
			strings.ReplaceAll(normalizeWindowsPath(projectDir), `'`, `''`),
			strings.ReplaceAll(normalizeWindowsPath(stdoutPath), `'`, `''`),
			strings.ReplaceAll(normalizeWindowsPath(stderrPath), `'`, `''`),
		)
	}

	cmd := exec.CommandContext(ctx, powershellPath, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start tool-router: %w: %s", err, strings.TrimSpace(string(output)))
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if isHTTPHealthy(ctx, toolRouterHealthURL, 1200*time.Millisecond) {
			s.cfg.ToolRouter.BaseURL = strings.TrimSuffix(toolRouterHealthURL, "/healthz")
			return nil
		}
		time.Sleep(350 * time.Millisecond)
	}
	return nil
}

func buildToolRouterPortCandidates(primary string, fallbacks ...int) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 1+len(fallbacks))
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	add(primary)
	for _, port := range fallbacks {
		add(strconv.Itoa(port))
	}
	return out
}

func (s *Service) updateToolRouterPort(port string) error {
	port = strings.TrimSpace(port)
	if port == "" {
		return fmt.Errorf("tool-router port is required")
	}

	var toolRouterPayload map[string]any
	toolRouterData, err := os.ReadFile(s.toolRouterConfigPath)
	if err != nil {
		return fmt.Errorf("read tool-router config: %w", err)
	}
	if err := json.Unmarshal(toolRouterData, &toolRouterPayload); err != nil {
		return fmt.Errorf("parse tool-router config: %w", err)
	}
	listenPort, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid tool-router port %s: %w", port, err)
	}
	toolRouterPayload["listen_port"] = listenPort
	updatedToolRouterData, err := json.MarshalIndent(toolRouterPayload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode tool-router config: %w", err)
	}
	updatedToolRouterData = append(updatedToolRouterData, '\n')
	if err := os.WriteFile(s.toolRouterConfigPath, updatedToolRouterData, 0o644); err != nil {
		return fmt.Errorf("write tool-router config: %w", err)
	}

	if strings.TrimSpace(s.gatewayConfigPath) != "" {
		var gatewayPayload map[string]any
		gatewayData, err := os.ReadFile(s.gatewayConfigPath)
		if err != nil {
			return fmt.Errorf("read gateway config: %w", err)
		}
		if err := json.Unmarshal(gatewayData, &gatewayPayload); err != nil {
			return fmt.Errorf("parse gateway config: %w", err)
		}
		rawToolRouter, _ := gatewayPayload["tool_router"].(map[string]any)
		if rawToolRouter == nil {
			rawToolRouter = map[string]any{}
		}
		rawToolRouter["base_url"] = fmt.Sprintf("http://127.0.0.1:%s", port)
		gatewayPayload["tool_router"] = rawToolRouter
		updatedGatewayData, err := json.MarshalIndent(gatewayPayload, "", "  ")
		if err != nil {
			return fmt.Errorf("encode gateway config: %w", err)
		}
		updatedGatewayData = append(updatedGatewayData, '\n')
		if err := os.WriteFile(s.gatewayConfigPath, updatedGatewayData, 0o644); err != nil {
			return fmt.Errorf("write gateway config: %w", err)
		}
	}

	s.cfg.ToolRouter.BaseURL = fmt.Sprintf("http://127.0.0.1:%s", port)
	return nil
}

func (s *Service) stopToolRouter(ctx context.Context) error {
	powershellPath, err := resolveWindowsCommand("powershell.exe", "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe")
	if err != nil {
		return err
	}
	toolRouterPort, portErr := readToolRouterListenPort(strings.TrimSpace(s.toolRouterConfigPath))
	if portErr != nil {
		toolRouterPort = strings.TrimSpace(hostAgentPortFromURL(s.cfg.ToolRouter.BaseURL))
	}
	psCmd := `[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; ` +
		`$targets = Get-CimInstance Win32_Process | Where-Object { ` +
		`$_.Name -match 'tool-router-rs(\.exe)?|cargo(\.exe)?' -and ($_.CommandLine -like '*tool-router-rs*' -or $_.CommandLine -like '*tool-router.config.json*') }; ` +
		`foreach ($proc in $targets) { Stop-Process -Id $proc.ProcessId -Force -ErrorAction SilentlyContinue }`
	cmd := exec.CommandContext(ctx, powershellPath, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("stop tool-router: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if toolRouterPort != "" && !s.waitForTCPPortClosed(toolRouterPort, 8*time.Second) {
		return fmt.Errorf("tool-router stop timed out waiting for port %s to close", toolRouterPort)
	}
	return nil
}

func readToolRouterListenPort(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read tool-router config: %w", err)
	}
	var payload struct {
		ListenPort int `json:"listen_port"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("parse tool-router config: %w", err)
	}
	if payload.ListenPort <= 0 {
		return "", fmt.Errorf("tool-router listen_port is invalid in %s", configPath)
	}
	return fmt.Sprintf("%d", payload.ListenPort), nil
}

func isHTTPHealthy(ctx context.Context, rawURL string, timeout time.Duration) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dialer := &net.Dialer{Timeout: timeout}
	client := &httpClientWithTimeout{dialer: dialer, timeout: timeout}
	return client.ok(checkCtx, rawURL)
}

type httpClientWithTimeout struct {
	dialer  *net.Dialer
	timeout time.Duration
}

func (c *httpClientWithTimeout) ok(ctx context.Context, rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	port := parsed.Port()
	if host == "" || port == "" {
		return false
	}
	conn, err := c.dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return false
	}
	_ = conn.SetDeadline(time.Now().Add(c.timeout))
	defer conn.Close()
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", parsed.RequestURI(), parsed.Host)
	if _, err := conn.Write([]byte(req)); err != nil {
		return false
	}
	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil || n <= 0 {
		return false
	}
	return strings.Contains(string(buf[:n]), "200")
}

func (s *Service) clearStaleToolRouterListener(ctx context.Context, port string, healthURL string) error {
	port = strings.TrimSpace(port)
	if port == "" {
		return nil
	}
	if isHTTPHealthy(ctx, healthURL, 1500*time.Millisecond) {
		return nil
	}
	powershellPath, err := resolveWindowsCommand("powershell.exe", "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe")
	if err != nil {
		return err
	}
	psCmd := `[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; ` +
		fmt.Sprintf(`$listener = Get-NetTCPConnection -LocalPort %s -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1; `, port) +
		`if ($listener) { Stop-Process -Id $listener.OwningProcess -Force -ErrorAction SilentlyContinue; Start-Sleep -Milliseconds 900 }; exit 0`
	cmd := exec.CommandContext(ctx, powershellPath, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clear stale tool-router listener: %w: %s", err, strings.TrimSpace(string(output)))
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
		return s.waitForHostAgentShutdownFallback()
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
	binary := strings.TrimSpace(s.cfg.Admin.HostAgentBinary)
	cfgPath := strings.TrimSpace(s.cfg.Admin.HostAgentConfig)
	if binary == "" {
		return fmt.Errorf("host_agent_binary not configured")
	}
	binaryRuntimePath := pathutil.NormalizeRuntimePath(binary)
	if _, err := os.Stat(binaryRuntimePath); err != nil {
		return fmt.Errorf("host-agent binary not found: %s", binary)
	}
	if strings.TrimSpace(cfgPath) == "" {
		return fmt.Errorf("host_agent_config not configured")
	}
	cfgRuntimePath := pathutil.NormalizeRuntimePath(cfgPath)
	if _, err := os.Stat(cfgRuntimePath); err != nil {
		return fmt.Errorf("host-agent config not found: %s", cfgPath)
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
		`}; exit 0`
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

func (s *Service) gracefulStopHostAgent(ctx context.Context) error {
	_ = s.host.Stop(ctx)
	if err := s.host.Shutdown(ctx); err == nil {
		if s.waitForTCPPortClosed(hostAgentPortFromURL(s.cfg.Admin.HostAgentURL), 12*time.Second) {
			return nil
		}
	}

	return s.stopHostAgent(ctx)
}

func (s *Service) waitForHostAgentShutdownFallback() error {
	if s.waitForTCPPortClosed(hostAgentPortFromURL(s.cfg.Admin.HostAgentURL), 12*time.Second) {
		return nil
	}
	return fmt.Errorf("host-agent stop timed out waiting for port release")
}

func (s *Service) waitForTCPPortClosed(port string, timeout time.Duration) bool {
	port = strings.TrimSpace(port)
	if port == "" {
		return true
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), 700*time.Millisecond)
		if err != nil {
			return true
		}
		_ = conn.Close()
		time.Sleep(350 * time.Millisecond)
	}
	return false
}

func hostAgentPortFromURL(raw string) string {
	if parsed, err := url.Parse(strings.TrimSpace(raw)); err == nil {
		if port := strings.TrimSpace(parsed.Port()); port != "" {
			return port
		}
	}
	return "8101"
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
