package goupb08portruntime

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/orderedtransportplacement"
	"seme.local/reference/orderedtransportruntime"
	"seme.local/reference/wasmtarget"
	"testing"
)

func profile(t *testing.T) orderedtransportruntime.Profile {
	b, e := os.ReadFile("../../../modules/ordered-transport/v1/module.seme")
	if e != nil {
		t.Fatal(e)
	}
	c, e := contractcatalog.ResolveOrderedTransportContract(b)
	if e != nil {
		t.Fatal(e)
	}
	p, e := orderedtransportruntime.AuthenticatedProfile(c)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func layouts() orderedtransportruntime.Layouts {
	l := wasmtarget.PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "bytes", FixedSize: 8, VariablePayload: true, MaximumPayload: 4096, Encoding: "u32le-offset-u32le-byte-length/opaque"}
	return orderedtransportruntime.Layouts{Command: l, Response: l}
}
func typed(s string) []byte {
	b := make([]byte, 8+len(s))
	binary.LittleEndian.PutUint32(b[:4], 8)
	binary.LittleEndian.PutUint32(b[4:8], uint32(len(s)))
	copy(b[8:], s)
	return b
}

type chunks struct {
	b []byte
	n int
}

func (c *chunks) Read(p []byte) (int, error) {
	if len(c.b) == 0 {
		return 0, io.EOF
	}
	n := c.n
	if n > len(c.b) {
		n = len(c.b)
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p, c.b[:n])
	c.b = c.b[n:]
	return n, nil
}

type handler struct{ calls int }

func (h *handler) Handle(orderedtransportruntime.Frame) orderedtransportruntime.SemanticResult {
	h.calls++
	return orderedtransportruntime.SemanticResult{Response: orderedtransportruntime.Frame{Kind: orderedtransportruntime.FrameResponse, Payload: typed("response")}, Committed: true}
}
func TestSelectedHostPortFragmentationCoalescingEvidenceAndRetry(t *testing.T) {
	p, l := profile(t), layouts()
	a, _ := orderedtransportruntime.EncodeFrame(p, l, orderedtransportruntime.Frame{Kind: orderedtransportruntime.FrameCommand, Payload: typed("a")})
	b, _ := orderedtransportruntime.EncodeFrame(p, l, orderedtransportruntime.Frame{Kind: orderedtransportruntime.FrameCommand, Payload: typed("b")})
	store := DirectoryStore{Root: t.TempDir()}
	port := &HostPort{Profile: p, Layouts: l, Input: &chunks{b: append(bytes.Clone(a), b...), n: 1}, Store: store}
	grants := orderedtransportruntime.Grants{p.Receive.Capability: true, p.Send.Capability: true}
	h := &handler{}
	first := orderedtransportruntime.Execute(p, l, grants, port, h)
	second := orderedtransportruntime.Execute(p, l, grants, port, h)
	if !first.Sent || !second.Sent || h.calls != 2 || len(first.Trace) != 2 || first.Trace[0].Sequence != p.Receive.Sequence || first.Trace[1].Sequence != p.Send.Sequence {
		t.Fatalf("execution %#v %#v", first, second)
	}
	evidence, ok := store.Get(first.Trace[1].Sent.Receipt)
	if !ok || !bytes.Equal(evidence, first.ResponseFrame) || sha256.Sum256(evidence) != first.Trace[1].Sent.AcceptedSHA256 {
		t.Fatal("external exact bytes missing")
	}
	failing := &HostPort{Profile: p, Layouts: l, Input: bytes.NewReader(a), Store: store, FailSend: errors.New("fail")}
	h = &handler{}
	failed := orderedtransportruntime.Execute(p, l, grants, failing, h)
	if !failed.Committed || failed.Retry == nil || failed.Sent || h.calls != 1 {
		t.Fatal("committed failure")
	}
	failing.FailSend = nil
	retry := orderedtransportruntime.RetrySend(p, grants, failing, *failed.Retry)
	if !retry.Sent || h.calls != 1 || !bytes.Equal(retry.ResponseFrame, failed.ResponseFrame) {
		t.Fatal("exact retry")
	}
}
func TestPulpPlacementReportRejectsTransportPort(t *testing.T) {
	c := orderedtransportplacement.Candidate{PulpCommit: orderedtransportplacement.PinnedPulpCommit, ManifestSHA256: orderedtransportplacement.PinnedManifestSHA256, Provider: orderedtransportplacement.PlannerProvider, Carrier: orderedtransportplacement.PlannerCarrier, Synchronous: true, OpaqueBytes: true}
	r, e := RejectPulpTransportPort(c)
	if e != nil || r.TransportPortSupported || r.Reason == "" {
		t.Fatal(r, e)
	}
}
