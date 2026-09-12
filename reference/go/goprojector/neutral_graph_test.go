package goprojector

import "testing"

func TestProjectionTargetPresentation(t *testing.T) {
	for _, test := range []struct {
		module, identity, want string
		ok                     bool
	}{{"example.test/service", "example.test/service/domain", "domain", true}, {"example.test/service", "example.test/service", ".", true}, {"example.test/service", "example.test/other", "", false}, {"example.test/service", "example.test/service/../escape", "", false}} {
		got, ok := projectionRelative(test.module, test.identity)
		if got != test.want || ok != test.ok {
			t.Fatalf("relative %q %q = %q %v", test.module, test.identity, got, ok)
		}
	}
	if got := projectionImport("domain/service.js", "durable/state.js", "javascript"); got != "../durable/state.js" {
		t.Fatal(got)
	}
	if got := projectionImport("domain/service.lua", "durable/state.lua", "lua"); got != "durable.state" {
		t.Fatal(got)
	}
}
