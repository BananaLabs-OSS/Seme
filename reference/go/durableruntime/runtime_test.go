package durableruntime

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"seme.local/reference/durableinstance"
)

type testTransform struct{ calls int }

func (x *testTransform) Canonical(p Payload) bool {
	prefix := "v" + strconv.FormatUint(p.Version, 10) + ":"
	return strings.HasPrefix(string(p.Bytes), prefix) && !strings.Contains(string(p.Bytes), "alternate")
}
func (x *testTransform) Prepare(found *Payload) (Payload, *DomainError) {
	x.calls++
	if found == nil {
		return payload(2, "v2:1"), nil
	}
	s := string(found.Bytes)
	switch {
	case found.Version == 1 && s == "v1:valid":
		return payload(2, "v2:1"), nil
	case found.Version == 1:
		return Payload{}, &DomainError{Identity: "domain.migration"}
	case found.Version != 2:
		return Payload{}, &DomainError{Identity: "domain.unsupported_version"}
	case s == "v2:invalid":
		return Payload{}, &DomainError{Identity: "domain.validator"}
	case s == "v2:reject":
		return Payload{}, &DomainError{Identity: "domain.update"}
	case s == "v2:max":
		return Payload{}, &DomainError{Identity: "domain.overflow"}
	}
	n, err := strconv.Atoi(strings.TrimPrefix(s, "v2:"))
	if err != nil {
		return Payload{}, &DomainError{Identity: "domain.decode"}
	}
	return payload(2, "v2:"+strconv.Itoa(n+1)), nil
}

type memoryPort struct {
	stored                        *Payload
	token                         Token
	loadError, casError, conflict bool
	loads, compares               int
	lastCAS                       CompareExchangeRequest
}

func (m *memoryPort) Load(r LoadRequest) LoadOutcome {
	m.loads++
	if m.loadError {
		return LoadOutcome{Variant: LoadError, Error: &PortError{Identity: "port.load", OperationIdentity: profile().Load.Identity}}
	}
	if m.stored == nil {
		return LoadOutcome{Variant: LoadMissing, MissingToken: cloneBytes(m.token)}
	}
	p := clonePayload(*m.stored)
	return LoadOutcome{Variant: LoadFound, Found: &Found{Payload: p, Token: cloneBytes(m.token)}}
}
func (m *memoryPort) CompareExchange(r CompareExchangeRequest) CompareExchangeOutcome {
	m.compares++
	m.lastCAS = cloneCompareRequest(r)
	if m.casError {
		return CompareExchangeOutcome{Variant: CompareExchangeError, Error: &PortError{Identity: "port.cas", OperationIdentity: profile().CompareExchange.Identity}}
	}
	if m.conflict || !bytes.Equal(r.ExpectedToken, m.token) {
		return CompareExchangeOutcome{Variant: CompareExchangeConflict, Token: cloneBytes(m.token)}
	}
	p := clonePayload(r.Payload)
	m.stored = &p
	m.token = Token(fmt.Sprintf("token-%d", m.compares))
	return CompareExchangeOutcome{Variant: CompareExchangeSaved, Token: cloneBytes(m.token)}
}

func profile() Profile {
	return sealProfile(Profile{FamilyIdentity: "family", CurrentVersion: 2, CodecIdentity: "codec", TokenPolicy: "opaque-thread-only", MaximumKeyBytes: 32, MaximumPayloadBytes: 64, Load: Operation{Identity: "load", Capability: "read", Sequence: 0}, CompareExchange: Operation{Identity: "cas", Capability: "write", Sequence: 1}})
}
func grants() Grants { return Grants{"read": true, "write": true} }
func payload(version uint64, text string) Payload {
	b := []byte(text)
	return Payload{Version: version, Bytes: b, SHA256: sha256.Sum256(b)}
}
func request() Request { return Request{Family: "family", Key: "slot"} }

func TestSuccessTracesAndOpaqueTokenPassThrough(t *testing.T) {
	for _, c := range []struct {
		name   string
		stored *Payload
		token  string
		want   string
	}{
		{"missing-create", nil, "opaque-absence", "v2:1"},
		{"migrate-v1", ptr(payload(1, "v1:valid")), "opaque-v1", "v2:1"},
		{"update-v2", ptr(payload(2, "v2:8")), "opaque-v2", "v2:9"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := &memoryPort{stored: c.stored, token: Token(c.token)}
			x := &testTransform{}
			got := Execute(profile(), grants(), request(), m, x)
			if !got.Committed || got.Failure != "" || string(got.Payload.Bytes) != c.want || len(got.Trace) != 2 || got.Trace[0].Sequence != 0 || got.Trace[1].Sequence != 1 || m.loads != 1 || m.compares != 1 {
				t.Fatalf("result=%#v port=%#v", got, m)
			}
			if !bytes.Equal(m.lastCAS.ExpectedToken, []byte(c.token)) {
				t.Fatalf("token changed: %q", m.lastCAS.ExpectedToken)
			}
			if m.stored == nil || !samePayload(*m.stored, got.Payload) {
				t.Fatal("exact committed payload mismatch")
			}
		})
	}
}

func TestLocalAndDomainFailuresNeverCAS(t *testing.T) {
	badDigest := payload(2, "v2:1")
	badDigest.SHA256[0] ^= 1
	cases := []struct {
		name    string
		stored  *Payload
		token   Token
		grants  Grants
		request Request
		want    string
		loads   int
	}{
		{"unauthorized-read", nil, Token("absent"), Grants{"write": true}, request(), "seme.durable.unauthorized", 0},
		{"unauthorized-cas", nil, Token("absent"), Grants{"read": true}, request(), "seme.durable.unauthorized", 0},
		{"bad-key", nil, Token("absent"), grants(), Request{Family: "family"}, "seme.durable.input.invalid", 0},
		{"empty-missing-token", nil, nil, grants(), request(), "seme.durable.load.invalid", 1},
		{"payload-digest", &badDigest, Token("found"), grants(), request(), "seme.durable.load.invalid", 1},
		{"unsupported", ptr(payload(3, "v3:value")), Token("found"), grants(), request(), "domain.unsupported_version", 1},
		{"migration", ptr(payload(1, "v1:bad")), Token("found"), grants(), request(), "domain.migration", 1},
		{"validator", ptr(payload(2, "v2:invalid")), Token("found"), grants(), request(), "domain.validator", 1},
		{"update", ptr(payload(2, "v2:reject")), Token("found"), grants(), request(), "domain.update", 1},
		{"overflow", ptr(payload(2, "v2:max")), Token("found"), grants(), request(), "domain.overflow", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := &memoryPort{stored: c.stored, token: c.token}
			before := clonePayloadPtr(m.stored)
			got := Execute(profile(), c.grants, c.request, m, &testTransform{})
			if got.Committed || got.Failure != c.want || m.loads != c.loads || m.compares != 0 || !samePayloadPtr(before, m.stored) {
				t.Fatalf("got=%#v port=%#v", got, m)
			}
		})
	}
}

func TestPortFailureAndConflictAreOneAttemptAndAtomic(t *testing.T) {
	for _, c := range []struct {
		name                   string
		load, cas, conflict    bool
		want                   string
		trace, loads, compares int
	}{
		{"load", true, false, false, "port.load", 1, 1, 0},
		{"cas", false, true, false, "port.cas", 2, 1, 1},
		{"conflict", false, false, true, "seme.durable.compare_exchange.conflict", 2, 1, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			initial := payload(2, "v2:2")
			m := &memoryPort{stored: &initial, token: Token("opaque"), loadError: c.load, casError: c.cas, conflict: c.conflict}
			before := clonePayloadPtr(m.stored)
			got := Execute(profile(), grants(), request(), m, &testTransform{})
			if got.Committed || got.Failure != c.want || len(got.Trace) != c.trace || m.loads != c.loads || m.compares != c.compares || !samePayloadPtr(before, m.stored) {
				t.Fatalf("got=%#v port=%#v", got, m)
			}
		})
	}
}

func TestMalformedLoadVariantRejects(t *testing.T) {
	t.Run("load-mixed-variant", func(t *testing.T) {
		p := &mixedLoadPort{}
		got := Execute(profile(), grants(), request(), p, &testTransform{})
		if got.Committed || got.Failure != "seme.durable.load.invalid" || len(got.Trace) != 1 {
			t.Fatalf("%#v", got)
		}
	})
}

func TestProfileCannotBeCallerForgedOrMutated(t *testing.T) {
	forged := Profile{FamilyIdentity: "family", CurrentVersion: 2, CodecIdentity: "codec", TokenPolicy: "opaque-thread-only", MaximumKeyBytes: 32, MaximumPayloadBytes: 64, Load: Operation{Identity: "load", Capability: "read", Sequence: 0}, CompareExchange: Operation{Identity: "cas", Capability: "write", Sequence: 1}}
	for _, p := range []Profile{forged, func() Profile { p := profile(); p.MaximumPayloadBytes++; return p }()} {
		m := &memoryPort{token: Token("absence")}
		got := Execute(p, grants(), request(), m, &testTransform{})
		if got.Failure != "seme.durable.profile.invalid" || m.loads != 0 || m.compares != 0 {
			t.Fatalf("forged profile gained authority: %#v", got)
		}
	}
	if _, err := AuthenticatedProfile(durableinstance.Inputs{}); err == nil {
		t.Fatal("unauthenticated instance produced runtime profile")
	}
	if got, err := ExecuteAuthenticated(durableinstance.Inputs{}, grants(), request(), &memoryPort{token: Token("absence")}, &testTransform{}); err == nil || got.Committed || len(got.Trace) != 0 {
		t.Fatal("unauthenticated instance reached host executor")
	}
}

func TestPortAndTransformerAliasesCannotRewriteTraceOrToken(t *testing.T) {
	p := &aliasPort{token: Token("opaque-original")}
	x := &aliasTransform{}
	got := Execute(profile(), grants(), request(), p, x)
	if !got.Committed || string(got.Trace[1].CompareRequest.ExpectedToken) != "opaque-original" || string(got.Payload.Bytes) != "v2:4" || p.stored == nil || !samePayload(*p.stored, got.Payload) {
		t.Fatalf("alias escaped: %#v", got)
	}
	p.seen.ExpectedToken[0] = 'X'
	p.seen.Payload.Bytes[0] = 'X'
	if string(got.Trace[1].CompareRequest.ExpectedToken) != "opaque-original" || string(got.Payload.Bytes) != "v2:4" {
		t.Fatal("retained provider alias rewrote result")
	}
}

type aliasTransform struct{}

func (*aliasTransform) Canonical(p Payload) bool { return strings.HasPrefix(string(p.Bytes), "v") }
func (*aliasTransform) Prepare(found *Payload) (Payload, *DomainError) {
	if found != nil && len(found.Bytes) != 0 {
		found.Bytes[0] = 'X'
	}
	return payload(2, "v2:4"), nil
}

type aliasPort struct {
	token  Token
	seen   CompareExchangeRequest
	stored *Payload
}

func (p *aliasPort) Load(LoadRequest) LoadOutcome {
	return LoadOutcome{Variant: LoadMissing, MissingToken: p.token}
}
func (p *aliasPort) CompareExchange(r CompareExchangeRequest) CompareExchangeOutcome {
	p.seen = r
	committed := clonePayload(r.Payload)
	p.stored = &committed
	return CompareExchangeOutcome{Variant: CompareExchangeSaved, Token: Token("next")}
}

type mixedLoadPort struct{}

func (*mixedLoadPort) Load(LoadRequest) LoadOutcome {
	p := payload(2, "v2:1")
	return LoadOutcome{Variant: LoadMissing, MissingToken: Token("a"), Found: &Found{Payload: p, Token: Token("b")}}
}
func (*mixedLoadPort) CompareExchange(CompareExchangeRequest) CompareExchangeOutcome {
	panic("must not call")
}

func ptr[T any](x T) *T { return &x }
func clonePayloadPtr(x *Payload) *Payload {
	if x == nil {
		return nil
	}
	v := clonePayload(*x)
	return &v
}
func samePayloadPtr(a, b *Payload) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return samePayload(*a, *b)
}
