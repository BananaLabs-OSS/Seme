package canonicaleval

import (
	"math"
	"reflect"
	"strconv"
	"testing"
)

func TestEvaluateCumulativeSuccessAndPolicy(t *testing.T) {
	base := CumulativeState{Name: "deck", Values: []int64{2, 3}, Counters: map[int64]int64{7: 10}}
	for _, test := range []struct {
		scale bool
		want  int64
	}{{false, 20}, {true, 31}} {
		result, effects, err := EvaluateCumulative(base, CumulativeCommand{Key: 7, Index: 0, Delta: 4, Amount: 1, Scale: test.scale}, map[string]bool{"observability.log": true})
		if err != nil || result.Variant != "ok" || len(effects) != 1 || !effects[0].Value {
			t.Fatalf("bad success: %#v %#v %v", result, effects, err)
		}
		if got := result.Payload.Result.I64; got != itoa(test.want) {
			t.Fatalf("counter %s, want %d", got, test.want)
		}
	}
}

func TestEvaluateCumulativeValidationOrderAtomicityAndOverflow(t *testing.T) {
	state := CumulativeState{Name: "x", Values: []int64{-1}, Counters: map[int64]int64{2: math.MaxInt64}}
	tests := []struct {
		command CumulativeCommand
		code    string
	}{{CumulativeCommand{Key: 9, Index: 9}, "1"}, {CumulativeCommand{Key: 2, Index: 9}, "2"}, {CumulativeCommand{Key: 2, Index: 0}, "3"}}
	for _, test := range tests {
		before := cloneCumulative(state)
		result, effects, err := EvaluateCumulative(state, test.command, map[string]bool{"observability.log": true})
		if err != nil || result.Variant != "error" || result.Payload.I64 != test.code || len(effects) != 0 || !reflect.DeepEqual(state, before) {
			t.Fatalf("non-atomic rejection %#v %#v %v", result, effects, err)
		}
	}
	positive := CumulativeState{Name: "x", Values: []int64{0}, Counters: map[int64]int64{2: math.MaxInt64}}
	result, _, err := EvaluateCumulative(positive, CumulativeCommand{Key: 2, Index: 0, Delta: 0, Amount: 1}, map[string]bool{"observability.log": true})
	if err != nil || result.Payload.Result.I64 != itoa(math.MinInt64) {
		t.Fatalf("modular overflow failed: %#v %v", result, err)
	}
}

func TestEvaluateCumulativeDenialPrecedesMutationAndTrace(t *testing.T) {
	state := CumulativeState{Name: "x", Values: []int64{1}, Counters: map[int64]int64{2: 3}}
	before := cloneCumulative(state)
	result, effects, err := EvaluateCumulative(state, CumulativeCommand{Key: 2, Index: 0}, nil)
	if err == nil || result.Kind != "" || effects != nil || !reflect.DeepEqual(state, before) {
		t.Fatalf("denial was not atomic")
	}
}

func cloneCumulative(s CumulativeState) CumulativeState {
	v := append([]int64(nil), s.Values...)
	m := map[int64]int64{}
	for k, x := range s.Counters {
		m[k] = x
	}
	return CumulativeState{s.Name, v, m}
}
func itoa(v int64) string { return strconv.FormatInt(v, 10) }
