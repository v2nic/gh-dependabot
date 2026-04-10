package submitter

// Snapshot represents a dependency graph snapshot
type Snapshot struct {
	Version   int                      `json:"version"`
	SHA       string                   `json:"sha"`
	Ref       string                   `json:"ref"`
	Job       Job                      `json:"job"`
	Detector  Detector                 `json:"detector"`
	Scanned   string                   `json:"scanned"`
	Manifests map[string]Manifest      `json:"manifests"`
}

type Job struct {
	Correlator string `json:"correlator"`
	ID         string `json:"id"`
}

type Detector struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

type Manifest struct {
	Name     string            `json:"name"`
	File     File              `json:"file"`
	Resolved map[string]Dependency `json:"resolved"`
}

type File struct {
	SourceLocation string `json:"source_location"`
}

type Dependency struct {
	PackageURL   string `json:"package_url"`
	Relationship string `json:"relationship"`
	Scope        string `json:"scope,omitempty"`
}