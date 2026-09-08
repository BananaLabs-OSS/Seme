package effectproof

import (
	"bytes"
	"log"
	"testing"
)

func TestObserve(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	flags := log.Flags()
	log.SetOutput(&output)
	log.SetFlags(0)
	defer func() { log.SetOutput(previous); log.SetFlags(flags) }()
	if Observe(true, false) || output.String() != "true\nfalse\n" {
		t.Fatalf("result/output = false/%q", output.String())
	}
}
