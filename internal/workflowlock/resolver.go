package workflowlock

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Resolver resolves a normalized workflow ref to an immutable commit SHA.
type Resolver interface {
	Resolve(context.Context, NormalizedRef) (string, error)
}

// GitResolver resolves refs with git ls-remote.
type GitResolver struct{}

// Resolve fetches the SHA for the provided ref from its remote git repository.
func (GitResolver) Resolve(ctx context.Context, ref NormalizedRef) (string, error) {
	remote := fmt.Sprintf("https://%s/%s/%s", ref.Host, ref.Owner, ref.Repo)
	patterns := []string{
		"refs/tags/" + ref.Ref + "^{}",
		"refs/tags/" + ref.Ref,
		"refs/heads/" + ref.Ref,
		ref.Ref,
	}

	args := append([]string{"ls-remote", remote}, patterns...)
	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("resolve %s via %s: %s", ref.Key(), remote, msg)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	candidates := make(map[string]string, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		candidates[fields[1]] = fields[0]
	}
	for _, pattern := range patterns {
		if sha, ok := candidates[pattern]; ok {
			return sha, nil
		}
	}

	return "", fmt.Errorf("resolve %s via %s: ref not found", ref.Key(), remote)
}
