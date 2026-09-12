package main

import "testing"

func TestPackageRelative(t *testing.T) {
	for _, x := range []struct {
		identity, want string
		ok             bool
	}{{"seme.upb12/service", ".", true}, {"seme.upb12/service/domain", "domain", true}, {"seme.other/x", "", false}, {"seme.upb12/service/../x", "", false}} {
		got, ok := packageRelative("seme.upb12/service", x.identity)
		if got != x.want || ok != x.ok {
			t.Fatalf("%q = %q %v", x.identity, got, ok)
		}
	}
}
