package tuning

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/deploy"
	"github.com/cshaizhihao/OU-UI/internal/provider"
)

const CapabilityHostOptimize = "host.optimize"

type Request struct {
	Profile            string `json:"profile"`
	AllowKernelInstall bool   `json:"allowKernelInstall"`
	RebootPolicy       string `json:"rebootPolicy"`
	Persist            bool   `json:"persist"`
}

type Result struct {
	Profile                       string                 `json:"profile"`
	Kernel                        string                 `json:"kernel"`
	CongestionControl             string                 `json:"congestionControl"`
	AvailableCongestionControls   []string               `json:"availableCongestionControls"`
	BBRAvailable                  bool                   `json:"bbrAvailable"`
	BBRV3Likely                   bool                   `json:"bbrV3Likely"`
	SysctlFile                    string                 `json:"sysctlFile"`
	KernelInstallAttempted        bool                   `json:"kernelInstallAttempted"`
	KernelInstallStatus           string                 `json:"kernelInstallStatus"`
	KernelInstallMessage          string                 `json:"kernelInstallMessage,omitempty"`
	RebootRequired                bool                   `json:"rebootRequired"`
	AppliedSysctl                 map[string]string      `json:"appliedSysctl"`
	Stages                        []provider.StageResult `json:"stages"`
}

type Optimizer struct {
	Runner provider.CommandRunner
	DataDir string
	Now    func() time.Time
}

func (o Optimizer) Optimize(ctx context.Context, req Request) (Result, error) {
	if req.Profile == "" {
		req.Profile = "bbr-v3"
	}
	result := Result{
		Profile:             req.Profile,
		KernelInstallStatus: "skipped",
		AppliedSysctl:       profileSysctl(req.Profile),
	}
	appendStage := func(stage provider.StageResult) {
		result.Stages = append(result.Stages, stage)
	}
	if o.Runner == nil {
		err := errors.New("command runner is required")
		appendStage(stageFailed("detect", err.Error(), o.now()))
		return result, err
	}

	detectStarted := o.now()
	if _, err := o.Runner.LookPath("sysctl"); err != nil {
		appendStage(stageFailed("detect", "sysctl not found", detectStarted))
		return result, err
	}
	kernel, _ := o.output(ctx, "uname", "-r")
	result.Kernel = kernel
	current, _ := o.output(ctx, "sysctl", "-n", "net.ipv4.tcp_congestion_control")
	result.CongestionControl = current
	available, _ := o.output(ctx, "sysctl", "-n", "net.ipv4.tcp_available_congestion_control")
	result.AvailableCongestionControls = strings.Fields(available)
	result.BBRAvailable = contains(result.AvailableCongestionControls, "bbr")
	result.BBRV3Likely = bbrV3Likely(kernel)
	appendStage(stageOK("detect", "network tuning state collected", detectStarted))

	if !result.BBRAvailable && req.AllowKernelInstall {
		installStarted := o.now()
		result.KernelInstallAttempted = true
		if err := o.installKernel(ctx); err != nil {
			result.KernelInstallStatus = "failed"
			result.KernelInstallMessage = err.Error()
			result.RebootRequired = true
			appendStage(stageFailed("bbr-install", "kernel install attempt failed", installStarted))
		} else {
			result.KernelInstallStatus = "installed"
			result.KernelInstallMessage = "kernel package install command completed"
			result.RebootRequired = true
			appendStage(stageOK("bbr-install", "kernel package install command completed", installStarted))
		}
	} else if !result.BBRAvailable {
		result.KernelInstallMessage = "bbr is not available; enable allowKernelInstall to attempt a BBR v3 capable kernel"
		result.RebootRequired = true
	}

	persistStarted := o.now()
	sysctlFile, err := o.writeSysctlProfile(result.AppliedSysctl)
	if err != nil {
		appendStage(stageFailed("sysctl-persist", "write sysctl profile failed", persistStarted))
		return result, err
	}
	result.SysctlFile = sysctlFile
	appendStage(stageOK("sysctl-persist", "sysctl profile written", persistStarted))

	applyStarted := o.now()
	for key, value := range result.AppliedSysctl {
		if _, err := o.Runner.Run(ctx, "sysctl", "-w", key+"="+value); err != nil {
			appendStage(stageFailed("sysctl-apply", "apply sysctl failed: "+key, applyStarted))
			return result, err
		}
	}
	appendStage(stageOK("sysctl-apply", "sysctl profile applied", applyStarted))

	verifyStarted := o.now()
	if current, err := o.output(ctx, "sysctl", "-n", "net.ipv4.tcp_congestion_control"); err == nil {
		result.CongestionControl = current
	}
	if available, err := o.output(ctx, "sysctl", "-n", "net.ipv4.tcp_available_congestion_control"); err == nil {
		result.AvailableCongestionControls = strings.Fields(available)
		result.BBRAvailable = contains(result.AvailableCongestionControls, "bbr")
	}
	if result.CongestionControl != "bbr" {
		result.RebootRequired = true
		appendStage(stageFailed("verify", "bbr is not active yet", verifyStarted))
		return result, errors.New("bbr is not active after tuning")
	}
	appendStage(stageOK("verify", "bbr network tuning verified", verifyStarted))
	return result, nil
}

func (o Optimizer) output(ctx context.Context, name string, args ...string) (string, error) {
	result, err := o.Runner.Run(ctx, name, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (o Optimizer) writeSysctlProfile(values map[string]string) (string, error) {
	path := "/etc/sysctl.d/99-ou-ui-network.conf"
	var builder strings.Builder
	builder.WriteString("# Managed by OU-UI host.optimize\n")
	for _, key := range sysctlKeys(values) {
		builder.WriteString(key)
		builder.WriteString(" = ")
		builder.WriteString(values[key])
		builder.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ou-ui-sysctl-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.WriteString(builder.String()); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	return path, os.Rename(tmpName, path)
}

func (o Optimizer) installKernel(ctx context.Context) error {
	if _, err := o.Runner.LookPath("bash"); err != nil {
		return err
	}
	script := strings.Join([]string{
		"set -Eeuo pipefail",
		"if command -v apt-get >/dev/null 2>&1; then",
		"  export DEBIAN_FRONTEND=noninteractive",
		"  apt-get update",
		"  apt-get install -y curl gnupg ca-certificates",
		"  install -d -m 0755 /etc/apt/keyrings",
		"  curl -fsSL https://dl.xanmod.org/archive.key | gpg --dearmor -o /etc/apt/keyrings/xanmod-archive-keyring.gpg",
		"  echo 'deb [signed-by=/etc/apt/keyrings/xanmod-archive-keyring.gpg] http://deb.xanmod.org releases main' >/etc/apt/sources.list.d/xanmod-release.list",
		"  apt-get update",
		"  apt-get install -y linux-xanmod-x64v3",
		"elif command -v dnf >/dev/null 2>&1; then",
		"  dnf install -y kernel kernel-core kernel-modules",
		"elif command -v yum >/dev/null 2>&1; then",
		"  yum install -y kernel kernel-core kernel-modules",
		"else",
		"  echo 'unsupported package manager for automatic kernel install' >&2",
		"  exit 42",
		"fi",
	}, "\n")
	result, err := o.Runner.Run(ctx, "bash", "-lc", script)
	if err != nil {
		message := strings.TrimSpace(result.Stderr)
		if message == "" {
			message = err.Error()
		}
		return errors.New(firstLine(message))
	}
	return nil
}

func (o Optimizer) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func profileSysctl(profile string) map[string]string {
	values := map[string]string{
		"net.core.default_qdisc":              "fq",
		"net.ipv4.tcp_congestion_control":     "bbr",
		"net.ipv4.tcp_fastopen":               "3",
		"net.ipv4.tcp_slow_start_after_idle":  "0",
		"net.ipv4.tcp_mtu_probing":            "1",
		"net.ipv4.tcp_notsent_lowat":          "16384",
		"net.ipv4.tcp_tw_reuse":               "1",
		"net.ipv4.ip_local_port_range":        "1024 65535",
		"net.ipv4.tcp_fin_timeout":            "15",
		"net.ipv4.tcp_keepalive_time":         "600",
		"net.ipv4.tcp_keepalive_intvl":        "30",
		"net.ipv4.tcp_keepalive_probes":       "5",
	}
	if profile == "conservative" {
		delete(values, "net.ipv4.tcp_tw_reuse")
		values["net.ipv4.tcp_fin_timeout"] = "30"
	}
	return values
}

func sysctlKeys(values map[string]string) []string {
	keys := []string{
		"net.core.default_qdisc",
		"net.ipv4.tcp_congestion_control",
		"net.ipv4.tcp_fastopen",
		"net.ipv4.tcp_slow_start_after_idle",
		"net.ipv4.tcp_mtu_probing",
		"net.ipv4.tcp_notsent_lowat",
		"net.ipv4.tcp_tw_reuse",
		"net.ipv4.ip_local_port_range",
		"net.ipv4.tcp_fin_timeout",
		"net.ipv4.tcp_keepalive_time",
		"net.ipv4.tcp_keepalive_intvl",
		"net.ipv4.tcp_keepalive_probes",
	}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, ok := values[key]; ok {
			out = append(out, key)
		}
	}
	return out
}

func bbrV3Likely(kernel string) bool {
	kernel = strings.ToLower(strings.TrimSpace(kernel))
	if strings.Contains(kernel, "xanmod") || strings.Contains(kernel, "bbr3") {
		return true
	}
	fields := strings.FieldsFunc(kernel, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	if len(fields) < 2 {
		return false
	}
	major, _ := strconv.Atoi(fields[0])
	minor, _ := strconv.Atoi(fields[1])
	return major > 6 || major == 6 && minor >= 7
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), want) {
			return true
		}
	}
	return false
}

func stageOK(name string, message string, startedAt time.Time) provider.StageResult {
	return deploy.StageOK(provider.DeployStageName(name), message, startedAt)
}

func stageFailed(name string, message string, startedAt time.Time) provider.StageResult {
	return deploy.StageFailed(provider.DeployStageName(name), message, startedAt)
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	return value
}
