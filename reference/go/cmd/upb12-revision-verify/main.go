package main

import (
	"flag"
	"fmt"
	"os"
	"seme.local/reference/upb12revision"
	"seme.local/reference/wire"
)

func main() {
	prior := flag.String("prior", "", "prior compiled graph")
	result := flag.String("result", "", "result compiled graph")
	targetText := flag.String("target", "", "semantic target identity")
	fieldText := flag.String("field", "00000000000000000000000000009110", "semantic field identity")
	expected := flag.String("expected", "", "expected field text")
	replacement := flag.String("replacement", "", "replacement field text")
	flag.Parse()
	if flag.NArg() != 0 || *prior == "" || *result == "" || *targetText == "" || *expected == "" || *replacement == "" {
		fatal("arguments")
	}
	target, err := wire.ParseID(*targetText)
	if err != nil {
		fatal(err)
	}
	field, err := wire.ParseID(*fieldText)
	if err != nil {
		fatal(err)
	}
	before, err := os.ReadFile(*prior)
	if err != nil {
		fatal(err)
	}
	after, err := os.ReadFile(*result)
	if err != nil {
		fatal(err)
	}
	if err = upb12revision.Verify(before, after, target, field, []byte(*expected), []byte(*replacement)); err != nil {
		fatal(err)
	}
}
func fatal(value any) { fmt.Fprintf(os.Stderr, "upb12-revision-verify: %v\n", value); os.Exit(64) }
