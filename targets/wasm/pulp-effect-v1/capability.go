package main

import (
	"context"
	"fmt"

	"github.com/BananaLabs-OSS/Pulp/ext"
	"github.com/tetratelabs/wazero"
)

func init() {
	ext.Register(ext.Capability{Name: "observability.log", Provider: "seme.pulp.log-v1", Register: bindObservation, Stub: bindObservationStub})
}

func bindObservation(builder wazero.HostModuleBuilder, cell ext.Cell) error {
	builder.NewFunctionBuilder().WithFunc(func(_ context.Context, value uint32) uint32 {
		fmt.Printf("[observability.log] cell=%s value=%t\n", cell.Name(), value != 0)
		return 0
	}).Export("log_bool")
	return nil
}

func bindObservationStub(builder wazero.HostModuleBuilder, _ ext.Cell) error {
	builder.NewFunctionBuilder().WithFunc(func(context.Context, uint32) uint32 { return 99 }).Export("log_bool")
	return nil
}
