package quota

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestAdmitLogsDecision(t *testing.T) {
	var output bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&output)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	})
	if !Admit(40, 2, 50) {
		t.Fatal("expected request to be admitted")
	}
	if got := strings.TrimSpace(output.String()); got != "quota.accepted=true" {
		t.Fatalf("log output = %q", got)
	}
}
