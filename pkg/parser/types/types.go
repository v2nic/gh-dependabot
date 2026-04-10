package types

// Dependency represents a parsed dependency from a lockfile
type Dependency struct {
	PackageURL   string `json:"package_url"`
	Relationship string `json:"relationship"`
	Scope        string `json:"scope,omitempty"`
}
