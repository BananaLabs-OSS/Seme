package immutableclosure

import "testing"

func TestReturnedImmutableClosure(t *testing.T) {
	if got := Run(12, 30); got != 42 {
		t.Fatalf("positive result = %d", got)
	}
	if got := Run(-50, 8); got != -42 {
		t.Fatalf("negative result = %d", got)
	}
	left, right := MakeAdder(10), MakeAdder(-10)
	if left(1) != 11 || right(1) != -9 {
		t.Fatal("closure environments aliased")
	}
}
