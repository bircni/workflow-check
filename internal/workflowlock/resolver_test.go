package workflowlock

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestGitResolverPrefersAnnotatedTags(t *testing.T) {
	t.Parallel()

	restore := stubExecCommand(t, "1111111111111111111111111111111111111111 refs/tags/v1.0.0\n2222222222222222222222222222222222222222 refs/tags/v1.0.0^{}\n", "", nil)
	defer restore()

	sha, err := GitResolver{}.Resolve(context.Background(), NormalizedRef{
		Host:  "github.com",
		Owner: "acme",
		Repo:  "tool",
		Ref:   "v1.0.0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != strings.Repeat("2", 40) {
		t.Fatalf("unexpected sha: %s", sha)
	}
}

func TestGitResolverPrefersBranchesWhenNoTagExists(t *testing.T) {
	t.Parallel()

	restore := stubExecCommand(t, "3333333333333333333333333333333333333333 refs/heads/main\n", "", nil)
	defer restore()

	sha, err := GitResolver{}.Resolve(context.Background(), NormalizedRef{
		Host:  "github.com",
		Owner: "acme",
		Repo:  "tool",
		Ref:   "main",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != strings.Repeat("3", 40) {
		t.Fatalf("unexpected sha: %s", sha)
	}
}

func TestGitResolverWrapsCommandFailure(t *testing.T) {
	t.Parallel()

	restore := stubExecCommand(t, "", "fatal: remote error", errors.New("exit status 2"))
	defer restore()

	_, err := GitResolver{}.Resolve(context.Background(), NormalizedRef{
		Host:  "github.com",
		Owner: "acme",
		Repo:  "tool",
		Ref:   "main",
	})
	if err == nil || !strings.Contains(err.Error(), "fatal: remote error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitResolverReportsMissingRef(t *testing.T) {
	t.Parallel()

	restore := stubExecCommand(t, "", "", nil)
	defer restore()

	_, err := GitResolver{}.Resolve(context.Background(), NormalizedRef{
		Host:  "github.com",
		Owner: "acme",
		Repo:  "tool",
		Ref:   "missing",
	})
	if err == nil || !strings.Contains(err.Error(), "ref not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func stubExecCommand(t *testing.T, stdout, stderr string, runErr error) func() {
	t.Helper()

	previous := execCommandContext
	execCommandContext = func(context.Context, string, ...string) *exec.Cmd {
		return fakeCmd(stdout, stderr, runErr)
	}
	return func() {
		execCommandContext = previous
	}
}

func fakeCmd(stdout, stderr string, runErr error) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"HELPER_STDOUT="+stdout,
		"HELPER_STDERR="+stderr,
	)
	if runErr != nil {
		cmd.Env = append(cmd.Env, "HELPER_EXIT_CODE=2")
	}
	return cmd
}

func TestHelperProcess(t *testing.T) {
	t.Helper()
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	_, _ = bytes.NewBufferString(os.Getenv("HELPER_STDOUT")).WriteTo(os.Stdout)
	_, _ = bytes.NewBufferString(os.Getenv("HELPER_STDERR")).WriteTo(os.Stderr)
	if os.Getenv("HELPER_EXIT_CODE") != "" {
		os.Exit(2)
	}
	os.Exit(0)
}
