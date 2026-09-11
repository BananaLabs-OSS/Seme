// Command go-upb10-policy-check proves that the real bounded project cannot be
// published under an exact-only target policy and that failures stay explicit.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"seme.local/reference/goprojectplacementadapter"
	"seme.local/reference/goupb09cmdload"
	"seme.local/reference/goupb10cmdload"
	"seme.local/reference/targetplaninstance"
)

func main() {
	set := flag.NewFlagSet("go-upb10-policy-check", flag.ExitOnError)
	var paths goupb10cmdload.Paths
	paths.Bind(set)
	name := set.String("target-name", "wasm32-pulp-go-host-v1", "target identity")
	ruleNamespace := set.String("rule-namespace", "go-upb10", "placement rule identity namespace")
	revision := set.Uint64("target-revision", 1, "target revision")
	exact := set.Int("expect-exact", -1, "expected exact resolution count")
	impossible := set.Int("expect-impossible", -1, "expected impossible resolution count")
	set.Parse(os.Args[1:])
	if set.NArg() != 0 || *name == "" || *revision == 0 || *exact < 0 || *impossible <= 0 {
		fatal("arguments")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	base, err := goupb09cmdload.Load(ctx, paths.Base)
	if err != nil {
		fatal(err)
	}
	contracts, err := goupb10cmdload.ResolveContracts(paths)
	if err != nil {
		fatal(err)
	}
	input, err := goprojectplacementadapter.Derive(base.Bundle.Project, contracts.Target(), goprojectplacementadapter.Policy{Name: *name, Revision: *revision, RuleNamespace: *ruleNamespace, AllowedFidelity: []targetplaninstance.Fidelity{targetplaninstance.Exact}})
	if err != nil {
		fatal(err)
	}
	input.Artifact, err = targetplaninstance.Emit(input)
	if err != nil {
		fatal(err)
	}
	summary, err := targetplaninstance.Inspect(input)
	if err != nil || summary.Executable || summary.Boundaries != 0 || summary.Fidelities[targetplaninstance.Exact] != *exact || summary.Fidelities[targetplaninstance.Impossible] != *impossible || summary.Diagnostics != *impossible || summary.Requirements != *exact+*impossible {
		fatal(fmt.Sprintf("summary=%+v error=%v", summary, err))
	}
	fmt.Printf("exact-only rejected: %d exact, %d impossible, %d diagnostics, zero boundaries\n", *exact, *impossible, summary.Diagnostics)
}

func fatal(value any) {
	fmt.Fprintln(os.Stderr, "go-upb10-policy-check:", value)
	os.Exit(1)
}
