package goprojector

import (
	"seme.local/reference/goprovider"
	"seme.local/reference/packagedetailmetadata"
)

// ProjectPackagesV2 is the strict Package Contract v2 projection entry point.
// The artifact must contain a complete validated detail graph; unlike
// ProjectPackages, this API has no exported-interface-only legacy fallback.
func ProjectPackagesV2(g1, packageV2Artifact []byte) (map[string][]byte, error) {
	metadata, err := packagedetailmetadata.Extract(packageV2Artifact)
	if err != nil {
		return nil, err
	}
	packages := make([]goprovider.PackageMetadata, len(metadata.Packages))
	for i := range metadata.Packages {
		packages[i] = metadata.Packages[i].Projection
	}
	return ProjectPackages(g1, packages)
}
