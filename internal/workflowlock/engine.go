package workflowlock

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Engine coordinates workflow discovery, resolution, and lockfile operations.
type Engine struct {
	resolver    Resolver
	defaultHost string
}

// DriftIssue describes one workflow ref whose resolved SHA no longer matches the lockfile.
type DriftIssue struct {
	Discovered DiscoveredRef `json:"discovered"`
	Entry      LockEntry     `json:"entry"`
	Resolved   string        `json:"resolved"`
}

// Report summarizes the differences between workflows and an existing lockfile.
type Report struct {
	Missing []DiscoveredRef `json:"missing"`
	Drift   []DriftIssue    `json:"drift"`
	Stale   []LockEntry     `json:"stale"`
}

// NewEngine constructs a workflow lock engine with the provided resolver.
func NewEngine(resolver Resolver) Engine {
	return NewEngineWithDefaultHost(resolver, "github.com")
}

// NewEngineWithDefaultHost constructs a workflow lock engine with a configurable
// default host for plain owner/repo workflow refs.
func NewEngineWithDefaultHost(resolver Resolver, defaultHost string) Engine {
	return Engine{
		resolver:    resolver,
		defaultHost: defaultHost,
	}
}

// Lock discovers workflow refs, resolves them, and writes the lockfile.
func (e Engine) Lock(ctx context.Context, workflowDir, lockfilePath string) error {
	refs, err := DiscoverRefsForHost(workflowDir, e.defaultHost)
	if err != nil {
		return err
	}

	entries, err := e.resolveEntries(ctx, refs)
	if err != nil {
		return err
	}
	return WriteLockfile(lockfilePath, entries)
}

// Verify checks that workflows still match the existing lockfile.
func (e Engine) Verify(ctx context.Context, workflowDir, lockfilePath string) error {
	report, err := e.Report(ctx, workflowDir, lockfilePath)
	if err != nil {
		return err
	}
	if report.IsClean() {
		return nil
	}

	var issues []string
	for _, missing := range report.Missing {
		issues = append(issues, fmt.Sprintf("%s:%d missing lock entry for %s", missing.File, missing.Line, missing.Raw))
	}
	for _, stale := range report.Stale {
		issues = append(issues, "stale lock entry for "+stale.Key())
	}
	for _, drift := range report.Drift {
		if drift.Resolved == "" {
			issues = append(issues, fmt.Sprintf("%s:%d lock entry mismatch for %s (workflow ref vs lockfile fields differ; SHA not checked)", drift.Discovered.File, drift.Discovered.Line, drift.Discovered.Raw))
			continue
		}
		issues = append(issues, fmt.Sprintf("%s:%d lock drift for %s: lockfile=%s resolved=%s", drift.Discovered.File, drift.Discovered.Line, drift.Discovered.Raw, drift.Entry.ResolvedSHA, drift.Resolved))
	}

	return fmt.Errorf("verification failed:\n%s", strings.Join(issues, "\n"))
}

func (e Engine) resolveEntries(ctx context.Context, refs []DiscoveredRef) ([]LockEntry, error) {
	unique := make(map[string]NormalizedRef, len(refs))
	for _, ref := range refs {
		unique[ref.Normalized.Key()] = ref.Normalized
	}

	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	normRefs := make([]NormalizedRef, len(keys))
	for i, key := range keys {
		normRefs[i] = unique[key]
	}
	shas, err := resolveRefs(ctx, e.resolver, normRefs)
	if err != nil {
		return nil, err
	}

	entries := make([]LockEntry, 0, len(keys))
	for _, key := range keys {
		ref := unique[key]
		sha := shas[key]
		entries = append(entries, LockEntry{
			Host:        ref.Host,
			Owner:       ref.Owner,
			Repo:        ref.Repo,
			Path:        ref.Path,
			Ref:         ref.Ref,
			ResolvedSHA: sha,
			SourceKind:  ref.SourceKind,
		})
	}
	return entries, nil
}

// Report compares current workflow refs against the lockfile and returns the drift summary.
func (e Engine) Report(ctx context.Context, workflowDir, lockfilePath string) (Report, error) {
	refs, err := DiscoverRefsForHost(workflowDir, e.defaultHost)
	if err != nil {
		return Report{}, err
	}

	lockfile, err := ReadLockfile(lockfilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Report{}, fmt.Errorf("lockfile %s does not exist", lockfilePath)
		}
		return Report{}, err
	}
	if lockfile.Version != SchemaVersion {
		return Report{}, fmt.Errorf("unsupported lockfile version %d", lockfile.Version)
	}

	index := IndexEntries(lockfile.Entries)
	expectedKeys := make(map[string]DiscoveredRef, len(refs))
	for _, ref := range refs {
		expectedKeys[ref.Normalized.Key()] = ref
	}

	keys := make([]string, 0, len(expectedKeys))
	for key := range expectedKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	report := Report{
		Missing: []DiscoveredRef{},
		Drift:   []DriftIssue{},
		Stale:   []LockEntry{},
	}

	for _, entry := range lockfile.Entries {
		if _, ok := expectedKeys[entry.Key()]; !ok {
			report.Stale = append(report.Stale, entry)
		}
	}

	type driftCheck struct {
		discovered DiscoveredRef
		entry      LockEntry
	}
	var needsResolve []driftCheck
	for _, key := range keys {
		discovered := expectedKeys[key]
		entry, ok := index[key]
		if !ok {
			report.Missing = append(report.Missing, discovered)
			continue
		}
		if !LockFieldsMatch(entry, discovered.Normalized) {
			report.Drift = append(report.Drift, DriftIssue{
				Discovered: discovered,
				Entry:      entry,
				Resolved:   "",
			})
			continue
		}
		needsResolve = append(needsResolve, driftCheck{discovered: discovered, entry: entry})
	}

	if len(report.Missing) > 0 || len(report.Stale) > 0 {
		sort.Slice(report.Missing, func(i, j int) bool {
			return report.Missing[i].Normalized.Key() < report.Missing[j].Normalized.Key()
		})
		sort.Slice(report.Drift, func(i, j int) bool {
			return report.Drift[i].Discovered.Normalized.Key() < report.Drift[j].Discovered.Normalized.Key()
		})
		sort.Slice(report.Stale, func(i, j int) bool {
			return report.Stale[i].Key() < report.Stale[j].Key()
		})
		return report, nil
	}

	if len(needsResolve) > 0 {
		normRefs := make([]NormalizedRef, len(needsResolve))
		for i := range needsResolve {
			normRefs[i] = needsResolve[i].discovered.Normalized
		}
		shas, err := resolveRefs(ctx, e.resolver, normRefs)
		if err != nil {
			return Report{}, err
		}
		for _, c := range needsResolve {
			sha := shas[c.discovered.Normalized.Key()]
			if sha != c.entry.ResolvedSHA {
				report.Drift = append(report.Drift, DriftIssue{
					Discovered: c.discovered,
					Entry:      c.entry,
					Resolved:   sha,
				})
			}
		}
	}

	sort.Slice(report.Missing, func(i, j int) bool {
		return report.Missing[i].Normalized.Key() < report.Missing[j].Normalized.Key()
	})
	sort.Slice(report.Drift, func(i, j int) bool {
		return report.Drift[i].Discovered.Normalized.Key() < report.Drift[j].Discovered.Normalized.Key()
	})
	sort.Slice(report.Stale, func(i, j int) bool {
		return report.Stale[i].Key() < report.Stale[j].Key()
	})
	return report, nil
}

// IsClean reports whether the comparison found any missing, drifted, or stale entries.
func (r Report) IsClean() bool {
	return len(r.Missing) == 0 && len(r.Drift) == 0 && len(r.Stale) == 0
}
