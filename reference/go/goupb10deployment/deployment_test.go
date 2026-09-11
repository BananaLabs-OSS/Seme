package goupb10deployment

import (
	"bytes"
	"seme.local/reference/targetplaninstance"
	"seme.local/reference/wire"
	"strings"
	"testing"
)

func TestCatalogAndLaunchAreDeterministicAndBound(t *testing.T) {
	id := func(last byte) wire.ID { var x wire.ID; x[15] = last; return x }
	dep := id(2)
	target := targetplaninstance.Target{Name: "test", Revision: 1, Rules: []targetplaninstance.Rule{{Identity: "b", Construct: id(1), MaximumRevision: 1, Fidelity: targetplaninstance.NativeIsland, Dependency: &dep, Evidence: []wire.ID{id(1)}}}}
	a, err := EmitCatalog([]byte("authority"), target)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EmitCatalog([]byte("authority"), target)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatal("catalog nondeterministic")
	}
	artifact, _ := DigestArtifact("vm.wasm", "wasm", []byte("wasm"))
	boundary := targetplaninstance.Boundary{Requirement: "r", Provider: id(1), Consumer: id(2), Interface: id(3), Transport: id(4)}
	l, err := EmitLaunch(a, []byte("plan"), []byte("project"), "commit", map[string]Artifact{"vm.wasm": artifact}, []targetplaninstance.Boundary{boundary})
	if err != nil {
		t.Fatal(err)
	}
	if err = ValidateLaunch(l, a, []byte("changed"), []byte("project"), "commit", map[string]Artifact{"vm.wasm": artifact}, []targetplaninstance.Boundary{boundary}); err == nil {
		t.Fatal("accepted stale plan")
	}
	tampered := bytes.Replace(a, []byte("test"), []byte("evil"), 1)
	if err = ValidateCatalog(tampered, []byte("authority"), target); err == nil || !strings.Contains(err.Error(), "catalog") {
		t.Fatal("accepted catalog tamper")
	}
}
