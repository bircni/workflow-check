package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bircni/workflow-check/internal/workflowlock"
)

func TestRunRequiresCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr.String(), "usage: workflow-lock") {
		t.Fatalf("expected usage in stderr, got %q", stderr.String())
	}
}

func TestRunHelpIncludesNewCommands(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run(context.Background(), []string{"help"}, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "<lock|verify|list|diff|version>") {
		t.Fatalf("unexpected help output: %q", stdout.String())
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), []string{"version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(stdout.String()) == "" {
		t.Fatalf("expected version output, got %q", stdout.String())
	}
}

func TestRunRejectsUnsupportedFormat(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), []string{"list", "-format", "yaml"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunListJSONWithDefaultHost(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(
		context.Background(),
		[]string{"list", "-workflows", workflowDir, "-default-host", "code.gitea.example.com", "-format", "json"},
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "\"host\": \"code.gitea.example.com\"") {
		t.Fatalf("unexpected json output: %s", stdout.String())
	}
}

func TestRunDiffJSONUsesStableShape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "workflow-lock.yaml"), []byte("version: 1\nentries:\n  - host: github.com\n    owner: actions\n    repo: checkout\n    ref: v4\n    resolved_sha: deadbeefdeadbeefdeadbeefdeadbeefdeadbeef\n    source_kind: action\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	previous := newEngine
	t.Cleanup(func() { newEngine = previous })
	newEngine = func(_ workflowlock.Resolver, host string) workflowlock.Engine {
		return workflowlock.NewEngineWithDefaultHost(fakeResolver{
			results: map[string]string{
				host + "/actions/checkout@v4": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
			},
		}, host)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(
		context.Background(),
		[]string{"diff", "-workflows", workflowDir, "-lockfile", filepath.Join(root, "workflow-lock.yaml"), "-format", "json"},
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "\"missing\": []") {
		t.Fatalf("unexpected json output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "\"drift\":") {
		t.Fatalf("unexpected json output: %s", stdout.String())
	}
}

func TestParseConfigAutoLoadsRepoConfig(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	config := "workflows: custom-workflows\nlockfile: custom-lock.yaml\ndefault_host: code.gitea.example.com\nformat: json\n"
	if err := os.WriteFile(filepath.Join(root, defaultConfigPath), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := parseConfig(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WorkflowDir != "custom-workflows" || cfg.Lockfile != "custom-lock.yaml" || cfg.DefaultHost != "code.gitea.example.com" || cfg.Format != "json" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestParseConfigFlagOverridesRepoConfig(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	config := "workflows: custom-workflows\nlockfile: custom-lock.yaml\ndefault_host: code.gitea.example.com\nformat: json\n"
	if err := os.WriteFile(filepath.Join(root, defaultConfigPath), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := parseConfig([]string{"-workflows", "override-workflows", "-format", "text"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WorkflowDir != "override-workflows" {
		t.Fatalf("expected flag override, got %+v", cfg)
	}
	if cfg.Format != "text" {
		t.Fatalf("expected format override, got %+v", cfg)
	}
	if cfg.DefaultHost != "code.gitea.example.com" {
		t.Fatalf("expected config default host, got %+v", cfg)
	}
}

func TestParseConfigExplicitConfigPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "custom-config.yaml")
	config := "workflows: alt-workflows\nlockfile: alt-lock.yaml\ndefault_host: gitea.example.com\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := parseConfig([]string{"-config", configPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ConfigPath != configPath || cfg.WorkflowDir != "alt-workflows" || cfg.Lockfile != "alt-lock.yaml" || cfg.DefaultHost != "gitea.example.com" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestParseConfigMissingExplicitConfigFails(t *testing.T) {
	t.Parallel()

	_, err := parseConfig([]string{"-config", "/tmp/does-not-exist-workflow-lock.yaml"})
	if err == nil || !strings.Contains(err.Error(), "read config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseConfigMalformedConfigFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "broken.yaml")
	if err := os.WriteFile(configPath, []byte("workflows: [oops"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := parseConfig([]string{"-config", configPath})
	if err == nil || !strings.Contains(err.Error(), "parse config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLockWritesLockfile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	previous := newEngine
	t.Cleanup(func() { newEngine = previous })
	newEngine = func(_ workflowlock.Resolver, host string) workflowlock.Engine {
		return workflowlock.NewEngineWithDefaultHost(fakeResolver{
			results: map[string]string{
				host + "/actions/checkout@v4": strings.Repeat("a", 40),
			},
		}, host)
	}

	var stdout bytes.Buffer
	err := runLock(context.Background(), Config{
		WorkflowDir: workflowDir,
		Lockfile:    filepath.Join(root, "workflow-lock.yaml"),
		DefaultHost: "github.com",
	}, &stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "updated") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, "workflow-lock.yaml")); err != nil {
		t.Fatalf("expected lockfile to exist: %v", err)
	}
}

func TestRunVerifyReportsSuccess(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lockfile := filepath.Join(root, "workflow-lock.yaml")
	if err := workflowlock.WriteLockfile(lockfile, []workflowlock.LockEntry{{
		Host:        "github.com",
		Owner:       "actions",
		Repo:        "checkout",
		Ref:         "v4",
		ResolvedSHA: strings.Repeat("a", 40),
		SourceKind:  workflowlock.SourceKindAction,
	}}); err != nil {
		t.Fatal(err)
	}

	previous := newEngine
	t.Cleanup(func() { newEngine = previous })
	newEngine = func(_ workflowlock.Resolver, host string) workflowlock.Engine {
		return workflowlock.NewEngineWithDefaultHost(fakeResolver{
			results: map[string]string{
				host + "/actions/checkout@v4": strings.Repeat("a", 40),
			},
		}, host)
	}

	var stdout bytes.Buffer
	err := runVerify(context.Background(), Config{
		WorkflowDir: workflowDir,
		Lockfile:    lockfile,
		DefaultHost: "github.com",
	}, &stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "verified") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunDiffTextShowsIssues(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lockfile := filepath.Join(root, "workflow-lock.yaml")
	if err := workflowlock.WriteLockfile(lockfile, []workflowlock.LockEntry{{
		Host:        "github.com",
		Owner:       "actions",
		Repo:        "checkout",
		Ref:         "v4",
		ResolvedSHA: strings.Repeat("b", 40),
		SourceKind:  workflowlock.SourceKindAction,
	}}); err != nil {
		t.Fatal(err)
	}

	previous := newEngine
	t.Cleanup(func() { newEngine = previous })
	newEngine = func(_ workflowlock.Resolver, host string) workflowlock.Engine {
		return workflowlock.NewEngineWithDefaultHost(fakeResolver{
			results: map[string]string{
				host + "/actions/checkout@v4": strings.Repeat("a", 40),
			},
		}, host)
	}

	var stdout bytes.Buffer
	err := runDiff(context.Background(), Config{
		WorkflowDir: workflowDir,
		Lockfile:    lockfile,
		DefaultHost: "github.com",
		Format:      "text",
	}, &stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "drift\t") {
		t.Fatalf("unexpected diff output: %q", stdout.String())
	}
}

func TestRunLockPropagatesEngineError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	previous := newEngine
	t.Cleanup(func() { newEngine = previous })
	newEngine = func(_ workflowlock.Resolver, host string) workflowlock.Engine {
		return workflowlock.NewEngineWithDefaultHost(errorResolver{err: errors.New("boom")}, host)
	}

	err := runLock(context.Background(), Config{
		WorkflowDir: workflowDir,
		Lockfile:    filepath.Join(root, "workflow-lock.yaml"),
		DefaultHost: "github.com",
	}, ioDiscard{})
	if err == nil {
		t.Fatal("expected error")
	}
}

type fakeResolver struct {
	results map[string]string
}

func (f fakeResolver) Resolve(_ context.Context, ref workflowlock.NormalizedRef) (string, error) {
	sha, ok := f.results[ref.Key()]
	if !ok {
		return "", os.ErrNotExist
	}
	return sha, nil
}

type errorResolver struct {
	err error
}

func (e errorResolver) Resolve(context.Context, workflowlock.NormalizedRef) (string, error) {
	return "", e.err
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
