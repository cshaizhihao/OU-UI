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

	"github.com/cshaizhihao/OU-UI/internal/provider"
)

type RuntimeManager struct {
	RuntimeName string
	BinaryNames []string
	ServiceName string
	ConfigExt   string
	TCPHealth   bool
}

func (m RuntimeManager) Install(ctx context.Context, req provider.DeployRequest) (provider.StageResult, error) {
	startedAt := time.Now()
	if err := ctx.Err(); err != nil {
		return StageFailed(provider.DeployStageInstall, err.Error(), startedAt), err
	}
	if _, err := m.lookupBinary(req.Runner); err != nil {
		return StageFailed(provider.DeployStageInstall, fmt.Sprintf("%s binary not found", m.RuntimeName), startedAt), err
	}
	return StageOK(provider.DeployStageInstall, m.RuntimeName+" binary found", startedAt), nil
}

func (m RuntimeManager) ApplyConfig(ctx context.Context, req provider.DeployRequest) (provider.ApplyResult, error) {
	startedAt := time.Now()
	if err := ctx.Err(); err != nil {
		stage := StageFailed(provider.DeployStageApply, err.Error(), startedAt)
		return provider.ApplyResult{StageResult: stage}, err
	}
	ext := m.ConfigExt
	if ext == "" {
		ext = "conf"
	}
	configDir := filepath.Join(req.DataDir, "runtimes", sanitizePath(m.RuntimeName))
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		stage := StageFailed(provider.DeployStageApply, "create runtime config directory failed", startedAt)
		return provider.ApplyResult{StageResult: stage}, err
	}
	configPath := filepath.Join(configDir, sanitizePath(req.NodeID)+"."+ext)
	backupPath := ""
	if existing, err := os.ReadFile(configPath); err == nil && len(existing) > 0 {
		backupDir := filepath.Join(configDir, "backups")
		if err := os.MkdirAll(backupDir, 0o700); err != nil {
			stage := StageFailed(provider.DeployStageApply, "create backup directory failed", startedAt)
			return provider.ApplyResult{StageResult: stage, ConfigPath: configPath}, err
		}
		backupPath = filepath.Join(backupDir, sanitizePath(req.NodeID)+"-"+sanitizePath(req.Revision)+"."+ext)
		if err := os.WriteFile(backupPath, existing, 0o600); err != nil {
			stage := StageFailed(provider.DeployStageApply, "write config backup failed", startedAt)
			return provider.ApplyResult{StageResult: stage, ConfigPath: configPath}, err
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		stage := StageFailed(provider.DeployStageApply, "read existing config failed", startedAt)
		return provider.ApplyResult{StageResult: stage, ConfigPath: configPath}, err
	}
	if err := os.WriteFile(configPath, req.Rendered, 0o600); err != nil {
		stage := StageFailed(provider.DeployStageApply, "write active config failed", startedAt)
		return provider.ApplyResult{StageResult: stage, ConfigPath: configPath, BackupPath: backupPath, RollbackAvailable: backupPath != ""}, err
	}
	stage := StageOK(provider.DeployStageApply, "active config written", startedAt)
	return provider.ApplyResult{
		StageResult: stage,
		ConfigPath: configPath,
		BackupPath: backupPath,
		RollbackAvailable: backupPath != "",
	}, nil
}

func (m RuntimeManager) Reload(ctx context.Context, req provider.DeployRequest) (provider.StageResult, error) {
	startedAt := time.Now()
	if err := ctx.Err(); err != nil {
		return StageFailed(provider.DeployStageReload, err.Error(), startedAt), err
	}
	if req.Runner == nil {
		return StageOK(provider.DeployStageReload, "no command runner available", startedAt), nil
	}
	if _, err := req.Runner.LookPath("systemctl"); err != nil {
		return StageFailed(provider.DeployStageReload, "systemctl not found; reload cannot apply config", startedAt), err
	}
	if result, err := req.Runner.Run(ctx, "systemctl", "reload", m.ServiceName); err == nil {
		return StageOK(provider.DeployStageReload, "systemd reload requested", startedAt), nil
	} else if result.ExitCode == -1 {
		return StageFailed(provider.DeployStageReload, "systemd reload failed", startedAt), err
	}
	if _, err := req.Runner.Run(ctx, "systemctl", "restart", m.ServiceName); err != nil {
		return StageFailed(provider.DeployStageReload, "systemd restart failed", startedAt), err
	}
	return StageOK(provider.DeployStageReload, "systemd restart requested", startedAt), nil
}

func (m RuntimeManager) Health(ctx context.Context, req provider.DeployRequest) (provider.HealthResult, error) {
	startedAt := time.Now()
	version := ""
	if binaryPath, err := m.lookupBinary(req.Runner); err == nil {
		version = m.runtimeVersion(ctx, req.Runner, binaryPath)
	}
	serviceStatus := "unknown"
	systemdAvailable := false
	if req.Runner != nil {
		if _, err := req.Runner.LookPath("systemctl"); err == nil {
			systemdAvailable = true
			if result, runErr := req.Runner.Run(ctx, "systemctl", "is-active", m.ServiceName); runErr == nil && strings.TrimSpace(result.Stdout) != "" {
				serviceStatus = strings.TrimSpace(result.Stdout)
			} else if result.Stdout != "" {
				serviceStatus = strings.TrimSpace(result.Stdout)
			} else if result.Stderr != "" {
				serviceStatus = strings.TrimSpace(result.Stderr)
			}
		}
	}
	ok := true
	message := "runtime health metadata collected"
	listen := normalizeListen(req.Spec.Listen)
	if systemdAvailable && serviceStatus != "active" {
		ok = false
		message = "runtime service is not active: " + serviceStatus
	}
	if ok && m.TCPHealth && req.Spec.Port > 0 {
		if err := checkTCP(ctx, listen, req.Spec.Port); err != nil {
			ok = false
			message = err.Error()
		} else {
			message = "runtime tcp health check passed"
		}
	}
	stage := StageOK(provider.DeployStageHealth, message, startedAt)
	if !ok {
		stage = StageFailed(provider.DeployStageHealth, message, startedAt)
	}
	health := provider.HealthResult{
		StageResult:    stage,
		OK:             ok,
		ServiceName:    m.ServiceName,
		ServiceStatus:  serviceStatus,
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
	if req.BackupPath == "" {
		if req.ConfigPath == "" {
			return StageOK(provider.DeployStageRollback, "no backup available; rollback skipped", startedAt), nil
		}
		if err := os.Remove(req.ConfigPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return StageFailed(provider.DeployStageRollback, "remove new config failed", startedAt), err
		}
		if req.Runner != nil {
			if _, err := req.Runner.LookPath("systemctl"); err == nil {
				_, _ = req.Runner.Run(ctx, "systemctl", "restart", m.ServiceName)
			}
		}
		return StageOK(provider.DeployStageRollback, "new config removed", startedAt), nil
	}
	content, err := os.ReadFile(req.BackupPath)
	if err != nil {
		return StageFailed(provider.DeployStageRollback, "read backup failed", startedAt), err
	}
	if err := os.WriteFile(req.ConfigPath, content, 0o600); err != nil {
		return StageFailed(provider.DeployStageRollback, "restore backup failed", startedAt), err
	}
	if req.Runner != nil {
		if _, err := req.Runner.LookPath("systemctl"); err == nil {
			_, _ = req.Runner.Run(ctx, "systemctl", "restart", m.ServiceName)
		}
	}
	return StageOK(provider.DeployStageRollback, "backup restored", startedAt), nil
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
