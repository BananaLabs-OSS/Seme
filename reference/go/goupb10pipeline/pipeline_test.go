package goupb10pipeline

import (
	"reflect"
	"testing"
)

func TestBuildRejectsUnauthenticatedWithoutPartialAuthority(t *testing.T) {
	result, err := Build(t.Context(), Input{})
	if err == nil || !reflect.DeepEqual(result, Result{}) {
		t.Fatal("accepted unauthenticated input or exposed partial placement authority")
	}
}
