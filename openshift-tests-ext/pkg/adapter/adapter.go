package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	ext "github.com/openshift-eng/openshift-tests-extension/pkg/extension/extensiontests"
	"github.com/openshift-eng/openshift-tests-extension/pkg/util/sets"
)

const (
	// defaultTimeout is the default timeout for ACM Policy tests.
	defaultTimeoutStr = "5m"
	defaultTimeout    = 5 * time.Minute

	// skipExitCode is the scenario convention for ResultSkipped (product not installed, missing KUBECONFIG, etc).
	skipExitCode = 2

	// shutdownGracePeriod is how long we wait after SIGTERM before SIGKILL.
	shutdownGracePeriod = 10 * time.Second

	// maxCapturedOutputBytes bounds captured stdout+stderr.
	maxCapturedOutputBytes = 512 * 1024
)

// BuildExtensionTestSpecsFromScenarios parses a JSON array of ScenarioSpec
// entries and produces one ExtensionTestSpec per entry.
//
// Usage in a binary:
//
//	specs, err := adapter.BuildExtensionTestSpecsFromScenarios(scenarioMetadata, scripts.FS)
func BuildExtensionTestSpecsFromScenarios(metadata []byte, scriptFS fs.FS) (ext.ExtensionTestSpecs, error) {
	var scenarios []ScenarioSpec
	if err := json.Unmarshal(metadata, &scenarios); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scenario metadata: %w", err)
	}

	scriptsDir, err := extractScripts(scriptFS)
	if err != nil {
		return nil, err
	}

	specs := make(ext.ExtensionTestSpecs, 0, len(scenarios))
	for _, sc := range scenarios {
		sc := sc // capture loop variable

		if err := validateScenarioSpec(sc); err != nil {
			return nil, err
		}

		lifecycle := ext.LifecycleBlocking
		if sc.Informing {
			lifecycle = ext.LifecycleInforming
		}

		spec := &ext.ExtensionTestSpec{
			Name:          sc.Name,
			Labels:        sets.New(sc.Tags...),
			CodeLocations: []string{codeLocation(sc)},
			Lifecycle:     lifecycle,
			Resources: ext.Resources{
				Timeout: resolveTimeoutStr(sc.Timeout),
			},
			EnvironmentSelector: ext.EnvironmentSelector{
				Include: sc.IncludeCEL,
				Exclude: sc.ExcludeCEL,
			},
			Timeout: resolveTimeout(sc.Timeout, defaultTimeout),
		}

		// Add tags as metadata
		spec.Tags = make(map[string]string)
		for _, tag := range sc.Tags {
			spec.Tags[tag] = "true"
		}
		spec.Tags["component"] = getComponent(sc.Tags)

		runFn := func(ctx context.Context) *ext.ExtensionTestResult {
			timeout := spec.Timeout
			if timeout <= 0 {
				timeout = defaultTimeout
			}
			return runScenario(ctx, sc, scriptsDir, timeout)
		}
		spec.Run = runFn
		spec.RunParallel = runFn

		specs = append(specs, spec)
	}

	return specs, nil
}

// validateScenarioSpec checks that the scenario has required fields populated.
func validateScenarioSpec(sc ScenarioSpec) error {
	if sc.Name == "" {
		return fmt.Errorf("scenario spec is missing name")
	}
	if sc.Entrypoint == "" {
		return fmt.Errorf("scenario %q must set entrypoint", sc.Name)
	}
	return nil
}

// codeLocation builds a code location string from the scenario's workdir and entrypoint.
func codeLocation(sc ScenarioSpec) string {
	if sc.Entrypoint != "" {
		if sc.WorkDir != "" {
			return sc.WorkDir + " " + sc.Entrypoint
		}
		return sc.Entrypoint
	}
	return ""
}

// getComponent extracts the component name from scenario tags, defaulting to "acm".
func getComponent(tags []string) string {
	for _, tag := range tags {
		if tag == "acm" {
			return "acm"
		}
	}
	return "acm"
}

// resolveTimeout parses a duration string and returns it, falling back to the given duration if parsing fails.
func resolveTimeout(timeoutStr string, fallback time.Duration) time.Duration {
	if timeoutStr == "" {
		return fallback
	}
	d, err := time.ParseDuration(timeoutStr)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

// resolveTimeoutStr validates and returns the timeout string, falling back to the default if invalid.
func resolveTimeoutStr(timeoutStr string) string {
	if timeoutStr == "" {
		return defaultTimeoutStr
	}
	d, err := time.ParseDuration(timeoutStr)
	if err != nil || d <= 0 {
		return defaultTimeoutStr
	}
	return timeoutStr
}

// extractScripts copies scriptFS into a temp directory so scenarios can find
// scripts via INTEROP_SCRIPTS_DIR when the binary is invoked via
// --extension-local-binaries (CWD is not the source tree).
func extractScripts(scriptFS fs.FS) (string, error) {
	if scriptFS == nil {
		return "", nil
	}

	dir, err := os.MkdirTemp("", "ote-scripts-*")
	if err != nil {
		return "", fmt.Errorf("failed to create scripts temp dir: %w", err)
	}

	err = fs.WalkDir(scriptFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == "." {
				return nil
			}
			return os.MkdirAll(filepath.Join(dir, path), 0o755)
		}
		data, err := fs.ReadFile(scriptFS, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0o755)
	})
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("failed to extract scripts: %w", err)
	}
	return dir, nil
}

// runScenario executes a single test scenario by running its entrypoint command with a timeout.
// It returns an ExtensionTestResult indicating pass, fail, skip, or error with captured output.
func runScenario(ctx context.Context, sc ScenarioSpec, scriptsDir string, timeout time.Duration) *ext.ExtensionTestResult {
	result := &ext.ExtensionTestResult{
		Name: sc.Name,
	}

	argv := append([]string{sc.Entrypoint}, sc.Args...)

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out := newBoundedBuffer(maxCapturedOutputBytes)

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = buildEnv(sc.EnvVars, sc.EnvFrom, scriptsDir)
	cmd.Dir = sc.WorkDir
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		result.Result = ext.ResultFailed
		result.Error = fmt.Sprintf("failed to start scenario process: %v", err)
		result.Output = out.String()
		return result
	}

	waitErr := waitWithGracefulCancel(runCtx, cmd, shutdownGracePeriod)
	result.Output = out.String()
	result.Result, result.Error = classifyWaitErr(waitErr, runCtx, timeout)
	return result
}

// classifyWaitErr maps a scenario process wait error to an OTE result.
// Exit code 2 is the skip convention used by the shell scripts.
func classifyWaitErr(waitErr error, runCtx context.Context, timeout time.Duration) (ext.Result, string) {
	if waitErr == nil {
		return ext.ResultPassed, ""
	}
	switch {
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		return ext.ResultFailed, fmt.Sprintf("scenario timed out after %s: %v", timeout, waitErr)
	case errors.Is(runCtx.Err(), context.Canceled):
		return ext.ResultFailed, fmt.Sprintf("scenario was cancelled: %v", waitErr)
	}
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) && exitErr.ExitCode() == skipExitCode {
		return ext.ResultSkipped, fmt.Sprintf("scenario skipped: %v", waitErr)
	}
	return ext.ResultFailed, fmt.Sprintf("scenario exited with error: %v", waitErr)
}

// waitWithGracefulCancel waits for a command to complete or terminates it gracefully on context cancellation.
// It sends SIGTERM first, waits gracePeriod, then sends SIGKILL if the process hasn't exited.
func waitWithGracefulCancel(ctx context.Context, cmd *exec.Cmd, gracePeriod time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		signalProcessGroup(cmd, syscall.SIGTERM)
		select {
		case err := <-done:
			return err
		case <-time.After(gracePeriod):
			signalProcessGroup(cmd, syscall.SIGKILL)
			return <-done
		}
	}
}

// signalProcessGroup sends a signal to the process group of the given command.
func signalProcessGroup(cmd *exec.Cmd, sig syscall.Signal) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, sig)
}

type boundedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	maxBytes  int
	truncated bool
}

// newBoundedBuffer creates a buffer that caps output at maxBytes, discarding oldest data when full.
func newBoundedBuffer(maxBytes int) *boundedBuffer {
	return &boundedBuffer{maxBytes: maxBytes}
}

// Write implements io.Writer, appending data to the buffer and discarding excess from the front.
func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n := len(p)
	b.buf.Write(p)

	if excess := b.buf.Len() - b.maxBytes; excess > 0 {
		b.buf.Next(excess)
		b.truncated = true
	}

	return n, nil
}

// String returns the buffered output, with a truncation notice if data was discarded.
func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.truncated {
		return fmt.Sprintf("... [output truncated to last %d bytes] ...\n%s", b.maxBytes, b.buf.String())
	}
	return b.buf.String()
}

// buildEnv constructs an environment slice by merging os.Environ, envFrom mappings, and override values.
// scriptsDir is written to INTEROP_SCRIPTS_DIR unless a scenario EnvVar already sets it.
func buildEnv(overrides, envFrom map[string]string, scriptsDir string) []string {
	env := make(map[string]string)

	for _, kv := range os.Environ() {
		if k, v, ok := splitEnv(kv); ok {
			env[k] = v
		}
	}

	for dest, src := range envFrom {
		if dest == "" || src == "" {
			continue
		}
		if v, ok := env[src]; ok && v != "" {
			env[dest] = v
		}
	}

	if scriptsDir != "" {
		env["INTEROP_SCRIPTS_DIR"] = scriptsDir
	}

	for k, v := range overrides {
		env[k] = v
	}

	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

// splitEnv splits a KEY=VALUE environment string into its components.
func splitEnv(kv string) (key, value string, ok bool) {
	for i := 0; i < len(kv); i++ {
		if kv[i] == '=' {
			return kv[:i], kv[i+1:], true
		}
	}
	return "", "", false
}
