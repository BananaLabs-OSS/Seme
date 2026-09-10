package contractcatalog

import (
	"os"
	"testing"
)

func TestResolveOrderedTransportContractExactPins(t *testing.T) {
	raw, err := os.ReadFile("../../../modules/ordered-transport/v1/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ResolveOrderedTransportContract(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Validated() || c.Pin() != (Pin{orderedTransportModule, orderedTransportRev}) || len(c.Exports()) != 125 || len(c.Imports()) != 3 {
		t.Fatalf("contract = %#v", c.Pin())
	}
	bad := append([]byte(nil), raw...)
	bad[len(bad)-1] ^= 1
	if got, err := ResolveOrderedTransportContract(bad); err == nil || got.Validated() {
		t.Fatal("tampered contract accepted")
	}
}
