package mutationproof

import "testing"

func TestAppendOnce(t *testing.T) {
	if got := AppendOnce("雪", "🦀", true); got != "雪🦀" { t.Fatalf("enabled = %q", got) }
	if got := AppendOnce("雪", "🦀", false); got != "雪" { t.Fatalf("disabled = %q", got) }
}
