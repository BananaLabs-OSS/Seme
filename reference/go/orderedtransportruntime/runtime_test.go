package orderedtransportruntime

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"os"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wasmtarget"
)

func testProfile(t *testing.T) Profile {
	t.Helper()
	raw, err := os.ReadFile("../../../modules/ordered-transport/v1/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	c, err := contractcatalog.ResolveOrderedTransportContract(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := AuthenticatedProfile(c)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func byteLayouts() Layouts {
	l := wasmtarget.PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "bytes", FixedSize: 8, VariablePayload: true, MaximumPayload: 4096, Encoding: "u32le-offset-u32le-byte-length/opaque"}
	return Layouts{Command: l, Response: l}
}

func typedBytes(size int, fill byte) []byte {
	out := make([]byte, 8+size)
	binary.LittleEndian.PutUint32(out[:4], 8)
	binary.LittleEndian.PutUint32(out[4:8], uint32(size))
	for i := 8; i < len(out); i++ {
		out[i] = fill
	}
	return out
}

type chunkReader struct {
	data    []byte
	maximum int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := len(p)
	if n > r.maximum {
		n = r.maximum
	}
	if n > len(r.data) {
		n = len(r.data)
	}
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

func TestFrameFragmentationCoalescingAndEveryTruncation(t *testing.T) {
	p, l := testProfile(t), byteLayouts()
	first, _ := EncodeFrame(p, l, Frame{Kind: FrameCommand, Payload: typedBytes(3, 7)})
	second, _ := EncodeFrame(p, l, Frame{Kind: FrameResponse, Payload: typedBytes(4, 8)})
	for chunk := 1; chunk <= len(first); chunk++ {
		got, err := ReadFrame(p, l, &chunkReader{data: bytes.Clone(first), maximum: chunk})
		if err != nil || got.Kind != FrameCommand || !bytes.Equal(got.Payload, typedBytes(3, 7)) {
			t.Fatalf("chunk %d: %v", chunk, err)
		}
	}
	stream := bytes.NewReader(append(bytes.Clone(first), second...))
	a, err := ReadFrame(p, l, stream)
	if err != nil || a.Kind != FrameCommand {
		t.Fatal(err)
	}
	b, err := ReadFrame(p, l, stream)
	if err != nil || b.Kind != FrameResponse || stream.Len() != 0 {
		t.Fatal("coalesced")
	}
	for cut := 0; cut < len(first); cut++ {
		if _, err := DecodeFrame(p, l, first[:cut]); err == nil {
			t.Fatalf("accepted truncation %d", cut)
		}
	}
	if _, err := DecodeFrame(p, l, append(bytes.Clone(first), 0)); err == nil {
		t.Fatal("trailing byte accepted")
	}
}

func TestFrameExactMaximumAndMalformedBoundaries(t *testing.T) {
	p, l := testProfile(t), byteLayouts()
	maximum, err := EncodeFrame(p, l, Frame{Kind: FrameCommand, Payload: typedBytes(4075, 1)})
	if err != nil || len(maximum) != 4096 {
		t.Fatalf("maximum %d %v", len(maximum), err)
	}
	if _, err = DecodeFrame(p, l, maximum); err != nil {
		t.Fatal(err)
	}
	tooLong := bytes.Clone(maximum)
	binary.LittleEndian.PutUint32(tooLong[:4], 4093)
	if _, err = ReadFrame(p, l, bytes.NewReader(tooLong)); err == nil {
		t.Fatal("4097-byte declaration accepted")
	}
	for _, mutate := range []func([]byte){func(x []byte) { x[4] = 'X' }, func(x []byte) { x[12] = 2 }, func(x []byte) { binary.LittleEndian.PutUint32(x[17:21], 999) }} {
		x := bytes.Clone(maximum)
		mutate(x)
		if _, err = DecodeFrame(p, l, x); err == nil {
			t.Fatal("malformed accepted")
		}
	}
	forged := p
	forged.MaximumFrameBytes = 8192
	if _, err = DecodeFrame(forged, l, maximum); err == nil {
		t.Fatal("forged profile accepted")
	}
}

type shortWriter struct {
	bytes.Buffer
	maximum int
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) > w.maximum {
		p = p[:w.maximum]
	}
	return w.Buffer.Write(p)
}
func TestWriteFrameHandlesShortWrites(t *testing.T) {
	p, l := testProfile(t), byteLayouts()
	w := &shortWriter{maximum: 1}
	if err := WriteFrame(p, l, w, Frame{Kind: FrameResponse, Payload: typedBytes(9, 3)}); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeFrame(p, l, w.Bytes()); err != nil {
		t.Fatal(err)
	}
}

type memoryPort struct {
	frame                                  []byte
	receiveCalls, sendCalls, evidenceCalls int
	failSend, lie                          bool
	accepted                               map[string][]byte
}

func (p *memoryPort) Receive(r ReceiveRequest) ReceiveOutcome {
	p.receiveCalls++
	return ReceiveOutcome{Frame: bytes.Clone(p.frame)}
}
func (p *memoryPort) Send(r SendRequest) SendOutcome {
	p.sendCalls++
	if p.failSend {
		return SendOutcome{Error: &PortError{Identity: "send.failed", OperationIdentity: r.OperationIdentity}}
	}
	receipt := []byte{byte(p.sendCalls)}
	if p.accepted == nil {
		p.accepted = map[string][]byte{}
	}
	p.accepted[string(receipt)] = bytes.Clone(r.Frame)
	return SendOutcome{Receipt: receipt, AcceptedSHA256: sha256.Sum256(r.Frame)}
}
func (p *memoryPort) Evidence(receipt []byte) ([]byte, bool) {
	p.evidenceCalls++
	b, ok := p.accepted[string(receipt)]
	if p.lie && ok {
		b = append(bytes.Clone(b), 0)
	}
	return bytes.Clone(b), ok
}

type countingHandler struct {
	calls                int
	committed, duplicate bool
}

func (h *countingHandler) Handle(Frame) SemanticResult {
	h.calls++
	return SemanticResult{Response: Frame{Kind: FrameResponse, Payload: typedBytes(2, 9)}, Committed: h.committed, Duplicate: h.duplicate}
}

func TestExecutorAuthorizationEvidenceAndCommittedRetry(t *testing.T) {
	p, l := testProfile(t), byteLayouts()
	command, _ := EncodeFrame(p, l, Frame{Kind: FrameCommand, Payload: typedBytes(1, 4)})
	grants := Grants{p.Receive.Capability: true, p.Send.Capability: true}
	port := &memoryPort{frame: command}
	handler := &countingHandler{committed: true}
	denied := Execute(p, l, Grants{}, port, handler)
	if denied.Failure != "seme.transport.unauthorized" || port.receiveCalls != 0 || handler.calls != 0 {
		t.Fatal("denial performed work")
	}
	ok := Execute(p, l, grants, port, handler)
	if !ok.Committed || !ok.Sent || ok.Failure != "" || handler.calls != 1 || port.receiveCalls != 1 || port.sendCalls != 1 || port.evidenceCalls != 1 || len(ok.Trace) != 2 || ok.Trace[0].Sequence != 0 || ok.Trace[1].Sequence != 1 {
		t.Fatalf("success %#v", ok)
	}
	liar := &memoryPort{frame: command, lie: true}
	bad := Execute(p, l, grants, liar, &countingHandler{committed: true})
	if bad.Failure != "seme.transport.send.evidence" || bad.Retry == nil || !bad.Committed {
		t.Fatal("lying send accepted")
	}
	failing := &memoryPort{frame: command, failSend: true}
	h := &countingHandler{committed: true}
	failed := Execute(p, l, grants, failing, h)
	if failed.Failure != "send.failed" || failed.Retry == nil || !failed.Committed || failed.Sent || h.calls != 1 {
		t.Fatalf("failure %#v", failed)
	}
	failing.failSend = false
	retried := RetrySend(p, grants, failing, *failed.Retry)
	if !retried.Committed || !retried.Sent || retried.Failure != "" || h.calls != 1 || failing.receiveCalls != 1 || failing.sendCalls != 2 {
		t.Fatalf("retry %#v", retried)
	}
	forged := *failed.Retry
	forged.frame = append(forged.frame, 0)
	if got := RetrySend(p, grants, failing, forged); got.Failure != "seme.transport.retry.invalid" {
		t.Fatal("forged retry accepted")
	}
}

func TestExecutorRejectsMalformedPortsAndResponses(t *testing.T) {
	p, l := testProfile(t), byteLayouts()
	command, _ := EncodeFrame(p, l, Frame{Kind: FrameCommand, Payload: typedBytes(1, 4)})
	grants := Grants{p.Receive.Capability: true, p.Send.Capability: true}
	port := &memoryPort{frame: command, failSend: true}
	portResult := Execute(p, l, grants, port, &countingHandler{committed: true})
	if portResult.Failure != "send.failed" {
		t.Fatal(portResult.Failure)
	}
	malformed := &memoryPort{frame: append(command, 0)}
	h := &countingHandler{}
	got := Execute(p, l, grants, malformed, h)
	if got.Failure != "seme.transport.frame.invalid" || h.calls != 0 || malformed.sendCalls != 0 {
		t.Fatal("malformed reached semantic handler")
	}
}
