package goorderedtransportmanifest

import "testing"

const valid = `{"version":"seme.ordered-transport-selection/v1","streams":[{"identity":"s","package":"p"}],"command_kinds":[{"identity":"c","owner":"p","payload":{"package":"p","name":"C"}}],"event_kinds":[{"identity":"e","owner":"p","payload":{"package":"p","name":"E"}}],"ports":[{"identity":"x","package":"p"}],"dispatch":{"package":"p","name":"D"},"replay":{"package":"p","name":"R"}}`

func TestParseStrict(t *testing.T) {
	x, err := Parse([]byte(valid))
	if err != nil || len(x.Streams) != 1 || x.Dispatch.Name != "D" {
		t.Fatal(err)
	}
	for _, bad := range []string{"", valid + "{}", `{"version":"wrong"}`, `{"version":"seme.ordered-transport-selection/v1","unknown":1}`} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}
