package callproof

import "testing"

func TestRender(t *testing.T) {
	if got := Render("雪", "🦀"); got != "[[雪][🦀]]" {
		t.Fatalf("Render = %q", got)
	}
}
