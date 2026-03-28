package workflowlock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLockfileRejectsDuplicateKeys(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "workflow-lock.yaml")
	// Two entries that normalize to the same Key() (identical coordinates).
	content := `version: 1
entries:
  - host: github.com
    owner: actions
    repo: checkout
    ref: v4
    resolved_sha: "1111111111111111111111111111111111111111"
    source_kind: action
  - host: github.com
    owner: actions
    repo: checkout
    ref: v4
    resolved_sha: "2222222222222222222222222222222222222222"
    source_kind: action
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadLockfile(path)
	if err == nil {
		t.Fatal("expected error for duplicate lockfile keys")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate in error, got %v", err)
	}
}

func TestWriteLockfileReplacesExistingFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "workflow-lock.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nentries: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []LockEntry{{
		Host:        "github.com",
		Owner:       "actions",
		Repo:        "checkout",
		Ref:         "v4",
		ResolvedSHA: strings.Repeat("a", 40),
		SourceKind:  SourceKindAction,
	}}
	if err := WriteLockfile(path, entries); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	lf, err := ReadLockfile(path)
	if err != nil {
		t.Fatalf("ReadLockfile: %v", err)
	}
	if len(lf.Entries) != 1 {
		t.Fatalf("entries: got %d want 1", len(lf.Entries))
	}
}

func TestWriteLockfileDoesNotLeaveTempFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "workflow-lock.yaml")
	entries := []LockEntry{{
		Host:        "github.com",
		Owner:       "actions",
		Repo:        "checkout",
		Ref:         "v4",
		ResolvedSHA: strings.Repeat("b", 40),
		SourceKind:  SourceKindAction,
	}}
	if err := WriteLockfile(path, entries); err != nil {
		t.Fatalf("WriteLockfile: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(root, "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range matches {
		base := filepath.Base(m)
		if strings.Contains(base, ".tmp") || strings.HasPrefix(base, ".workflow-lock") {
			t.Fatalf("unexpected temp artifact: %s", m)
		}
	}
}
