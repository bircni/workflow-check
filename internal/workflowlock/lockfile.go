package workflowlock

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"gopkg.in/yaml.v3"
)

// ReadLockfile reads and decodes a workflow lockfile from disk.
func ReadLockfile(path string) (Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Lockfile{}, fmt.Errorf("read lockfile %s: %w", path, err)
	}
	var lockfile Lockfile
	if err := yaml.Unmarshal(data, &lockfile); err != nil {
		return Lockfile{}, fmt.Errorf("parse lockfile %s: %w", path, err)
	}
	if err := validateLockfileUniqueKeys(lockfile.Entries); err != nil {
		return Lockfile{}, fmt.Errorf("lockfile %s: %w", path, err)
	}
	return lockfile, nil
}

// WriteLockfile writes entries to disk in deterministic key order.
func WriteLockfile(path string, entries []LockEntry) error {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key() < entries[j].Key()
	})
	lockfile := Lockfile{
		Version: SchemaVersion,
		Entries: entries,
	}
	data, err := yaml.Marshal(lockfile)
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}
	return writeFileAtomically(path, data, 0o644)
}

func validateLockfileUniqueKeys(entries []LockEntry) error {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		key := entry.Key()
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate lockfile entry key %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp.")
	if err != nil {
		return fmt.Errorf("create temp lockfile: %w", err)
	}
	tmpName := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("write temp lockfile: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync temp lockfile: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp lockfile: %w", err)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("chmod temp lockfile: %w", err)
	}

	if runtime.GOOS == "windows" {
		_ = os.Remove(path)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace lockfile %s: %w", path, err)
	}
	cleanup = false
	return nil
}

// IndexEntries builds a lookup map keyed by normalized lock entry identity.
func IndexEntries(entries []LockEntry) map[string]LockEntry {
	index := make(map[string]LockEntry, len(entries))
	for _, entry := range entries {
		index[entry.Key()] = entry
	}
	return index
}
