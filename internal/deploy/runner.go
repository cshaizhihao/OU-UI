package deploy

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/cshaizhihao/OU-UI/internal/provider"
)

type OSRunner struct {
	Timeout time.Duration
	MaxOutputBytes int
}

func (r OSRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (r OSRunner) Run(ctx context.Context, name string, args ...string) (provider.CommandResult, error) {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	maxOutput := r.MaxOutputBytes
	if maxOutput <= 0 {
		maxOutput = 4096
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	cmd := exec.CommandContext(runCtx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := provider.CommandResult{
		Command:  append([]string{name}, args...),
		ExitCode: exitCode(err),
		Stdout:   trimOutput(stdout.String(), maxOutput),
		Stderr:   trimOutput(stderr.String(), maxOutput),
		Duration: time.Since(startedAt),
	}
	if runCtx.Err() != nil {
		return result, runCtx.Err()
	}
	return result, err
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func trimOutput(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "...[truncated]"
}

func StageOK(name provider.DeployStageName, message string, startedAt time.Time) provider.StageResult {
	now := time.Now().UTC()
	return provider.StageResult{
		Name:       name,
		Status:     "succeeded",
		Message:    message,
		StartedAt:  startedAt.UTC().Format(time.RFC3339),
		FinishedAt: now.Format(time.RFC3339),
		DurationMs: time.Since(startedAt).Milliseconds(),
	}
}

func StageFailed(name provider.DeployStageName, message string, startedAt time.Time) provider.StageResult {
	now := time.Now().UTC()
	return provider.StageResult{
		Name:       name,
		Status:     "failed",
		Message:    message,
		StartedAt:  startedAt.UTC().Format(time.RFC3339),
		FinishedAt: now.Format(time.RFC3339),
		DurationMs: time.Since(startedAt).Milliseconds(),
	}
}
