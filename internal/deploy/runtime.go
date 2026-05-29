package deploy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/cshaizhihao/OU-UI/internal/provider"
)

const (
	ServiceModeManagedConfDir = "managed-confdir"
	ServiceModeManagedNode    = "managed-node"
)

type RuntimeManager struct {
	RuntimeName  string
	BinaryNames  []string
	ServiceName  string
	ServicePrefix string
	ConfigExt    string
	TCPHealth    bool
	ServiceMode  string
	CommandArgs  []string
}

type runtimeLayout struct {
	ConfigDir   string
	ConfigPath  string
	BackupPath  string
	ServiceName string
	UnitPath    string
	ServiceMode string
}

func (m RuntimeManager) Install(ctx context.Context, req provider.DeployRequest) (provider.StageResult, error) {
	startedAt := time.Now()
	if err := ctx.Err(); err != nil {
		return StageFailed(provider.DeployStageInstall, err.Error(), startedAt), err
	}
	if _, err := m.lookupBinary(req.Runner); err != nil {
		return StageFailed(provider.DeployStageInstall, fmt.Sprintf("%s binary not found", m.RuntimeName), startedAt), err
	}
	if req.Runner == nil {
		err := errors.New("command runner is required")
		return StageFailed(provider.DeployStageInstall, err.Error(), startedAt), err
	}
	if _, err := req.Runner.LookPath("systemctl"); err != nil {
		return StageFailed(provider.DeployStageInstall, "systemctl not found; managed runtime service cannot be installed", startedAt), err
	}
	return StageOK(provider.DeployStageInstall, m.RuntimeName+" binary and systemctl found", startedAt), nil
}

func (m RuntimeManager) ApplyConfig(ctx context.Context, req provider.DeployRequest) (provider.ApplyResult, error) {
	startedAt := time.Now()
	if err := ctx.Err(); err != nil {
		stage := StageFailed(provider.DeployStageApply, err.Error(), startedAt)
		return provider.ApplyResult{StageResult: stage}, err
	}
	binaryPath, err := m.lookupBinary(req.Runner)
	if err != nil {
		stage := StageFailed(provider.DeployStageApply, "runtime binary not found", startedAt)
		return provider.ApplyResult{StageResult: stage}, err
	}
	layout := m.layout(req, "")
	if err := os.MkdirAll(layout.ConfigDir, 0o700); err != nil {
		stage := StageFailed(provider.DeployStageApply, "create runtime config directory failed", startedAt)
		return m.applyFailure(stage, layout, "", err), err
	}

	backupPath := ""
	if existing, err := os.ReadFile(layout.ConfigPath); err == nil && len(existing) > 0 {
		backupDir := filepath.Join(layout.ConfigDir, "backups")
		if err := os.MkdirAll(backupDir, 0o700); err != nil {
			stage := StageFailed(provider.DeployStageApply, "create backup directory failed", startedAt)
			return m.applyFailure(stage, layout, "", err), err
		}
		backupPath = filepath.Join(backupDir, sanitizePath(req.NodeID)+"-"+sanitizePath(req.Revision)+"."+m.configExt())
		if err := os.WriteFile(backupPath, existing, 0o600); err != nil {
			stage := StageFailed(provider.DeployStageApply, "write config backup failed", startedAt)
			return m.applyFailure(stage, layout, "", err), err
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		stage := StageFailed(provider.DeployStageApply, "read existing config failed", startedAt)
		return m.applyFailure(stage, layout, backupPath, err), err
	}
	layout.BackupPath = backupPath

	if err := writeFileAtomic(layout.ConfigPath, req.Rendered, 0o600); err != nil {
		stage := StageFailed(provider.DeployStageApply, "write active config failed", startedAt)
		return m.applyFailure(stage, layout, backupPath, err), err
	}
	unit, err := m.renderUnit(binaryPath, layout)
	if err != nil {
		stage := StageFailed(provider.DeployStageApply, "render managed service unit failed", startedAt)
		return m.applyFailure(stage, layout, backupPath, err), err
	}
	if err := writeFileAtomic(layout.UnitPath, []byte(unit), 0o644); err != nil {
		stage := StageFailed(provider.DeployStageApply, "write managed service unit failed", startedAt)
		return m.applyFailure(stage, layout, backupPath, err), err
	}
	if result, err := req.Runner.Run(ctx, "systemctl", "daemon-reload"); err != nil {
		stage := StageFailed(provider.DeployStageApply, daemonReloadMessage(result), startedAt)
		return m.applyFailure(stage, layout, backupPath, err), err
	}

	stage := StageOK(provider.DeployStageApply, "active config and managed service unit written", startedAt)
	return provider.ApplyResult{
		StageResult:       stage,
		ConfigPath:        layout.ConfigPath,
		ConfigDir:         layout.ConfigDir,
		BackupPath:        backupPath,
		UnitPath:          layout.UnitPath,
		ServiceName:       layout.ServiceName,
		ServiceMode:       layout.ServiceMode,
		ManagedByOUUI:     true,
		RollbackAvailable: backupPath != "",
	}, nil
}

func (m RuntimeManager) Reload(ctx context.Context, req provider.DeployRequest) (provider.StageResult, error) {
	startedAt := time.Now()
	if err := ctx.Err(); err != nil {
		return StageFailed(provider.DeployStageReload, err.Error(), startedAt), err
	}
	if req.Runner == nil {
		err := errors.New("command runner is required")
		return StageFailed(provider.DeployStageReload, err.Error(), startedAt), err
	}
	if _, err := req.Runner.LookPath("systemctl"); err != nil {
		return StageFailed(provider.DeployStageReload, "systemctl not found; reload cannot apply config", startedAt), err
	}
	layout := m.layout(req, "")
	if result, err := req.Runner.Run(ctx, "systemctl", "daemon-reload"); err != nil {
		return StageFailed(provider.DeployStageReload, daemonReloadMessage(result), startedAt), err
	}
	if _, err := req.Runner.Run(ctx, "systemctl", "enable", "--now", layout.ServiceName); err != nil {
		return StageFailed(provider.DeployStageReload, "enable managed runtime service failed", startedAt), err
	}
	if _, err := req.Runner.Run(ctx, "systemctl", "restart", layout.ServiceName); err != nil {
		return StageFailed(provider.DeployStageReload, "restart managed runtime service failed", startedAt), err
	}
	return StageOK(provider.DeployStageReload, "managed runtime service restarted", startedAt), nil
}

func (m RuntimeManager) Health(ctx context.Context, req provider.DeployRequest) (provider.HealthResult, error) {
	startedAt := time.Now()
	layout := m.layout(req, "")
	version := ""
	if binaryPath, err := m.lookupBinary(req.Runner); err == nil {
		version = m.runtimeVersion(ctx, req.Runner, binaryPath)
	}
	serviceStatus := "unknown"
	systemdAvailable := false
	if req.Runner != nil {
		if _, err := req.Runner.LookPath("systemctl"); err == nil {
			systemdAvailable = true
			if result, runErr := req.Runner.Run(ctx, "systemctl", "is-active", layout.ServiceName); runErr == nil && strings.TrimSpace(result.Stdout) != "" {
				serviceStatus = strings.TrimSpace(result.Stdout)
			} else if strings.TrimSpace(result.Stdout) != "" {
				serviceStatus = strings.TrimSpace(result.Stdout)
			} else if strings.TrimSpace(result.Stderr) != "" {
				serviceStatus = strings.TrimSpace(result.Stderr)
			}
		}
	}
	ok := true
	message := "managed runtime health metadata collected"
	listen := normalizeListen(req.Spec.Listen)
	if systemdAvailable && serviceStatus != "active" {
		ok = false
		message = "managed runtime service is not active: " + serviceStatus
	}
	if ok && m.TCPHealth && req.Spec.Port > 0 {
		if err := checkTCP(ctx, listen, req.Spec.Port); err != nil {
			ok = false
			message = err.Error()
		} else {
			message = "managed runtime tcp health check passed"
		}
	}
	stage := StageOK(provider.DeployStageHealth, message, startedAt)
	if !ok {
		stage = StageFailed(provider.DeployStageHealth, message, startedAt)
	}
	health := provider.HealthResult{
		StageResult:    stage,
		OK:             ok,
		ServiceName:    layout.ServiceName,
		ServiceStatus:  serviceStatus,
		ConfigDir:      layout.ConfigDir,
		UnitPath:       layout.UnitPath,
		ServiceMode:    layout.ServiceMode,
		ManagedByOUUI:  true,
		Runtime:        m.RuntimeName,
		RuntimeVersion: version,
		Listen:         listen,
		Port:           req.Spec.Port,
	}
	if !ok {
		return health, errors.New(message)
	}
	return health, nil
}

func (m RuntimeManager) Rollback(ctx context.Context, req provider.RollbackRequest) (provider.StageResult, error) {
	startedAt := time.Now()
	layout := m.layout(provider.DeployRequest{
		NodeID:   req.NodeID,
		Spec:     req.Spec,
		DataDir:  req.DataDir,
		Revision: req.Revision,
		Runner:   req.Runner,
	}, req.BackupPath)
	if req.ConfigPath != "" {
		layout.ConfigPath = req.ConfigPath
	}
	if req.ConfigDir != "" {
		layout.ConfigDir = req.ConfigDir
	}
	if req.ServiceName != "" {
		layout.ServiceName = req.ServiceName
	}
	if req.UnitPath != "" {
		layout.UnitPath = req.UnitPath
	}

	if req.BackupPath == "" {
		if layout.ConfigPath != "" {
			if err := os.Remove(layout.ConfigPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return StageFailed(provider.DeployStageRollback, "remove new config failed", startedAt), err
			}
		}
		if m.serviceMode() == ServiceModeManagedNode {
			if _, err := os.Stat(layout.UnitPath); err == nil {
				if err := m.stopDisableService(ctx, req.Runner, layout.ServiceName); err != nil {
					return StageFailed(provider.DeployStageRollback, "stop managed service failed", startedAt), err
				}
				if err := os.Remove(layout.UnitPath); err != nil && !errors.Is(err, os.ErrNotExist) {
					return StageFailed(provider.DeployStageRollback, "remove new managed service unit failed", startedAt), err
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return StageFailed(provider.DeployStageRollback, "inspect managed service unit failed", startedAt), err
			}
			if err := m.daemonReload(ctx, req.Runner); err != nil {
				return StageFailed(provider.DeployStageRollback, "systemd daemon-reload failed after rollback", startedAt), err
			}
			return StageOK(provider.DeployStageRollback, "new config and managed service unit removed", startedAt), nil
		}
		if err := m.restartService(ctx, req.Runner, layout.ServiceName); err != nil {
			return StageFailed(provider.DeployStageRollback, "restart managed service failed after config removal", startedAt), err
		}
		return StageOK(provider.DeployStageRollback, "new config removed and managed service restarted", startedAt), nil
	}

	content, err := os.ReadFile(req.BackupPath)
	if err != nil {
		return StageFailed(provider.DeployStageRollback, "read backup failed", startedAt), err
	}
	if err := writeFileAtomic(layout.ConfigPath, content, 0o600); err != nil {
		return StageFailed(provider.DeployStageRollback, "restore backup failed", startedAt), err
	}
	if err := m.restartService(ctx, req.Runner, layout.ServiceName); err != nil {
		return StageFailed(provider.DeployStageRollback, "restart managed service failed after backup restore", startedAt), err
	}
	return StageOK(provider.DeployStageRollback, "backup restored and managed service restarted", startedAt), nil
}

func (m RuntimeManager) applyFailure(stage provider.StageResult, layout runtimeLayout, backupPath string, err error) provider.ApplyResult {
	return provider.ApplyResult{
		StageResult:       stage,
		ConfigPath:        layout.ConfigPath,
		ConfigDir:         layout.ConfigDir,
		BackupPath:        backupPath,
		UnitPath:          layout.UnitPath,
		ServiceName:       layout.ServiceName,
		ServiceMode:       layout.ServiceMode,
		ManagedByOUUI:     true,
		RollbackAvailable: backupPath != "",
	}
}

func (m RuntimeManager) renderUnit(binaryPath string, layout runtimeLayout) (string, error) {
	args := m.commandArgs(layout)
	if binaryPath == "" || len(args) == 0 {
		return "", errors.New("runtime unit command is empty")
	}
	command := systemdExecStart(append([]string{binaryPath}, args...))
	return fmt.Sprintf(`[Unit]
Description=OU-UI managed %s runtime (%s)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=3
LimitNOFILE=1048576
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=read-only
ProtectSystem=full
ReadOnlyPaths=%s
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
`, m.RuntimeName, layout.ServiceName, command, systemdQuote(layout.ConfigDir)), nil
}

func (m RuntimeManager) commandArgs(layout runtimeLayout) []string {
	args := m.CommandArgs
	if len(args) == 0 {
		args = []string{"run", "-config", "{configPath}"}
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.ReplaceAll(arg, "{configPath}", layout.ConfigPath)
		arg = strings.ReplaceAll(arg, "{configDir}", layout.ConfigDir)
		out = append(out, arg)
	}
	return out
}

func (m RuntimeManager) layout(req provider.DeployRequest, backupPath string) runtimeLayout {
	configDir := filepath.Join(req.DataDir, "runtimes", sanitizePath(m.RuntimeName), "active")
	configPath := filepath.Join(configDir, sanitizePath(req.NodeID)+"."+m.configExt())
	serviceName := m.serviceName(req.NodeID)
	return runtimeLayout{
		ConfigDir:   configDir,
		ConfigPath:  configPath,
		BackupPath:  backupPath,
		ServiceName: serviceName,
		UnitPath:    filepath.Join("/etc/systemd/system", serviceName+".service"),
		ServiceMode: m.serviceMode(),
	}
}

func (m RuntimeManager) configExt() string {
	if m.ConfigExt == "" {
		return "conf"
	}
	return m.ConfigExt
}

func (m RuntimeManager) serviceMode() string {
	if m.ServiceMode == "" {
		return ServiceModeManagedNode
	}
	return m.ServiceMode
}

func (m RuntimeManager) serviceName(nodeID string) string {
	name := strings.TrimSpace(m.ServiceName)
	if name == "" {
		prefix := strings.TrimSpace(m.ServicePrefix)
		if prefix == "" {
			prefix = "ou-ui-" + sanitizeSystemdName(m.RuntimeName)
		}
		name = prefix
		if m.serviceMode() == ServiceModeManagedNode {
			name += "-" + sanitizeSystemdName(nodeID)
		}
	}
	return sanitizeSystemdName(name)
}

func (m RuntimeManager) restartService(ctx context.Context, runner provider.CommandRunner, serviceName string) error {
	if runner == nil {
		return errors.New("command runner is required")
	}
	if _, err := runner.LookPath("systemctl"); err != nil {
		return err
	}
	_, err := runner.Run(ctx, "systemctl", "restart", serviceName)
	return err
}

func (m RuntimeManager) stopDisableService(ctx context.Context, runner provider.CommandRunner, serviceName string) error {
	if runner == nil {
		return errors.New("command runner is required")
	}
	if _, err := runner.LookPath("systemctl"); err != nil {
		return err
	}
	_, err := runner.Run(ctx, "systemctl", "disable", "--now", serviceName)
	return err
}

func (m RuntimeManager) daemonReload(ctx context.Context, runner provider.CommandRunner) error {
	if runner == nil {
		return errors.New("command runner is required")
	}
	if _, err := runner.LookPath("systemctl"); err != nil {
		return err
	}
	_, err := runner.Run(ctx, "systemctl", "daemon-reload")
	return err
}

func (m RuntimeManager) lookupBinary(runner provider.CommandRunner) (string, error) {
	if runner == nil {
		return "", errors.New("command runner is required")
	}
	for _, name := range m.BinaryNames {
		if path, err := runner.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s binary not found in PATH", m.RuntimeName)
}

func (m RuntimeManager) runtimeVersion(ctx context.Context, runner provider.CommandRunner, binaryPath string) string {
	if runner == nil || binaryPath == "" {
		return ""
	}
	for _, args := range [][]string{{"version"}, {"--version"}, {"-version"}} {
		result, err := runner.Run(ctx, binaryPath, args...)
		if err == nil && strings.TrimSpace(result.Stdout) != "" {
			return firstLine(result.Stdout)
		}
		if strings.TrimSpace(result.Stderr) != "" {
			return firstLine(result.Stderr)
		}
	}
	return ""
}

func normalizeListen(listen string) string {
	listen = strings.TrimSpace(listen)
	switch listen {
	case "", "0.0.0.0", "::":
		return "127.0.0.1"
	default:
		return listen
	}
}

func checkTCP(ctx context.Context, host string, port int) error {
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprint(port)))
	if err != nil {
		return fmt.Errorf("tcp health check failed on %s:%d", host, port)
	}
	_ = conn.Close()
	return nil
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	return value
}

func sanitizePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_", "..", "_")
	return replacer.Replace(value)
}

func writeFileAtomic(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".ou-ui-*")
	if err != nil {
		return err
	}
	tmp := file.Name()
	defer func() { _ = os.Remove(tmp) }()
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(perm); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func sanitizeSystemdName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "ou-ui-runtime"
	}
	var builder strings.Builder
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '-', r == '_', r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteByte('-')
		}
	}
	return strings.Trim(builder.String(), "-.")
}

func systemdExecStart(args []string) string {
	escaped := make([]string, 0, len(args))
	for _, arg := range args {
		escaped = append(escaped, systemdQuote(arg))
	}
	return strings.Join(escaped, " ")
}

func systemdQuote(value string) string {
	value = strings.ReplaceAll(value, "%", "%%")
	if value == "" || strings.ContainsAny(value, " \t\n\"'\\") {
		value = strings.ReplaceAll(value, "\\", "\\\\")
		value = strings.ReplaceAll(value, "\"", "\\\"")
		return `"` + value + `"`
	}
	return value
}

func daemonReloadMessage(result provider.CommandResult) string {
	if strings.TrimSpace(result.Stderr) != "" {
		return "systemd daemon-reload failed: " + firstLine(result.Stderr)
	}
	return "systemd daemon-reload failed"
}
