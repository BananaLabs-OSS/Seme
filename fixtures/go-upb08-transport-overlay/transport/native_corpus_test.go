package transport

import (
	"crypto/sha256"
	"encoding/json"
	"testing"
)

func TestNativeTransportCorpusHas4096DeterministicObservations(t *testing.T) {
	build := func() [32]byte {
		hash := sha256.New()
		for index := int64(0); index < 4096; index++ {
			current := NewState("match")
			value := command(1, index+1)
			if index%11 == 0 {
				value.Sequence = 2
			}
			result := Commit(current, value, index%3 != 0, map[bool]int64{true: 0, false: 71}[index%3 != 0], index+1, index%5)
			encoded, err := json.Marshal(struct {
				Index int64
				Value Result
			}{index, result})
			if err != nil {
				t.Fatal(err)
			}
			hash.Write(encoded)
			hash.Write([]byte{'\n'})
		}
		var result [32]byte
		copy(result[:], hash.Sum(nil))
		return result
	}
	first, second := build(), build()
	if first != second {
		t.Fatalf("corpus changed: %x != %x", first, second)
	}
}
