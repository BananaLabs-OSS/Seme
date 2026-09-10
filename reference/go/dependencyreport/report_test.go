package dependencyreport_test

import (
	"os"
	"reflect"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyemitter"
	"seme.local/reference/dependencyreport"
	"seme.local/reference/dependencyresolution"
)

func TestInspectAuthenticatedStableAndSourceByteFree(t *testing.T) {
	b, err := os.ReadFile("../../../modules/dependency/v1/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	c, err := contractcatalog.ResolveDependencyContract(b)
	if err != nil {
		t.Fatal(err)
	}
	closure := dependencyresolution.Closure{Requirements: []dependencyresolution.Requirement{{Identity: "example.test/a", Requirement: "v1", Kind: dependencyresolution.External}}, Entries: []dependencyresolution.Entry{{Identity: "example.test/a", Ecosystem: "registry", Version: "v1", Integrity: "sha256:abc", IntegrityAlgorithm: "sha256", Source: "registry/example.test/a", SourceKind: "registry", Digest: "abc", Kind: dependencyresolution.External}}}
	instance, err := dependencyemitter.Emit(c, closure)
	if err != nil {
		t.Fatal(err)
	}
	a, err := dependencyreport.Inspect(c, instance)
	if err != nil {
		t.Fatal(err)
	}
	z, err := dependencyreport.Inspect(c, instance)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, z) || len(a.Resolutions) != 1 || a.Resolutions[0].SourceIdentity != "registry/example.test/a" || a.ContentRevision == "" || a.ArtifactRevision == "" {
		t.Fatalf("bad report: %#v", a)
	}
	instance[len(instance)-1] ^= 1
	if out, err := dependencyreport.Inspect(c, instance); err == nil || !reflect.DeepEqual(out, dependencyreport.Report{}) {
		t.Fatalf("tamper output=%#v err=%v", out, err)
	}
	if out, err := dependencyreport.Inspect(contractcatalog.Contract{}, nil); err == nil || !reflect.DeepEqual(out, dependencyreport.Report{}) {
		t.Fatalf("untrusted output=%#v err=%v", out, err)
	}
}
