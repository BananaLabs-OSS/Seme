package greeting

import "testing"

func TestUseGreeting(t *testing.T) {
	if got := UseGreeting("Seme"); got != "Hello, Seme" {
		t.Fatalf("UseGreeting returned %q", got)
	}
}

