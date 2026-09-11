package projectv14instance

import (
	"reflect"
	"testing"
)

func TestEmitRejectsUnauthenticatedWithoutPartialArtifact(t *testing.T) {
	artifact, err := Emit(Inputs{})
	if err == nil || artifact != nil {
		t.Fatal("accepted unauthenticated input")
	}
	if err = Validate(Inputs{}); err == nil {
		t.Fatal("validated unauthenticated input")
	}
	if !reflect.DeepEqual(artifact, []byte(nil)) {
		t.Fatal("partial artifact")
	}
}
