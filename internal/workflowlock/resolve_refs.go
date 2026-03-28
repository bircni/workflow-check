package workflowlock

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

// resolveConcurrency caps concurrent git remotes (bulk) or resolver calls (fallback).
const resolveConcurrency = 16

// resolveRefs resolves many normalized refs. For GitResolver, one ls-remote is run per
// unique remote URL (host/owner/repo), then each ref is matched locally — much faster
// when many workflow pins hit the same action repo at different versions. Other
// resolvers are called concurrently with a bounded worker pool.
func resolveRefs(ctx context.Context, res Resolver, refs []NormalizedRef) (map[string]string, error) {
	if len(refs) == 0 {
		return map[string]string{}, nil
	}
	switch r := res.(type) {
	case GitResolver:
		return gitResolveBulk(ctx, r, refs)
	case *GitResolver:
		return gitResolveBulk(ctx, *r, refs)
	default:
		return resolveRefsParallel(ctx, res, refs)
	}
}

func gitResolveBulk(ctx context.Context, _ GitResolver, refs []NormalizedRef) (map[string]string, error) {
	groups := make(map[string][]NormalizedRef)
	remotes := make([]string, 0)
	seen := make(map[string]struct{})
	for _, ref := range refs {
		u := remoteURL(ref)
		if _, ok := seen[u]; !ok {
			seen[u] = struct{}{}
			remotes = append(remotes, u)
		}
		groups[u] = append(groups[u], ref)
	}

	out := make(map[string]string, len(refs))
	var mu sync.Mutex
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(resolveConcurrency)

	for _, remote := range remotes {
		groupRefs := groups[remote]
		eg.Go(func() error {
			stdout, errMsg := runGitLsRemoteArgs(gctx, "ls-remote", remote)
			if errMsg != "" {
				return fmt.Errorf("resolve via %s: %s", remote, errMsg)
			}
			candidates := parseLsRemote(stdout)
			for _, ref := range groupRefs {
				sha, err := pickRefSHA(candidates, ref, remote)
				if err != nil {
					return err
				}
				mu.Lock()
				out[ref.Key()] = sha
				mu.Unlock()
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

func resolveRefsParallel(ctx context.Context, res Resolver, refs []NormalizedRef) (map[string]string, error) {
	out := make(map[string]string, len(refs))
	var mu sync.Mutex
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(resolveConcurrency)

	for _, ref := range refs {
		eg.Go(func() error {
			sha, err := res.Resolve(gctx, ref)
			if err != nil {
				return err
			}
			mu.Lock()
			out[ref.Key()] = sha
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}
