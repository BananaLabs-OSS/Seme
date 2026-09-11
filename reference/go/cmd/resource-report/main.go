// Command resource-report emits a deterministic semantic view of Resource v1.
// Authentication remains the responsibility of the Project-v9 composer/loader.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"seme.local/reference/resourceinstance"
)

type resource struct {
	Identity, Owner, SourceUnit, Path, MediaType, SHA256 string
	Kind, Size                                           uint64
}
type report struct {
	Resources  []resource
	Placements []resourceinstance.Placement
}

func main() {
	if len(os.Args) != 2 {
		fatal(fmt.Errorf("usage: resource-report RESOURCE.seme"))
	}
	b, e := os.ReadFile(os.Args[1])
	if e != nil {
		fatal(e)
	}
	m, e := resourceinstance.ModelFromArtifact(b)
	if e != nil {
		fatal(e)
	}
	r := report{Placements: m.Placements}
	for _, x := range m.Resources {
		r.Resources = append(r.Resources, resource{x.Identity, x.Owner.String(), x.SourceUnit.String(), x.Path, x.MediaType, hex.EncodeToString(x.SHA256[:]), x.Kind, x.Size})
	}
	fatal(json.NewEncoder(os.Stdout).Encode(r))
}
func fatal(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, "resource-report:", e)
		os.Exit(1)
	}
}
