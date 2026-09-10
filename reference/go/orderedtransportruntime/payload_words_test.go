package orderedtransportruntime

import (
	"bytes"
	"math"
	"testing"
)

func TestPayloadWordsExactLittleEndianSignedRoundTrip(t *testing.T) {
	profile := testProfile(t)
	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x88, 0x09}
	encoded, retained, err := PackCanonicalPayload(profile, PayloadRetention{}, payload)
	if err != nil {
		t.Fatal(err)
	}
	if encoded.ByteLength != 9 || len(encoded.Words) != 2 || encoded.Words[0] >= 0 || retained != (PayloadRetention{Bytes: 9, Words: 2}) {
		t.Fatalf("encoded=%#v retained=%#v", encoded, retained)
	}
	payload[0] = 0xff
	decoded, err := UnpackCanonicalPayload(profile, encoded)
	if err != nil || !bytes.Equal(decoded, []byte{1, 2, 3, 4, 5, 6, 7, 0x88, 9}) {
		t.Fatalf("decoded=%x err=%v", decoded, err)
	}
	encoded.Words[0] = 0
	if decoded[0] != 1 {
		t.Fatal("decoded bytes alias encoded words")
	}
}

func TestPayloadWordsExactBoundsAndPadding(t *testing.T) {
	profile := testProfile(t)
	maximum := bytes.Repeat([]byte{0xa5}, int(profile.MaximumPayloadBytes))
	encoded, retained, err := PackCanonicalPayload(profile, PayloadRetention{}, maximum)
	if err != nil || retained != (PayloadRetention{Bytes: 3072, Words: 384}) {
		t.Fatalf("maximum: %#v %v", retained, err)
	}
	if decoded, err := UnpackCanonicalPayload(profile, encoded); err != nil || !bytes.Equal(decoded, maximum) {
		t.Fatal("maximum did not round trip")
	}
	if _, _, err := PackCanonicalPayload(profile, PayloadRetention{}, append(maximum, 0)); err == nil {
		t.Fatal("3073-byte payload accepted")
	}
	if _, err := UnpackCanonicalPayload(profile, PayloadWords{Words: []int64{1, 2}, ByteLength: 8}); err == nil {
		t.Fatal("word mismatch accepted")
	}
	if _, err := UnpackCanonicalPayload(profile, PayloadWords{Words: []int64{0x100}, ByteLength: 1}); err == nil {
		t.Fatal("nonzero padding accepted")
	}
}

func TestPayloadWordsCumulativeBytesAndPaddedWords(t *testing.T) {
	profile := testProfile(t)
	_, retained, err := PackCanonicalPayload(profile, PayloadRetention{}, make([]byte, 3072))
	if err != nil {
		t.Fatal(err)
	}
	_, exact, err := PackCanonicalPayload(profile, retained, make([]byte, 1024))
	if err != nil || exact != (PayloadRetention{Bytes: 4096, Words: 512}) {
		t.Fatalf("exact retention=%#v %v", exact, err)
	}
	if _, next, err := PackCanonicalPayload(profile, exact, []byte{1}); err == nil || next != exact {
		t.Fatal("retained-byte overflow was not atomic")
	}
	// Per-command padding makes 3071+1025 bytes occupy 513 words even though
	// the meaningful-byte sum is exactly 4096.
	_, first, err := PackCanonicalPayload(profile, PayloadRetention{}, make([]byte, 3071))
	if err != nil {
		t.Fatal(err)
	}
	if _, next, err := PackCanonicalPayload(profile, first, make([]byte, 1025)); err == nil || next != first {
		t.Fatal("retained-word overflow was not atomic")
	}
}

func TestPayloadWordsRejectsForgedProfileAndRetentionOverflow(t *testing.T) {
	profile := testProfile(t)
	forged := profile
	forged.MaximumPayloadBytes++
	if _, _, err := PackCanonicalPayload(forged, PayloadRetention{}, nil); err == nil {
		t.Fatal("forged profile accepted")
	}
	if _, _, err := PackCanonicalPayload(profile, PayloadRetention{Bytes: math.MaxUint64, Words: math.MaxUint64}, []byte{1}); err == nil {
		t.Fatal("overflowing retention accepted")
	}
}
