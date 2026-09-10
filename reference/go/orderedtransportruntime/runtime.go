package orderedtransportruntime

import (
	"bytes"
	"crypto/sha256"
)

type Grants map[string]bool
type PortError struct{ Identity, OperationIdentity string }
type ReceiveRequest struct{ OperationIdentity string }
type ReceiveOutcome struct {
	Frame []byte
	Error *PortError
}
type SendRequest struct {
	OperationIdentity string
	Frame             []byte
}
type SendOutcome struct {
	Receipt        []byte
	AcceptedSHA256 [sha256.Size]byte
	Error          *PortError
}

// Port's Evidence method is a logically independent read-back of bytes named
// by an opaque send receipt. A successful Send claim is never trusted alone.
type Port interface {
	Receive(ReceiveRequest) ReceiveOutcome
	Send(SendRequest) SendOutcome
	Evidence(receipt []byte) ([]byte, bool)
}

type SemanticResult struct {
	Response  Frame
	Committed bool
	Duplicate bool
	Failure   string
}
type Handler interface{ Handle(Frame) SemanticResult }

type Observation struct {
	Sequence  uint64
	Operation string
	Received  *ReceiveOutcome
	Sent      *SendOutcome
}
type Result struct {
	Committed, Duplicate, Sent bool
	Failure                    string
	ResponseFrame              []byte
	Retry                      *Retry
	Trace                      []Observation
}
type Retry struct {
	frame          []byte
	digest         [sha256.Size]byte
	authentication [sha256.Size]byte
}

func Execute(profile Profile, layouts Layouts, grants Grants, port Port, handler Handler) Result {
	fail := func(id string, trace []Observation) Result { return Result{Failure: id, Trace: trace} }
	if !validProfile(profile) || port == nil || handler == nil {
		return fail("seme.transport.profile.invalid", nil)
	}
	// Both capabilities are authorized before Receive so denial has no host trace.
	if !grants[profile.Receive.Capability] || !grants[profile.Send.Capability] {
		return fail("seme.transport.unauthorized", nil)
	}
	received := cloneReceive(port.Receive(ReceiveRequest{OperationIdentity: profile.Receive.Identity}))
	trace := []Observation{{Sequence: profile.Receive.Sequence, Operation: profile.Receive.Identity, Received: &received}}
	if received.Error != nil {
		if len(received.Frame) == 0 && validPortError(received.Error, profile.Receive.Identity) {
			return fail(received.Error.Identity, trace)
		}
		return fail("seme.transport.receive.invalid", trace)
	}
	frame, err := DecodeFrame(profile, layouts, received.Frame)
	if err != nil || frame.Kind != FrameCommand {
		return fail("seme.transport.frame.invalid", trace)
	}
	semantic := handler.Handle(frame)
	if semantic.Failure != "" {
		if semantic.Committed || semantic.Duplicate || semantic.Response.Kind != 0 || len(semantic.Response.Payload) != 0 {
			return Result{Failure: "seme.transport.semantic.invalid", Trace: trace}
		}
		return Result{Committed: semantic.Committed, Duplicate: semantic.Duplicate, Failure: semantic.Failure, Trace: trace}
	}
	if semantic.Committed == semantic.Duplicate {
		return Result{Failure: "seme.transport.semantic.invalid", Trace: trace}
	}
	if semantic.Response.Kind != FrameResponse {
		return Result{Committed: semantic.Committed, Duplicate: semantic.Duplicate, Failure: "seme.transport.response.invalid", Trace: trace}
	}
	response, err := EncodeFrame(profile, layouts, semantic.Response)
	if err != nil {
		return Result{Committed: semantic.Committed, Duplicate: semantic.Duplicate, Failure: "seme.transport.response.invalid", Trace: trace}
	}
	sent := cloneSend(port.Send(SendRequest{OperationIdentity: profile.Send.Identity, Frame: bytes.Clone(response)}))
	trace = append(trace, Observation{Sequence: profile.Send.Sequence, Operation: profile.Send.Identity, Sent: &sent})
	base := Result{Committed: semantic.Committed, Duplicate: semantic.Duplicate, ResponseFrame: bytes.Clone(response), Trace: trace}
	if sent.Error != nil {
		if len(sent.Receipt) == 0 && sent.AcceptedSHA256 == ([sha256.Size]byte{}) && validPortError(sent.Error, profile.Send.Identity) {
			base.Failure = sent.Error.Identity
		} else {
			base.Failure = "seme.transport.send.invalid"
		}
		if semantic.Committed {
			base.Retry = retryFor(profile, response)
		}
		return base
	}
	if len(sent.Receipt) == 0 || sent.AcceptedSHA256 != sha256.Sum256(response) {
		base.Failure = "seme.transport.send.invalid"
		if semantic.Committed {
			base.Retry = retryFor(profile, response)
		}
		return base
	}
	evidence, ok := port.Evidence(bytes.Clone(sent.Receipt))
	if !ok || !bytes.Equal(evidence, response) {
		base.Failure = "seme.transport.send.evidence"
		if semantic.Committed {
			base.Retry = retryFor(profile, response)
		}
		return base
	}
	base.Sent = true
	return base
}

// RetrySend retries only the exact response retained after a committed
// semantic transition. It performs no Receive and cannot invoke the handler.
func RetrySend(profile Profile, grants Grants, port Port, retry Retry) Result {
	if !validProfile(profile) || port == nil || retry.authentication != retryDigest(profile, retry.frame, retry.digest) || retry.digest != sha256.Sum256(retry.frame) {
		return Result{Failure: "seme.transport.retry.invalid"}
	}
	if !grants[profile.Send.Capability] {
		return Result{Committed: true, Failure: "seme.transport.unauthorized", ResponseFrame: bytes.Clone(retry.frame), Retry: &retry}
	}
	sent := cloneSend(port.Send(SendRequest{OperationIdentity: profile.Send.Identity, Frame: bytes.Clone(retry.frame)}))
	trace := []Observation{{Sequence: profile.Send.Sequence, Operation: profile.Send.Identity, Sent: &sent}}
	result := Result{Committed: true, ResponseFrame: bytes.Clone(retry.frame), Trace: trace}
	if sent.Error != nil {
		if len(sent.Receipt) == 0 && sent.AcceptedSHA256 == ([sha256.Size]byte{}) && validPortError(sent.Error, profile.Send.Identity) {
			result.Failure = sent.Error.Identity
		} else {
			result.Failure = "seme.transport.send.invalid"
		}
		result.Retry = &retry
		return result
	}
	if len(sent.Receipt) == 0 || sent.AcceptedSHA256 != retry.digest {
		result.Failure = "seme.transport.send.invalid"
		result.Retry = &retry
		return result
	}
	evidence, ok := port.Evidence(bytes.Clone(sent.Receipt))
	if !ok || !bytes.Equal(evidence, retry.frame) {
		result.Failure = "seme.transport.send.evidence"
		result.Retry = &retry
		return result
	}
	result.Sent = true
	return result
}

func retryFor(p Profile, frame []byte) *Retry {
	d := sha256.Sum256(frame)
	r := Retry{frame: bytes.Clone(frame), digest: d}
	r.authentication = retryDigest(p, r.frame, d)
	return &r
}
func retryDigest(p Profile, frame []byte, d [sha256.Size]byte) [sha256.Size]byte {
	h := sha256.New()
	h.Write([]byte("seme.ordered-transport.retry.v1\x00"))
	pd := profileDigest(p)
	h.Write(pd[:])
	h.Write(d[:])
	h.Write(frame)
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out
}
func validPortError(e *PortError, operation string) bool {
	return e != nil && e.Identity != "" && e.OperationIdentity == operation
}
func cloneReceive(v ReceiveOutcome) ReceiveOutcome {
	v.Frame = bytes.Clone(v.Frame)
	if v.Error != nil {
		x := *v.Error
		v.Error = &x
	}
	return v
}
func cloneSend(v SendOutcome) SendOutcome {
	v.Receipt = bytes.Clone(v.Receipt)
	if v.Error != nil {
		x := *v.Error
		v.Error = &x
	}
	return v
}
