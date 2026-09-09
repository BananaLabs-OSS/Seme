package composite

import "testing"

func TestVectors(t *testing.T) {
	vectors := []struct {
		name string
		in   Option[Result[[]byte, string]]
		want bool
	}{
		{"none", Option[Result[[]byte, string]]{}, false},
		{"ok", Option[Result[[]byte, string]]{Some: true, Value: Result[[]byte, string]{Ok: true, Value: []byte("ok")}}, true},
		{"wrong bytes", Option[Result[[]byte, string]]{Some: true, Value: Result[[]byte, string]{Ok: true, Value: []byte("no")}}, false},
		{"error", Option[Result[[]byte, string]]{Some: true, Value: Result[[]byte, string]{Error: "bad"}}, true},
		{"wrong error", Option[Result[[]byte, string]]{Some: true, Value: Result[[]byte, string]{Error: "no"}}, false},
	}
	for _, vector := range vectors {
		t.Run(vector.name, func(t *testing.T) {
			if got := Admit(vector.in); got != vector.want {
				t.Fatalf("got %v, want %v", got, vector.want)
			}
		})
	}
}
