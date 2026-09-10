package orderedtransportruntime

import (
	"encoding/binary"
	"fmt"
)

// PayloadWords is an exact, project-neutral projection of canonical bytes for
// languages whose bounded aggregate representation is a slice of signed i64.
// ByteLength distinguishes meaningful bytes from canonical zero padding.
type PayloadWords struct {
	Words      []int64
	ByteLength uint64
}

// PayloadRetention tracks both meaningful bytes and padded words. Tracking
// both prevents many short payloads from bypassing the retained representation
// bound through per-command padding.
type PayloadRetention struct {
	Bytes uint64
	Words uint64
}

// PackCanonicalPayload copies payload into little-endian signed words and
// returns the authenticated cumulative retention. It performs no semantic
// interpretation of the payload.
func PackCanonicalPayload(profile Profile, retained PayloadRetention, payload []byte) (PayloadWords, PayloadRetention, error) {
	if !validProfile(profile) {
		return PayloadWords{}, retained, fmt.Errorf("ordered_transport.payload_profile")
	}
	maximumWords := wordsFor(profile.MaximumRetainedPayloadBytes)
	if retained.Bytes > profile.MaximumRetainedPayloadBytes || retained.Words > maximumWords {
		return PayloadWords{}, retained, fmt.Errorf("ordered_transport.payload_retention")
	}
	length := uint64(len(payload))
	if length > profile.MaximumPayloadBytes {
		return PayloadWords{}, retained, fmt.Errorf("ordered_transport.payload_length")
	}
	wordCount := wordsFor(length)
	if length > ^uint64(0)-retained.Bytes || wordCount > ^uint64(0)-retained.Words {
		return PayloadWords{}, retained, fmt.Errorf("ordered_transport.payload_overflow")
	}
	next := PayloadRetention{Bytes: retained.Bytes + length, Words: retained.Words + wordCount}
	if next.Bytes > profile.MaximumRetainedPayloadBytes || next.Words > maximumWords {
		return PayloadWords{}, retained, fmt.Errorf("ordered_transport.payload_retention")
	}
	words := make([]int64, wordCount)
	for index := range words {
		var padded [8]byte
		start := index * 8
		end := start + 8
		if end > len(payload) {
			end = len(payload)
		}
		copy(padded[:], payload[start:end])
		words[index] = int64(binary.LittleEndian.Uint64(padded[:]))
	}
	return PayloadWords{Words: words, ByteLength: length}, next, nil
}

// UnpackCanonicalPayload validates and copies one exact word projection.
func UnpackCanonicalPayload(profile Profile, encoded PayloadWords) ([]byte, error) {
	if !validProfile(profile) {
		return nil, fmt.Errorf("ordered_transport.payload_profile")
	}
	if encoded.ByteLength > profile.MaximumPayloadBytes || uint64(len(encoded.Words)) != wordsFor(encoded.ByteLength) {
		return nil, fmt.Errorf("ordered_transport.payload_shape")
	}
	bytes := make([]byte, len(encoded.Words)*8)
	for index, word := range encoded.Words {
		binary.LittleEndian.PutUint64(bytes[index*8:], uint64(word))
	}
	for _, value := range bytes[encoded.ByteLength:] {
		if value != 0 {
			return nil, fmt.Errorf("ordered_transport.payload_padding")
		}
	}
	return append([]byte(nil), bytes[:encoded.ByteLength]...), nil
}

func wordsFor(bytes uint64) uint64 {
	words := bytes / 8
	if bytes%8 != 0 {
		words++
	}
	return words
}
