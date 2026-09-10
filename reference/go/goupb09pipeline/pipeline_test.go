package goupb09pipeline

import (
	"reflect"
	"testing"
)

func TestBuildRejectsUnauthenticatedWithoutPartialAuthority(t *testing.T) {
	result, err := Build(t.Context(), Input{})
	if err == nil || !reflect.DeepEqual(result, Result{}) {
		t.Fatal("accepted unauthenticated or returned partial authority")
	}
}

func TestBuildRejectsManifestBeforeEffectsAuthority(t *testing.T) {
	// A malformed, unauthenticated top-level request cannot return either new
	// artifact. Contract validation intentionally precedes all cumulative work.
	result, err := Build(t.Context(), Input{Manifest: []byte(`{"version":"wrong"}`)})
	if err == nil || len(result.ControlledEffects) != 0 || len(result.ProjectV12) != 0 || result.ControlledEffectsInput.Artifact != nil || result.ProjectInput.Composed != nil {
		t.Fatal("atomic failure violated")
	}
}
