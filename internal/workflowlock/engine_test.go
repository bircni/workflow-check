package workflowlock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeResolver struct {
	results map[string]string
}

func (f fakeResolver) Resolve(_ context.Context, ref NormalizedRef) (string, error) {
	sha, ok := f.results[ref.Key()]
	if !ok {
		return "", os.ErrNotExist
	}
	return sha, nil
}

func TestNormalizeRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		raw        string
		want       NormalizedRef
		wantIgnore bool
		wantErr    string
	}{
		{
			name: "default github action",
			raw:  "actions/checkout@v4",
			want: NormalizedRef{
				Host:       "github.com",
				Owner:      "actions",
				Repo:       "checkout",
				Ref:        "v4",
				SourceKind: SourceKindAction,
			},
		},
		{
			name: "custom default host action",
			raw:  "actions/checkout@v4",
			want: NormalizedRef{
				Host:       "code.gitea.example.com",
				Owner:      "actions",
				Repo:       "checkout",
				Ref:        "v4",
				SourceKind: SourceKindAction,
			},
		},
		{
			name: "host qualified gitea",
			raw:  "gitea.example.com/acme/build-action@v1",
			want: NormalizedRef{
				Host:       "gitea.example.com",
				Owner:      "acme",
				Repo:       "build-action",
				Ref:        "v1",
				SourceKind: SourceKindAction,
			},
		},
		{
			name: "reusable workflow",
			raw:  "github.com/acme/infra/.github/workflows/release.yml@v2",
			want: NormalizedRef{
				Host:       "github.com",
				Owner:      "acme",
				Repo:       "infra",
				Path:       ".github/workflows/release.yml",
				Ref:        "v2",
				SourceKind: SourceKindReusableWorkflow,
			},
		},
		{
			name:       "local ref ignored",
			raw:        "./.github/actions/lint",
			wantIgnore: true,
		},
		{
			name:    "generic url rejected",
			raw:     "https://github.com/actions/checkout@v4",
			wantErr: "unsupported uses reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var (
				got     NormalizedRef
				ignored bool
				err     error
			)
			if tt.name == "custom default host action" {
				got, ignored, err = NormalizeRefForHost(tt.raw, "code.gitea.example.com")
			} else {
				got, ignored, err = NormalizeRef(tt.raw)
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ignored != tt.wantIgnore {
				t.Fatalf("ignored=%v want %v", ignored, tt.wantIgnore)
			}
			if got != tt.want {
				t.Fatalf("got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestDiscoverRefsForHostUsesConfiguredDefault(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	refs, err := DiscoverRefsForHost(workflowDir, "code.gitea.example.com")
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected one ref, got %d", len(refs))
	}
	if refs[0].Normalized.Host != "code.gitea.example.com" {
		t.Fatalf("unexpected host: %+v", refs[0].Normalized)
	}
}

func TestLockAndVerify(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}

	workflow := `
name: ci
on: push
jobs:
  test:
    steps:
      - uses: actions/checkout@v4
      - uses: actions/cache/save@v4
      - uses: gitea.example.com/acme/build-action@v1
      - uses: ./local-action
  release:
    uses: acme/infra/.github/workflows/release.yml@v2
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}

	lockfile := filepath.Join(root, "workflow-lock.yaml")
	resolver := fakeResolver{
		results: map[string]string{
			"github.com/actions/cache/save@v4":                       strings.Repeat("a", 40),
			"github.com/actions/checkout@v4":                         strings.Repeat("b", 40),
			"github.com/acme/infra/.github/workflows/release.yml@v2": strings.Repeat("c", 40),
			"gitea.example.com/acme/build-action@v1":                 strings.Repeat("d", 40),
		},
	}

	engine := NewEngine(resolver)
	if err := engine.Lock(context.Background(), workflowDir, lockfile); err != nil {
		t.Fatalf("lock failed: %v", err)
	}

	data, err := os.ReadFile(lockfile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "local-action") {
		t.Fatalf("lockfile should ignore local actions: %s", text)
	}
	first := strings.Index(text, "repo: cache")
	second := strings.Index(text, "repo: checkout")
	if first == -1 || second == -1 || first > second {
		t.Fatalf("lockfile order is not deterministic:\n%s", text)
	}

	if err := engine.Verify(context.Background(), workflowDir, lockfile); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestVerifyDetectsDriftAndMissingEntries(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}

	workflow := `
jobs:
  test:
    steps:
      - uses: actions/checkout@v4
      - uses: github.com/actions/setup-go@v5
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yaml"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}

	err := WriteLockfile(filepath.Join(root, "workflow-lock.yaml"), []LockEntry{
		{
			Host:        "github.com",
			Owner:       "actions",
			Repo:        "checkout",
			Ref:         "v4",
			ResolvedSHA: strings.Repeat("a", 40),
			SourceKind:  SourceKindAction,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	engine := NewEngine(fakeResolver{
		results: map[string]string{
			"github.com/actions/checkout@v4": strings.Repeat("b", 40),
			"github.com/actions/setup-go@v5": strings.Repeat("c", 40),
		},
	})

	err = engine.Verify(context.Background(), workflowDir, filepath.Join(root, "workflow-lock.yaml"))
	if err == nil {
		t.Fatal("expected verify error")
	}
	if !strings.Contains(err.Error(), "lock drift") {
		t.Fatalf("expected drift error, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing lock entry") {
		t.Fatalf("expected missing entry error, got %v", err)
	}
}

func TestDiscoverRefsReportsUnsupportedSyntax(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}

	workflow := `
jobs:
  test:
    steps:
      - uses: https://github.com/actions/checkout@v4
`
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := DiscoverRefs(workflowDir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported uses reference") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLockMatchesLargeFixture(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := copyDir(filepath.Join("testdata", "large", "workflows"), workflowDir); err != nil {
		t.Fatal(err)
	}

	lockfile := filepath.Join(root, "workflow-lock.yaml")
	engine := NewEngine(fakeResolver{
		results: map[string]string{
			"github.com/acme/infra/.github/workflows/release.yml@v2": strings.Repeat("c", 40),
			"github.com/actions/cache/save@v4":                       strings.Repeat("a", 40),
			"github.com/actions/checkout@v4":                         strings.Repeat("b", 40),
			"github.com/actions/setup-go@v5":                         strings.Repeat("d", 40),
			"gitea.example.com/acme/build-action@v1":                 strings.Repeat("e", 40),
			"gitea.example.com/acme/qa-action/path/to/action@main":   strings.Repeat("f", 40),
		},
	})

	if err := engine.Lock(context.Background(), workflowDir, lockfile); err != nil {
		t.Fatalf("lock failed: %v", err)
	}

	got, err := os.ReadFile(lockfile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "large", "workflow-lock.golden.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("lockfile mismatch\n--- got ---\n%s\n--- want ---\n%s", string(got), string(want))
	}
}

func TestReportIncludesStaleEntries(t *testing.T) {
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
	if err := WriteLockfile(lockfile, []LockEntry{
		{
			Host:        "github.com",
			Owner:       "actions",
			Repo:        "checkout",
			Ref:         "v4",
			ResolvedSHA: strings.Repeat("a", 40),
			SourceKind:  SourceKindAction,
		},
		{
			Host:        "github.com",
			Owner:       "actions",
			Repo:        "setup-go",
			Ref:         "v5",
			ResolvedSHA: strings.Repeat("b", 40),
			SourceKind:  SourceKindAction,
		},
	}); err != nil {
		t.Fatal(err)
	}

	report, err := NewEngine(fakeResolver{
		results: map[string]string{
			"github.com/actions/checkout@v4": strings.Repeat("a", 40),
		},
	}).Report(context.Background(), workflowDir, lockfile)
	if err != nil {
		t.Fatalf("report failed: %v", err)
	}
	if len(report.Stale) != 1 || report.Stale[0].Repo != "setup-go" {
		t.Fatalf("unexpected stale entries: %+v", report.Stale)
	}
}

func TestLockWithEmptyWorkflowDirectoryWritesEmptyLockfile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}

	lockfile := filepath.Join(root, "workflow-lock.yaml")
	if err := NewEngine(fakeResolver{results: map[string]string{}}).Lock(context.Background(), workflowDir, lockfile); err != nil {
		t.Fatalf("lock failed: %v", err)
	}

	data, err := os.ReadFile(lockfile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "entries: []") {
		t.Fatalf("expected empty entries, got %s", string(data))
	}
}

func TestVerifyRejectsUnsupportedLockfileVersion(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "workflow-lock.yaml"), []byte("version: 99\nentries: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := NewEngine(fakeResolver{results: map[string]string{}}).Verify(context.Background(), workflowDir, filepath.Join(root, "workflow-lock.yaml"))
	if err == nil || !strings.Contains(err.Error(), "unsupported lockfile version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReportRejectsMalformedLockfile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "workflow-lock.yaml"), []byte("version: [oops"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := NewEngine(fakeResolver{results: map[string]string{}}).Report(context.Background(), workflowDir, filepath.Join(root, "workflow-lock.yaml"))
	if err == nil || !strings.Contains(err.Error(), "parse lockfile") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReportPropagatesResolverErrors(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteLockfile(filepath.Join(root, "workflow-lock.yaml"), []LockEntry{{
		Host:        "github.com",
		Owner:       "actions",
		Repo:        "checkout",
		Ref:         "v4",
		ResolvedSHA: strings.Repeat("a", 40),
		SourceKind:  SourceKindAction,
	}}); err != nil {
		t.Fatal(err)
	}

	_, err := NewEngine(erroringResolver{err: errors.New("resolver failed")}).Report(context.Background(), workflowDir, filepath.Join(root, "workflow-lock.yaml"))
	if err == nil || !strings.Contains(err.Error(), "resolver failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type erroringResolver struct {
	err error
}

func (e erroringResolver) Resolve(context.Context, NormalizedRef) (string, error) {
	return "", e.err
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
