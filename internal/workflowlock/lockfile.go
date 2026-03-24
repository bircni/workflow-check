package workflowlock

import (
	"fmt"
	"os"
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write lockfile %s: %w", path, err)
	}
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
