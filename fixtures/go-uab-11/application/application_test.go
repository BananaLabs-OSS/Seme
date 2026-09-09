package application

import (
	"io"
	"log"
	"math"
	"reflect"
	"strconv"
	"testing"
)

func TestApplyValidationOrderAndSuccess(t *testing.T) {
	log.SetOutput(io.Discard)
	base := State{Name: "世界", Values: []int64{2, 3}, Counters: map[int64]int64{7: 10}}
	for _, test := range []struct {
		command Command
		failure int64
	}{{Command{Key: 9, Index: 9}, 1}, {Command{Key: 7, Index: 9}, 2}} {
		before := clone(base)
		got := Apply(base, test.command)
		if got.Ok || got.Error != test.failure || !reflect.DeepEqual(base, before) {
			t.Fatalf("rejection %#v mutated=%v", got, !reflect.DeepEqual(base, before))
		}
	}
	negative := State{Name: "n", Values: []int64{1, -1}, Counters: map[int64]int64{7: 0}}
	if got := Apply(negative, Command{Key: 7}); got.Ok || got.Error != 3 {
		t.Fatalf("negative=%#v", got)
	}
	got := Apply(base, Command{Key: 7, Index: 1, Delta: 4, Amount: 1, Scale: true})
	if !got.Ok || got.Value.Result != 31 || !reflect.DeepEqual(got.Value.State.Values, []int64{2, 21, 21}) || got.Value.State.Counters[7] != 31 {
		t.Fatalf("success=%#v", got)
	}
	overflow := State{Name: "o", Values: []int64{0}, Counters: map[int64]int64{1: math.MaxInt64}}
	if got := Apply(overflow, Command{Key: 1, Amount: 1}); !got.Ok || got.Value.Result != math.MinInt64 {
		t.Fatalf("overflow=%#v", got)
	}
}

func TestApplyDeterministic128By16Corpus(t *testing.T) {
	log.SetOutput(io.Discard)
	random := uint64(0x5eed1234abcdef01)
	for sequence := int64(0); sequence < 128; sequence++ {
		boundary := sequence % 17
		if sequence%11 == 0 {
			boundary = math.MaxInt64
		}
		values := []int64{boundary, sequence % 5, 2}
		if sequence%13 == 0 {
			values = []int64{1, -1, 2}
		}
		state := State{Name: "sequence-" + itoa(sequence), Values: values, Counters: map[int64]int64{1: boundary, 2: 0}}
		for step := int64(0); step < 16; step++ {
			random ^= random << 13
			random ^= random >> 7
			random ^= random << 17
			sample := int64(random)
			index := sample % 2
			if index < 0 {
				index = -index
			}
			key := int64(2)
			if step%2 == 0 {
				key = 1
			}
			if step%7 == 1 {
				key = 99
			}
			delta := sample
			if step%8 == 3 {
				delta = math.MaxInt64
			}
			amount := sample >> 3
			if step%8 == 4 {
				amount = math.MinInt64
			}
			command := Command{Key: key, Index: index, Delta: delta, Amount: amount, Scale: step%2 == 0}
			before := clone(state)
			got := Apply(state, command)
			if got.Ok {
				state = got.Value.State
			} else if got.Error < 1 || got.Error > 3 || !reflect.DeepEqual(state, before) {
				t.Fatalf("sequence=%d step=%d: %#v", sequence, step, got)
			}
		}
	}
}

func itoa(value int64) string { return strconv.FormatInt(value, 10) }

func clone(state State) State {
	values := append([]int64(nil), state.Values...)
	counters := map[int64]int64{}
	for key, value := range state.Counters {
		counters[key] = value
	}
	return State{Name: state.Name, Values: values, Counters: counters}
}
