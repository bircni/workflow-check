package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if !strings.Contains(stdout.String(), "<lock|verify|list|diff>") {
		t.Fatalf("unexpected help output: %q", stdout.String())
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
