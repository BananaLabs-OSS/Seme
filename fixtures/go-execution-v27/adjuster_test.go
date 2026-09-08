package interfacedispatch

import "testing"

func TestTwoImplementationsDispatchThroughOneContract(t *testing.T) {
	if got := Apply(OffsetAdjuster{Offset: 5}, 7); got != 12 {
		t.Fatalf("offset result = %d", got)
	}
	if got := Apply(ScaleAdjuster{Factor: 5}, 7); got != 35 {
		t.Fatalf("scale result = %d", got)
	}
	if got := Dispatch(false, 5, 7); got != 12 {
		t.Fatalf("offset dispatch = %d", got)
	}
	if got := Dispatch(true, 5, 7); got != 35 {
		t.Fatalf("scale dispatch = %d", got)
	}
}
