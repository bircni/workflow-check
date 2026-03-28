package workflowlock

// SchemaVersion identifies the current workflow lockfile schema.
const SchemaVersion = 1

// SourceKind describes the kind of remote workflow reference.
type SourceKind string

const (
	// SourceKindAction is a normal remote action reference.
	SourceKindAction SourceKind = "action"
	// SourceKindReusableWorkflow is a remote reusable workflow reference.
	SourceKindReusableWorkflow SourceKind = "reusable_workflow"
)

// NormalizedRef is a canonicalized workflow reference used for locking.
type NormalizedRef struct {
	Host       string     `json:"host"`
	Owner      string     `json:"owner"`
	Repo       string     `json:"repo"`
	Path       string     `json:"path,omitempty"`
	Ref        string     `json:"ref"`
	SourceKind SourceKind `json:"source_kind"`
}

// Key returns the stable lockfile key for the normalized reference.
func (r NormalizedRef) Key() string {
	if r.Path == "" {
		return r.Host + "/" + r.Owner + "/" + r.Repo + "@" + r.Ref
	}
	return r.Host + "/" + r.Owner + "/" + r.Repo + "/" + r.Path + "@" + r.Ref
}

// DiscoveredRef is a normalized workflow reference plus its source location.
type DiscoveredRef struct {
	File       string        `json:"file"`
	Line       int           `json:"line"`
	Raw        string        `json:"raw"`
	Normalized NormalizedRef `json:"normalized"`
}

// Lockfile is the serialized workflow lock document.
type Lockfile struct {
	Version int         `yaml:"version" json:"version"`
	Entries []LockEntry `yaml:"entries" json:"entries"`
}

// LockEntry stores one locked remote workflow reference.
type LockEntry struct {
	Host        string         `yaml:"host" json:"host"`
	Owner       string         `yaml:"owner" json:"owner"`
	Repo        string         `yaml:"repo" json:"repo"`
	Path        string         `yaml:"path,omitempty" json:"path,omitempty"`
	Ref         string         `yaml:"ref" json:"ref"`
	ResolvedSHA string         `yaml:"resolved_sha" json:"resolved_sha"`
	SourceKind  SourceKind     `yaml:"source_kind" json:"source_kind"`
	Metadata    map[string]any `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// Key returns the stable lockfile key for the entry.
func (e LockEntry) Key() string {
	ref := NormalizedRef{
		Host:       e.Host,
		Owner:      e.Owner,
		Repo:       e.Repo,
		Path:       e.Path,
		Ref:        e.Ref,
		SourceKind: e.SourceKind,
	}
	return ref.Key()
}

// LockFieldsMatch reports whether stored lock metadata matches the workflow-normalized
// ref (host, repo coordinates, pin, and kind). This is checked before any remote SHA
// resolution during verify/report.
func LockFieldsMatch(e LockEntry, n NormalizedRef) bool {
	return e.Host == n.Host &&
		e.Owner == n.Owner &&
		e.Repo == n.Repo &&
		e.Path == n.Path &&
		e.Ref == n.Ref &&
		e.SourceKind == n.SourceKind
}
