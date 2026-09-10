package main

import (
	"strings"
	"testing"
)

func TestV3IsAdditiveNeutralOwnershipSchema(t *testing.T) {
	v3 := declarations(3)
	if len(v3) != len(declarations(2))+3 {
		t.Fatalf("schemas=%d", len(v3))
	}
	names := map[string]bool{}
	for _, s := range v3 {
		if names[s.name] {
			t.Fatalf("duplicate %s", s.name)
		}
		names[s.name] = true
	}
	for _, want := range []string{"SemanticDeclarationKind", "OwnedSemanticDeclaration", "CompletePackageGraph"} {
		if !names[want] {
			t.Fatalf("missing %s", want)
		}
	}
	for _, bad := range []string{"Go", "JavaScript", "C#", "record declaration", "method declaration"} {
		for name := range names {
			if strings.Contains(strings.ToLower(name), strings.ToLower(bad)) {
				t.Fatalf("language-specific schema %s", name)
			}
		}
	}
}
