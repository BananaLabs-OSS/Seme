package orderedtransportinstance

import (
	"os"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

func TestAuthenticatesExactNeutralAuthority(t *testing.T) {
	raw, err := os.ReadFile("../../../modules/ordered-transport/v1/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	c, err := contractcatalog.ResolveOrderedTransportContract(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := Authenticate(c)
	if err != nil {
		t.Fatal(err)
	}
	if a.CodecIdentity != CodecIdentity || a.DigestIdentity != DigestIdentity || a.MaximumCommands != 256 || a.MaximumEvents != 1024 || a.MaximumEventsPerCommand != 4 || a.MaximumPayloadBytes != 3072 || a.MaximumFrameBytes != 4096 || a.CorrelationBytes != 16 || a.MaximumStreamBytes != 128 || a.FirstCommandSequence != 1 || a.CommandTerminalSentinel != 257 || a.EventTerminalSentinel != 1025 {
		t.Fatalf("authority = %#v", a)
	}
	if a.CodecLayout[1] != "ordered_transport.codec.magic.SEMEOT01" {
		t.Fatalf("codec = %#v", a.CodecLayout)
	}
}

func TestCatalogRejectsAuthorityMutation(t *testing.T) {
	raw, _ := os.ReadFile("../../../modules/ordered-transport/v1/module.seme")
	e, err := wire.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	f := e.Entities[id("110ec")]
	n := f.Fields[id("110")]
	n.Bytes = []byte("ordered_transport.maximum_frame_bytes.8192")
	f.Fields[id("110")] = n
	e.Entities[f.ID] = f
	bad, _ := wire.Encode(e)
	if c, err := contractcatalog.ResolveOrderedTransportContract(bad); err == nil || c.Validated() {
		t.Fatal("mutated authority accepted")
	}
}
