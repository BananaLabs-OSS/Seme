package controlled

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"testing"
)

var upb12NativeOutput = flag.String("upb12-native-output", "", "optional canonical JSONL observation output")

func TestUPB12NativeCorpus(t *testing.T) {
	oldWriter, oldFlags := log.Writer(), log.Flags()
	defer func() { log.SetOutput(oldWriter); log.SetFlags(oldFlags) }()
	var output *os.File
	if *upb12NativeOutput != "" {
		var err error
		output, err = os.OpenFile(*upb12NativeOutput, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		defer output.Close()
	}
	var trace bytes.Buffer
	log.SetOutput(&trace)
	log.SetFlags(0)
	for i := int64(0); i < 4096; i++ {
		state, delta := i*7919-10000000, i*104729-200000000
		if i%257 == 0 {
			state, delta = math.MaxInt64, 1
		}
		trace.Reset()
		got := DispatchControlled(ControlledState{Value: state}, ControlledCommand{Delta: delta})
		if got.Value != state+delta || trace.String() != "true\n" {
			t.Fatalf("observation %d: value=%d trace=%q", i, got.Value, trace.String())
		}
		if output != nil {
			fmt.Fprintf(output, "{\"value\":{\"kind\":\"record\",\"fields\":{\"Value\":{\"kind\":\"i64\",\"i64\":\"%d\"}}},\"effects\":[{\"capability\":\"observability.log\",\"value\":true}]}\n", got.Value)
		}
	}
}
