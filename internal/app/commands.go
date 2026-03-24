package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/bircni/workflow-check/internal/workflowlock"
)

var newEngine = workflowlock.NewEngineWithDefaultHost

func runLock(ctx context.Context, cfg Config, stdout io.Writer) error {
	engine := newEngine(workflowlock.GitResolver{}, cfg.DefaultHost)
	if err := engine.Lock(ctx, cfg.WorkflowDir, cfg.Lockfile); err != nil {
		return err
	}
	_, err := fmt.Fprintf(stdout, "updated %s\n", cfg.Lockfile)
	return err
}

func runVerify(ctx context.Context, cfg Config, stdout io.Writer) error {
	engine := newEngine(workflowlock.GitResolver{}, cfg.DefaultHost)
	if err := engine.Verify(ctx, cfg.WorkflowDir, cfg.Lockfile); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout, "workflow lock verified")
	return err
}

func runList(ctx context.Context, cfg Config, stdout io.Writer) error {
	_ = ctx

	refs, err := workflowlock.DiscoverRefsForHost(cfg.WorkflowDir, cfg.DefaultHost)
	if err != nil {
		return err
	}

	unique := make(map[string]workflowlock.DiscoveredRef, len(refs))
	for _, ref := range refs {
		unique[ref.Normalized.Key()] = ref
	}

	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	if cfg.Format == "json" {
		items := make([]workflowlock.DiscoveredRef, 0, len(keys))
		for _, key := range keys {
			items = append(items, unique[key])
		}
		return writeJSON(stdout, items)
	}

	for _, key := range keys {
		ref := unique[key]
		if _, err := fmt.Fprintf(stdout, "%s\t%s:%d\t%s\n", ref.Normalized.SourceKind, ref.File, ref.Line, key); err != nil {
			return err
		}
	}
	return nil
}

func runDiff(ctx context.Context, cfg Config, stdout io.Writer) error {
	engine := newEngine(workflowlock.GitResolver{}, cfg.DefaultHost)
	report, err := engine.Report(ctx, cfg.WorkflowDir, cfg.Lockfile)
	if err != nil {
		return err
	}

	if cfg.Format == "json" {
		return writeJSON(stdout, report)
	}

	if report.IsClean() {
		_, err := fmt.Fprintln(stdout, "clean")
		return err
	}

	for _, missing := range report.Missing {
		if _, err := fmt.Fprintf(stdout, "missing\t%s:%d\t%s\n", missing.File, missing.Line, missing.Normalized.Key()); err != nil {
			return err
		}
	}
	for _, drift := range report.Drift {
		if _, err := fmt.Fprintf(stdout, "drift\t%s:%d\t%s\t%s -> %s\n", drift.Discovered.File, drift.Discovered.Line, drift.Discovered.Normalized.Key(), drift.Entry.ResolvedSHA, drift.Resolved); err != nil {
			return err
		}
	}
	for _, stale := range report.Stale {
		if _, err := fmt.Fprintf(stdout, "stale\t%s\n", stale.Key()); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
