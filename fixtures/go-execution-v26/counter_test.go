package methodtransition

import "testing"

func TestValueReceiverReturnsExplicitTransition(t *testing.T) {
	original := Counter{Value: -7}
	transition := Step(original, 12)
	if transition.State.Value != 5 || transition.Result != 5 {
		t.Fatalf("transition = %+v", transition)
	}
	if original.Value != -7 {
		t.Fatalf("receiver mutated: %+v", original)
	}
}
