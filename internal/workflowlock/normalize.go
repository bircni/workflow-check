package workflowlock

import (
	"fmt"
	"strings"
)

// NormalizeRef converts a raw uses string into a canonical remote reference.
func NormalizeRef(raw string) (NormalizedRef, bool, error) {
	return NormalizeRefForHost(raw, "github.com")
}

// NormalizeRefForHost converts a raw uses string into a canonical remote reference
// using the provided default host for plain owner/repo refs.
func NormalizeRefForHost(raw, defaultHost string) (NormalizedRef, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return NormalizedRef{}, false, fmt.Errorf("empty uses value")
	}
	if strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") {
		return NormalizedRef{}, true, nil
	}

	at := strings.LastIndex(raw, "@")
	if at <= 0 || at == len(raw)-1 {
		return NormalizedRef{}, false, fmt.Errorf("unsupported uses reference %q", raw)
	}

	target := raw[:at]
	ref := raw[at+1:]
	parts := strings.Split(target, "/")
	if len(parts) < 2 {
		return NormalizedRef{}, false, fmt.Errorf("unsupported uses reference %q", raw)
	}

	host := defaultHost
	start := 0
	if looksLikeHost(parts[0]) {
		host = strings.ToLower(parts[0])
		start = 1
	}

	if len(parts[start:]) < 2 {
		return NormalizedRef{}, false, fmt.Errorf("unsupported uses reference %q", raw)
	}

	normalized := NormalizedRef{
		Host:  host,
		Owner: parts[start],
		Repo:  parts[start+1],
		Ref:   ref,
	}
	if len(parts[start:]) > 2 {
		normalized.Path = strings.Join(parts[start+2:], "/")
	}
	if strings.HasPrefix(normalized.Path, ".github/workflows/") {
		normalized.SourceKind = SourceKindReusableWorkflow
	} else {
		normalized.SourceKind = SourceKindAction
	}

	if err := validateNormalizedRef(normalized, raw); err != nil {
		return NormalizedRef{}, false, err
	}
	return normalized, false, nil
}

func looksLikeHost(part string) bool {
	return strings.Contains(part, ".") || strings.Contains(part, ":") || part == "localhost"
}

func validateNormalizedRef(ref NormalizedRef, raw string) error {
	for _, part := range []string{ref.Host, ref.Owner, ref.Repo, ref.Ref} {
		if part == "" {
			return fmt.Errorf("unsupported uses reference %q", raw)
		}
	}
	if strings.Contains(ref.Host, "://") {
		return fmt.Errorf("unsupported uses reference %q", raw)
	}
	if err := validateHostSegment(ref.Host, raw); err != nil {
		return err
	}
	if err := validateIdentitySegment(ref.Owner, raw, "owner"); err != nil {
		return err
	}
	if err := validateIdentitySegment(ref.Repo, raw, "repo"); err != nil {
		return err
	}
	if ref.Path != "" {
		if err := validateSlashDelimitedSegments(ref.Path, raw, "path"); err != nil {
			return err
		}
	}
	if err := validateSlashDelimitedSegments(ref.Ref, raw, "ref"); err != nil {
		return err
	}
	return nil
}

func validateHostSegment(host, raw string) error {
	if host == "." || host == ".." {
		return fmt.Errorf("unsupported uses reference %q: invalid host", raw)
	}
	if strings.ContainsAny(host, "/\\\t\n\r \x00") {
		return fmt.Errorf("unsupported uses reference %q: invalid host", raw)
	}
	return nil
}

func validateIdentitySegment(part, raw, label string) error {
	if part == "." || part == ".." {
		return fmt.Errorf("unsupported uses reference %q: invalid %s", raw, label)
	}
	if strings.ContainsAny(part, "/\\\t\n\r \x00") {
		return fmt.Errorf("unsupported uses reference %q: invalid %s", raw, label)
	}
	return nil
}

func validateSlashDelimitedSegments(value, raw, label string) error {
	for _, seg := range strings.Split(value, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("unsupported uses reference %q: invalid %s", raw, label)
		}
		if strings.ContainsAny(seg, "\\\t\n\r \x00") {
			return fmt.Errorf("unsupported uses reference %q: invalid %s", raw, label)
		}
	}
	return nil
}
