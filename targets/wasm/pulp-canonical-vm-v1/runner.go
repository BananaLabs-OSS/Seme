// Command pulp-seme-canonical-vm-proof is copied into a pinned Pulp checkout.
// It supplies opaque graph and request bytes through Pulp's public cell ABI;
// all semantic interpretation remains inside the Seme Wasm cell.
package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/BananaLabs-OSS/Pulp/ext"
	"github.com/BananaLabs-OSS/Pulp/internal/host"
	"github.com/BananaLabs-OSS/Pulp/internal/manifest"
	"github.com/tetratelabs/wazero"
)

// Keep this carrier bound synchronized with the exported Seme Pure Value ABI
// ceiling. This runner is copied into a pinned Pulp module and therefore cannot
// import Seme's Go package.
const maximumPureValueHexLineSize = 2*(256<<10) + 4096

var observations []bool

func init() {
	ext.Register(ext.Capability{Name: "observability.log", Provider: "seme.pulp.log-v1", Register: bindLog, Stub: bindDenied})
}

func bindLog(builder wazero.HostModuleBuilder, _ ext.Cell) error {
	builder.NewFunctionBuilder().WithFunc(func(_ context.Context, value uint32) uint32 {
		observations = append(observations, value != 0)
		return 0
	}).Export("log_bool")
	return nil
}
func bindDenied(builder wazero.HostModuleBuilder, _ ext.Cell) error {
	builder.NewFunctionBuilder().WithFunc(func(context.Context, uint32) uint32 { return 99 }).Export("log_bool")
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pulp-seme-canonical-vm-proof:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	options := map[string]string{}
	for i := 0; i < len(args); i++ {
		name, value, found := strings.Cut(args[i], "=")
		if !found {
			if i+1 == len(args) {
				return fmt.Errorf("%s requires a value", name)
			}
			i++
			value = args[i]
		}
		if name != "-manifest" && name != "-graph" && name != "-requests" {
			return fmt.Errorf("unsupported argument %s", name)
		}
		options[name] = value
	}
	if options["-manifest"] == "" || options["-graph"] == "" || options["-requests"] == "" {
		return fmt.Errorf("requires -manifest, -graph, and -requests")
	}
	graph, err := os.ReadFile(options["-graph"])
	if err != nil {
		return err
	}
	spec, err := manifest.Load(options["-manifest"])
	if err != nil {
		return err
	}
	ctx := context.Background()
	cell, err := host.Load(ctx, spec, host.NewRegistry(), &host.Limits{}, slog.Default())
	if err != nil {
		return err
	}
	defer cell.Close(ctx)
	if err := cell.Init(ctx, graph); err != nil {
		return err
	}
	defer cell.Shutdown(ctx)
	f, err := os.Open(options["-requests"])
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), maximumPureValueHexLineSize)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		request, err := hex.DecodeString(strings.TrimSpace(scanner.Text()))
		if err != nil {
			return err
		}
		before := len(observations)
		response, err := cell.Call(ctx, "seme.evaluate.v1", request)
		if err != nil {
			return err
		}
		if err := encoder.Encode(struct {
			Response string `json:"response"`
			Effects  []bool `json:"effects"`
		}{hex.EncodeToString(response), append([]bool{}, observations[before:]...)}); err != nil {
			return err
		}
	}
	return scanner.Err()
}
