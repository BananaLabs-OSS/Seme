package godurablemanifest

import (
	"strings"
	"testing"
)

func TestParseStrictProfile(t *testing.T) {
	good := `{"version":"seme.durable-state-selection/v1","identity":"state","state_owner":"p/state","port_identity":"port","port_owner":"p/port","version_1_type":{"package":"p/state","name":"V1"},"version_2_type":{"package":"p/state","name":"V2"},"validator_1":{"package":"p/state","name":"ValidateV1"},"validator_2":{"package":"p/state","name":"ValidateV2"},"migration":{"package":"p/state","name":"Migrate"},"error_type":"int64","key_type":"string","codec":"seme.durable-state.canonical.v1","maximum_payload_bytes":1024,"maximum_key_bytes":128}`
	if _, err := Parse([]byte(good)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Replace(good, "int64", "bool", 1), strings.Replace(good, "canonical.v1", "json.v1", 1), good[:len(good)-1] + `,"extra":1}`} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Fatal("accepted malformed/profile drift")
		}
	}
}
