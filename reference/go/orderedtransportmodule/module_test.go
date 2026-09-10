package orderedtransportmodule

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmitDeterministicNeutralAndBounded(t *testing.T) {
	var a, b bytes.Buffer
	if Emit(&a) != nil || Emit(&b) != nil || a.String() != b.String() {
		t.Fatal("nondeterministic")
	}
	for _, value := range []string{ModuleID, RevisionID, "00000000000000000000000000010100", "00000000000000000000000000010114", "000000000000000000000000000110e8", "000000000000000000000000000110fe"} {
		if !strings.Contains(a.String(), value) {
			t.Fatalf("missing %s", value)
		}
	}
	lower := strings.ToLower(a.String())
	for _, forbidden := range []string{"http", "websocket", "socket", "sse", "messagepack", "tcp", "udp", "filesystem", "database", "javascript", "golang"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("mechanism contamination %s", forbidden)
		}
	}
}
