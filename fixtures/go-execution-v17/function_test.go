package recordproof

import "testing"

func TestLabel(t *testing.T) {
	if got := Label("雪🦀"); got != "雪🦀" { t.Fatalf("Label = %q", got) }
}
