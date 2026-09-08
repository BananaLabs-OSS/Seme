package stringabi

import "testing"

func TestStringFunctions(t *testing.T) {
	if Join("雪", "🦀") != "雪🦀" {
		t.Fatal("multibyte concatenation failed")
	}
	if Join("", "") != "" {
		t.Fatal("empty concatenation failed")
	}
	if !Equal("same", "same") || Equal("same", "different") {
		t.Fatal("equality failed")
	}
}
