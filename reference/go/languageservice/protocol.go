package languageservice

import (
	"crypto/sha256"
	"errors"
)

const (
	LiftAcceptedValid uint64 = iota
	LiftAcceptedInvalid
	LiftRejectedStale
)

type DocumentState struct {
	Document          string
	ClientRevision    uint64
	ContentDigest     [32]byte
	LastValidRevision []byte
}

type UpdateRequest struct {
	Document       string
	ClientRevision uint64
	ContentDigest  [32]byte
	Content        []byte
}

type Diagnostic struct {
	Severity      uint64
	Code, Message string
	Start, End    *uint64
}
type SourceMapping struct {
	SemanticIdentity [16]byte
	Start, End, Role uint64
}
type LiftEvidence struct {
	CanonicalRevision []byte
	Diagnostics       []Diagnostic
	SourceMappings    []SourceMapping
}
type LiftResult struct {
	Document          string
	ClientRevision    uint64
	ContentDigest     [32]byte
	Disposition       uint64
	CanonicalRevision []byte
	LastValidRevision []byte
	Diagnostics       []Diagnostic
	SourceMappings    []SourceMapping
}

var ErrDigestMismatch = errors.New("language_service.content_digest_mismatch")
var ErrDocumentMismatch = errors.New("language_service.document_mismatch")
var ErrInvalidLiftEvidence = errors.New("language_service.invalid_lift_evidence")

// ApplyUpdate rejects stale updates without invoking lift. Accepted invalid
// content advances the client revision while preserving last-valid meaning.
func ApplyUpdate(current DocumentState, request UpdateRequest, lift func([]byte) LiftEvidence) (DocumentState, LiftResult, error) {
	result := LiftResult{Document: request.Document, ClientRevision: request.ClientRevision, ContentDigest: request.ContentDigest}
	if request.Document != current.Document {
		return current, LiftResult{}, ErrDocumentMismatch
	}
	if request.ClientRevision <= current.ClientRevision {
		result.Disposition = LiftRejectedStale
		result.LastValidRevision = clone(current.LastValidRevision)
		return current, result, nil
	}
	if sha256.Sum256(request.Content) != request.ContentDigest {
		return current, LiftResult{}, ErrDigestMismatch
	}
	evidence := lift(request.Content)
	if !validEvidence(evidence, uint64(len(request.Content))) {
		return current, LiftResult{}, ErrInvalidLiftEvidence
	}
	next := DocumentState{Document: current.Document, ClientRevision: request.ClientRevision, ContentDigest: request.ContentDigest, LastValidRevision: clone(current.LastValidRevision)}
	result.Diagnostics = append([]Diagnostic(nil), evidence.Diagnostics...)
	result.SourceMappings = append([]SourceMapping(nil), evidence.SourceMappings...)
	if len(evidence.CanonicalRevision) == 16 {
		result.Disposition = LiftAcceptedValid
		result.CanonicalRevision = clone(evidence.CanonicalRevision)
		next.LastValidRevision = clone(evidence.CanonicalRevision)
	} else {
		result.Disposition = LiftAcceptedInvalid
	}
	result.LastValidRevision = clone(next.LastValidRevision)
	return next, result, nil
}

func clone(value []byte) []byte { return append([]byte(nil), value...) }

func validEvidence(evidence LiftEvidence, contentSize uint64) bool {
	if len(evidence.CanonicalRevision) != 0 && len(evidence.CanonicalRevision) != 16 {
		return false
	}
	for _, diagnostic := range evidence.Diagnostics {
		if (diagnostic.Start == nil) != (diagnostic.End == nil) {
			return false
		}
		if diagnostic.Start != nil && (*diagnostic.Start > *diagnostic.End || *diagnostic.End > contentSize) {
			return false
		}
	}
	for _, mapping := range evidence.SourceMappings {
		if mapping.Start > mapping.End || mapping.End > contentSize {
			return false
		}
	}
	return true
}
