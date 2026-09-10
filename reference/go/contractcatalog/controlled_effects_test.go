package contractcatalog

import (
	"os"
	"testing"
)

func TestResolveControlledEffectsContractExactPins(t *testing.T) {
	raw, err := os.ReadFile("../../../modules/controlled-effects/v1/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := ResolveControlledEffectsContract(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !contract.Validated() || contract.Pin() != (Pin{controlledEffectsModule, controlledEffectsRev}) || len(contract.Exports()) != 52 || len(contract.Imports()) != 3 {
		t.Fatalf("contract=%#v exports=%d imports=%d", contract.Pin(), len(contract.Exports()), len(contract.Imports()))
	}
	tampered := append([]byte(nil), raw...)
	tampered[len(tampered)-1] ^= 1
	if got, err := ResolveControlledEffectsContract(tampered); err == nil || got.Validated() {
		t.Fatal("tampered contract accepted")
	}
}
