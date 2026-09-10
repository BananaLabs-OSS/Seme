package service

import (
	"reflect"
	"testing"

	"example.test/go-uab-11/application"
	"example.test/go-uab-11/configuration"
	"example.test/go-uab-11/resource"
)

func TestApplyConfiguredResource(t *testing.T) {
	input := configuration.Input{NamePrefix: "x", UseDefaultLimit: true}
	state := application.State{Name: "x", Values: []int64{2, 3}, Counters: map[int64]int64{7: 10}}
	command := application.Command{Key: 7, Index: 1, Delta: 4, Amount: 1, Scale: true}
	valid := resource.Set{Notice: "Seme resources — 世界\n", Marker: []byte{0x00, 0xff, 'S', 'E', 'M', 'E', '\n'}}
	if got, want := ApplyConfiguredResource(input, state, command, valid), ApplyConfigured(input, state, command); !reflect.DeepEqual(got, want) {
		t.Fatalf("resource delegation drift: got %+v want %+v", got, want)
	}
	if got := ApplyConfiguredResource(input, state, command, resource.Set{}); got.Ok || got.Error != 40 {
		t.Fatalf("notice rejection: %+v", got)
	}
	if got := ApplyConfiguredResource(input, state, command, resource.Set{Notice: valid.Notice}); got.Ok || got.Error != 41 {
		t.Fatalf("marker rejection: %+v", got)
	}
}
