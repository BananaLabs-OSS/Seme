package controlled

import "testing"

func TestExplicitClockAndSeededDraw(t *testing.T) {
	if ValidClock(ClockSample{UnixMilliseconds: -1}) || !ValidClock(ClockSample{}) || ValidRandom(RandomState{}) {
		t.Fatal("input validation")
	}
	first := Next(RandomState{Value: 7})
	second := Next(RandomState{Value: 7})
	if first != second || first.Before.Value != 7 || first.After.Value != first.Value {
		t.Fatal("seeded draw is not deterministic")
	}
}
