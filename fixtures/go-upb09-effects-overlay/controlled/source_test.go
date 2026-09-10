package controlled

import "testing"

func TestExplicitClockAndSeededDraw(t *testing.T) {
	if ValidClock(ClockSample{Sequence: 1, UnixMilliseconds: -1}) || ValidClock(ClockSample{}) || !ValidClock(ClockSample{Sequence: 1}) || ValidClock(ClockSample{Sequence: 257}) || ValidClock(ClockSample{Sequence: 1, UnixMilliseconds: 4102444800001}) || ValidSeed(RandomState{}) || !ValidSeed(RandomState{Value: 1}) || !ValidState(RandomState{Value: 0, Draws: 1}) || !ValidState(RandomState{Value: 1, Draws: 256}) {
		t.Fatal("input validation")
	}
	first := Next(RandomState{Value: 7})
	second := Next(RandomState{Value: 7})
	if first != second || first.Before.Value != 7 || first.After.Value != first.Value || first.After.Draws != 1 || first.Value != 337898 {
		t.Fatal("seeded draw is not deterministic")
	}
}
