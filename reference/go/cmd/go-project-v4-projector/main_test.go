package main

import "testing"

func TestPackagePath(t *testing.T) {
	for _, test := range []struct {
		module, identity, want string
		ok                     bool
	}{
		{"example.test/app", "example.test/app", ".", true},
		{"example.test/app", "example.test/app/domain", "domain", true},
		{"example.test/app", "example.test/other", "", false},
		{"example.test/app", "example.test/app/../escape", "", false},
	} {
		got, ok := packagePath(test.module, test.identity)
		if got != test.want || ok != test.ok {
			t.Fatalf("packagePath(%q,%q)=(%q,%v)", test.module, test.identity, got, ok)
		}
	}
}
