// Package durableruntime enforces the neutral Durable-State-v1 host boundary.
// It performs no filesystem, database, clock, retry, or transaction work.
package durableruntime

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"unicode/utf8"

	"seme.local/reference/durableinstance"
)

type Operation struct {
	Identity, Capability string
	Sequence             uint64
}

type Profile struct {
	FamilyIdentity             string
	CurrentVersion             uint64
	CodecIdentity, TokenPolicy string
	MaximumKeyBytes            uint64
	MaximumPayloadBytes        uint64
	Load, CompareExchange      Operation
	authentication             [sha256.Size]byte
}

type Grants map[string]bool
type Token []byte
type Payload struct {
	Version uint64
	Bytes   []byte
	SHA256  [sha256.Size]byte
}
type LogicalKey struct{ Value string }
type LoadRequest struct {
	Family string
	Key    LogicalKey
}
type Found struct {
	Payload Payload
	Token   Token
}

type LoadVariant uint8

const (
	LoadMissing LoadVariant = iota
	LoadFound
	LoadError
)

type PortError struct{ Identity, OperationIdentity string }
type LoadOutcome struct {
	Variant      LoadVariant
	MissingToken Token
	Found        *Found
	Error        *PortError
}

type CompareExchangeRequest struct {
	Family        string
	Key           LogicalKey
	ExpectedToken Token
	Payload       Payload
}
type CompareExchangeVariant uint8

const (
	CompareExchangeSaved CompareExchangeVariant = iota
	CompareExchangeConflict
	CompareExchangeError
)

type CompareExchangeOutcome struct {
	Variant   CompareExchangeVariant
	Token     Token
	Committed *Payload
	Error     *PortError
}

type Port interface {
	Load(LoadRequest) LoadOutcome
	CompareExchange(CompareExchangeRequest) CompareExchangeOutcome
}

// Transformer owns project semantics: decode, validate, migrate, update,
// validate, and canonical encode. Canonical must reject alternate encodings.
type Transformer interface {
	Prepare(found *Payload) (Payload, *DomainError)
	Canonical(Payload) bool
}
type DomainError struct{ Identity string }

type Observation struct {
	Sequence       uint64
	Operation      string
	LoadRequest    *LoadRequest
	LoadOutcome    *LoadOutcome
	CompareRequest *CompareExchangeRequest
	CompareOutcome *CompareExchangeOutcome
}
type Result struct {
	Committed bool
	Payload   Payload
	Token     Token
	Failure   string
	Trace     []Observation
}
type Request struct {
	Family string
	Key    string
}

// ExecuteAuthenticated is the convenience path from authenticated
// Durable-State-v1 metadata into the host boundary. It derives the sealed
// profile from the exact instance before delegating to Execute; it does not
// assert that transform implements the declared codec or domain semantics.
func ExecuteAuthenticated(in durableinstance.Inputs, grants Grants, request Request, port Port, transform Transformer) (Result, error) {
	profile, err := AuthenticatedProfile(in)
	if err != nil {
		return Result{}, err
	}
	return Execute(profile, grants, request, port, transform), nil
}

func Execute(profile Profile, grants Grants, request Request, port Port, transform Transformer) Result {
	fail := func(identity string, trace []Observation) Result { return Result{Failure: identity, Trace: trace} }
	if !validProfile(profile) || port == nil || transform == nil {
		return fail("seme.durable.profile.invalid", nil)
	}
	if !grants[profile.Load.Capability] || !grants[profile.CompareExchange.Capability] {
		return fail("seme.durable.unauthorized", nil)
	}
	if request.Family != profile.FamilyIdentity || request.Key == "" || !utf8.ValidString(request.Key) || uint64(len([]byte(request.Key))) > profile.MaximumKeyBytes {
		return fail("seme.durable.input.invalid", nil)
	}
	loadRequest := LoadRequest{Family: request.Family, Key: LogicalKey{Value: request.Key}}
	loaded := cloneLoadOutcome(port.Load(loadRequest))
	trace := []Observation{{Sequence: profile.Load.Sequence, Operation: profile.Load.Identity, LoadRequest: &loadRequest, LoadOutcome: &loaded}}
	var found *Payload
	var expected Token
	switch loaded.Variant {
	case LoadMissing:
		if len(loaded.MissingToken) == 0 || loaded.Found != nil || loaded.Error != nil {
			return fail("seme.durable.load.invalid", trace)
		}
		expected = cloneBytes(loaded.MissingToken)
	case LoadFound:
		if len(loaded.MissingToken) != 0 || loaded.Found == nil || loaded.Error != nil || len(loaded.Found.Token) == 0 || !validPayload(profile, loaded.Found.Payload, transform) {
			return fail("seme.durable.load.invalid", trace)
		}
		copyFound := cloneFound(*loaded.Found)
		copyPayload := clonePayload(copyFound.Payload)
		found = &copyPayload
		expected = cloneBytes(copyFound.Token)
	case LoadError:
		if len(loaded.MissingToken) != 0 || loaded.Found != nil || !validPortError(loaded.Error, profile.Load.Identity) {
			return fail("seme.durable.load.invalid", trace)
		}
		return fail(loaded.Error.Identity, trace)
	default:
		return fail("seme.durable.load.invalid", trace)
	}
	next, domainFailure := transform.Prepare(found)
	if domainFailure != nil {
		if domainFailure.Identity == "" {
			return fail("seme.durable.domain.invalid_error", trace)
		}
		return fail(domainFailure.Identity, trace)
	}
	if next.Version != profile.CurrentVersion || !validPayload(profile, next, transform) {
		return fail("seme.durable.payload.invalid", trace)
	}
	casRequest := CompareExchangeRequest{Family: request.Family, Key: LogicalKey{Value: request.Key}, ExpectedToken: cloneBytes(expected), Payload: clonePayload(next)}
	cas := cloneCompareOutcome(port.CompareExchange(cloneCompareRequest(casRequest)))
	trace = append(trace, Observation{Sequence: profile.CompareExchange.Sequence, Operation: profile.CompareExchange.Identity, CompareRequest: &casRequest, CompareOutcome: &cas})
	switch cas.Variant {
	case CompareExchangeSaved:
		if len(cas.Token) == 0 || cas.Error != nil || cas.Committed == nil || !samePayload(*cas.Committed, next) {
			return fail("seme.durable.compare_exchange.invalid", trace)
		}
		return Result{Committed: true, Payload: clonePayload(next), Token: cloneBytes(cas.Token), Trace: trace}
	case CompareExchangeConflict:
		if len(cas.Token) == 0 || cas.Error != nil || cas.Committed != nil {
			return fail("seme.durable.compare_exchange.invalid", trace)
		}
		return fail("seme.durable.compare_exchange.conflict", trace)
	case CompareExchangeError:
		if len(cas.Token) != 0 || cas.Committed != nil || !validPortError(cas.Error, profile.CompareExchange.Identity) {
			return fail("seme.durable.compare_exchange.invalid", trace)
		}
		return fail(cas.Error.Identity, trace)
	default:
		return fail("seme.durable.compare_exchange.invalid", trace)
	}
}

func validProfile(p Profile) bool {
	return p.authentication == profileDigest(p) && p.FamilyIdentity != "" && p.CurrentVersion != 0 && p.CodecIdentity != "" && p.TokenPolicy == "opaque-thread-only" && p.MaximumKeyBytes > 0 && p.MaximumPayloadBytes > 0 && p.Load.Identity != "" && p.CompareExchange.Identity != "" && p.Load.Identity != p.CompareExchange.Identity && p.Load.Capability != "" && p.CompareExchange.Capability != "" && p.Load.Capability != p.CompareExchange.Capability && p.Load.Sequence == 0 && p.CompareExchange.Sequence == 1
}
func sealProfile(p Profile) Profile { p.authentication = profileDigest(p); return p }
func profileDigest(p Profile) [sha256.Size]byte {
	h := sha256.New()
	fmt.Fprintf(h, "seme.durable-runtime.profile.v1\x00%q\x00%d\x00%q\x00%q\x00%d\x00%d\x00%q\x00%q\x00%d\x00%q\x00%q\x00%d", p.FamilyIdentity, p.CurrentVersion, p.CodecIdentity, p.TokenPolicy, p.MaximumKeyBytes, p.MaximumPayloadBytes, p.Load.Identity, p.Load.Capability, p.Load.Sequence, p.CompareExchange.Identity, p.CompareExchange.Capability, p.CompareExchange.Sequence)
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out
}
func validPayload(p Profile, v Payload, t Transformer) bool {
	return v.Version != 0 && len(v.Bytes) > 0 && uint64(len(v.Bytes)) <= p.MaximumPayloadBytes && sha256.Sum256(v.Bytes) == v.SHA256 && t.Canonical(clonePayload(v))
}
func validPortError(e *PortError, operation string) bool {
	return e != nil && e.Identity != "" && e.OperationIdentity == operation
}
func samePayload(a, b Payload) bool {
	return a.Version == b.Version && a.SHA256 == b.SHA256 && bytes.Equal(a.Bytes, b.Bytes)
}
func clonePayload(v Payload) Payload { v.Bytes = cloneBytes(v.Bytes); return v }
func cloneFound(v Found) Found {
	v.Payload = clonePayload(v.Payload)
	v.Token = cloneBytes(v.Token)
	return v
}
func cloneCompareRequest(v CompareExchangeRequest) CompareExchangeRequest {
	v.ExpectedToken = cloneBytes(v.ExpectedToken)
	v.Payload = clonePayload(v.Payload)
	return v
}
func cloneLoadOutcome(v LoadOutcome) LoadOutcome {
	v.MissingToken = cloneBytes(v.MissingToken)
	if v.Found != nil {
		x := cloneFound(*v.Found)
		v.Found = &x
	}
	if v.Error != nil {
		x := *v.Error
		v.Error = &x
	}
	return v
}
func cloneCompareOutcome(v CompareExchangeOutcome) CompareExchangeOutcome {
	v.Token = cloneBytes(v.Token)
	if v.Committed != nil {
		x := clonePayload(*v.Committed)
		v.Committed = &x
	}
	if v.Error != nil {
		x := *v.Error
		v.Error = &x
	}
	return v
}
func cloneBytes[T ~[]byte](v T) T { return append(T(nil), v...) }
