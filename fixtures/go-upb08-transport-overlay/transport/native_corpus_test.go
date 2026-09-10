package transport

import (
	"crypto/sha256"
	"encoding/json"
	"testing"
)

func TestNativeTransportCorpusIsDeterministic(t *testing.T) {
	build := func() [32]byte {
		hash := sha256.New()
		for index := 0; index < 4096; index++ {
			calls := 0
			sequence := int64(1)
			command := command(t, sequence, byte(index%255+1))
			current := NewState("match")
			if index%11 == 0 {
				command.Sequence = 2
			}
			result := Dispatch(current, command, accepting(index%5, &calls))
			observation := struct {
				Index int
				Calls int
				Value Result
			}{index, calls, result}
			encoded, err := json.Marshal(observation)
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
