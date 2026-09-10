package goupb09report

import (
	"bytes"
	"reflect"
	"testing"

	"seme.local/reference/goupb09bundle"
)

func TestRejectsUnauthenticatedAndMarshalIsDeterministic(t *testing.T) {
	if got, err := Inspect(goupb09bundle.Result{}); err == nil || !reflect.DeepEqual(got, Report{}) {
		t.Fatal("reported unauthenticated authority")
	}
	r := Report{ProjectContractRevision: "revision", Clock: Clock{Identity: "clock.injected.unix-milliseconds.v1"}, Bounds: Bounds{MaximumSteps: 256}}
	first, err := Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Marshal(r)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("nondeterministic")
	}
	first[0] ^= 1
	third, err := Marshal(r)
	if err != nil || !bytes.Equal(second, third) {
		t.Fatal("marshal storage aliased")
	}
}
