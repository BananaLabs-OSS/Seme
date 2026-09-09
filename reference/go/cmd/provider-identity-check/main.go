// Command provider-identity-check independently verifies that a native-only
// formatting revision preserved every imported semantic declaration identity.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"seme.local/reference/goprovider"
)

func main() {
	if len(os.Args) == 4 && os.Args[1] == "forge-ambiguous" {
		manifest := read(os.Args[2])
		if len(manifest.Declarations) < 2 {
			fatal("requires two declarations")
		}
		manifest.Declarations[1].MatchFingerprint = manifest.Declarations[0].MatchFingerprint
		encoded, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			fatal(err.Error())
		}
		encoded = append(encoded, '\n')
		if err := os.WriteFile(os.Args[3], encoded, 0o644); err != nil {
			fatal(err.Error())
		}
		return
	}
	if len(os.Args) != 3 {
		fatal("usage: provider-identity-check BEFORE AFTER | forge-ambiguous PRIOR OUT")
	}
	before := read(os.Args[1])
	after := read(os.Args[2])
	if before.Revision == after.Revision {
		fatal("formatting did not produce a distinct source revision")
	}
	if len(before.Declarations) != len(after.Declarations) {
		fatal("declaration count changed")
	}
	byKey := map[string]goprovider.Declaration{}
	for _, declaration := range after.Declarations {
		byKey[declaration.NativeKey] = declaration
	}
	for _, old := range before.Declarations {
		now, ok := byKey[old.NativeKey]
		if !ok {
			fatal("declaration disappeared: " + old.NativeKey)
		}
		if old.ID != now.ID || old.EvidenceID != now.EvidenceID {
			fatal("semantic identity changed: " + old.NativeKey)
		}
		if old.Signature != now.Signature {
			fatal("semantic signature changed: " + old.NativeKey)
		}
	}
}

func read(path string) goprovider.Manifest {
	manifest, err := goprovider.ReadManifest(path)
	if err != nil {
		fatal(err.Error())
	}
	return manifest
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "provider-identity-check:", message)
	os.Exit(1)
}
