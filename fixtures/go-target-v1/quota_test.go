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
	response, err := Admit(AdmitRequest{Current: 40, Delta: 2, Limit: 50, Subject: "tenant-a", Evidence: []byte{1, 2, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if !response.Accepted {
		t.Fatal("expected request to be admitted")
	}
	if response.Subject != "tenant-a" || !bytes.Equal(response.Evidence, []byte{1, 2, 3}) {
		t.Fatalf("response did not preserve variable-width fields: %+v", response)
	}
	if got := strings.TrimSpace(output.String()); got != "quota.accepted=true" {
		t.Fatalf("log output = %q", got)
	}
	output.Reset()
	response, err = Admit(AdmitRequest{Current: 40, Delta: 2, Limit: 50})
	if err == nil || err.Error() != "subject required" || response.Accepted || response.Subject != "" || response.Evidence != nil {
		t.Fatalf("missing subject result = (%+v, %v)", response, err)
	}
	if output.Len() != 0 {
		t.Fatalf("invalid request performed logging effect: %q", output.String())
	}
}
