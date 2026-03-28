package workflowlock

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

var execCommandContext = exec.CommandContext

// Resolver resolves a normalized workflow ref to an immutable commit SHA.
type Resolver interface {
	Resolve(context.Context, NormalizedRef) (string, error)
}

func remoteURL(ref NormalizedRef) string {
	return fmt.Sprintf("https://%s/%s/%s", ref.Host, ref.Owner, ref.Repo)
}

func refRemotePatterns(ref NormalizedRef) []string {
	return []string{
		"refs/tags/" + ref.Ref + "^{}",
		"refs/tags/" + ref.Ref,
		"refs/heads/" + ref.Ref,
		ref.Ref,
	}
}

func parseLsRemote(stdout string) map[string]string {
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	candidates := make(map[string]string, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		candidates[fields[1]] = fields[0]
	}
	return candidates
}

func pickRefSHA(candidates map[string]string, ref NormalizedRef, remote string) (string, error) {
	for _, pattern := range refRemotePatterns(ref) {
		if sha, ok := candidates[pattern]; ok {
			return sha, nil
		}
	}
	return "", fmt.Errorf("resolve %s via %s: ref not found", ref.Key(), remote)
}

// runGitLsRemoteArgs runs git with the given args (e.g. "ls-remote", remote, patterns...).
// On failure it returns empty stdout and a non-empty stderr-or-error message.
func runGitLsRemoteArgs(ctx context.Context, args ...string) (stdout, errMsg string) {
	cmd := execCommandContext(ctx, "git", args...)
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", msg
	}
	return outBuf.String(), ""
}

// GitResolver resolves refs with git ls-remote.
type GitResolver struct{}

// Resolve fetches the SHA for the provided ref from its remote git repository.
func (GitResolver) Resolve(ctx context.Context, ref NormalizedRef) (string, error) {
	remote := remoteURL(ref)
	args := append([]string{"ls-remote", remote}, refRemotePatterns(ref)...)
	stdout, errMsg := runGitLsRemoteArgs(ctx, args...)
	if errMsg != "" {
		return "", fmt.Errorf("resolve %s via %s: %s", ref.Key(), remote, errMsg)
	}
	candidates := parseLsRemote(stdout)
	sha, err := pickRefSHA(candidates, ref, remote)
	if err != nil {
		return "", err
	}
	return sha, nil
}
