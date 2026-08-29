package ordinary

import "testing"

func TestGreeting(t *testing.T) {
	if got := Greeting("world"); got != "Hello, world" {
		t.Fatalf("Greeting() = %q", got)
	}
}
