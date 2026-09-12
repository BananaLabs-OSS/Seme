// Command upb12-authority-build strictly reopens a complete Project-v13
// deployment and publishes its language-neutral, source-free projection seed.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"seme.local/reference/goprojectplacementadapter"
	"seme.local/reference/goupb10cmdload"
	"seme.local/reference/targetplaninstance"
	"seme.local/reference/upb12authority"
)

func main() {
	set := flag.NewFlagSet("upb12-authority-build", flag.ExitOnError)
	var paths goupb10cmdload.Paths
	paths.Bind(set)
	target := set.String("target-name", "", "authenticated target policy name")
	namespace := set.String("rule-namespace", "", "target realization namespace")
	out := set.String("out", "", "new authority directory")
	set.Parse(os.Args[1:])
	if set.NArg() != 0 || *target == "" || *namespace == "" || *out == "" {
		fatal("arguments")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	policy := goprojectplacementadapter.Policy{Name: *target, Revision: 1, RuleNamespace: *namespace, AllowedFidelity: []targetplaninstance.Fidelity{targetplaninstance.Exact, targetplaninstance.NativeIsland}}
	result, err := goupb10cmdload.Load(ctx, paths, policy)
	if err != nil {
		fatal(err)
	}
	selections := upb12authority.Selections{Configuration: result.ConfigurationSelection, Durable: result.DurableSelection, Transport: result.TransportSelection, Effects: result.EffectsSelection}
	if err = upb12authority.Publish(*out, upb12authority.Files(result, selections)); err != nil {
		fatal(err)
	}
}
func fatal(value any) { fmt.Fprintln(os.Stderr, "upb12-authority-build:", value); os.Exit(1) }
