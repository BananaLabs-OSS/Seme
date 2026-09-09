// canonical-wasm-cell is a reusable Wasm realization of an ordinary Seme
// canonical program. The canonical graph is supplied to pulp_init; neither
// the cell nor its ABI selects an application, package, or function by name.
package main

import (
	"runtime"
	"unsafe"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/wasmtarget"
	"seme.local/reference/wire"
)

var (
	program  wire.Envelope
	boundary wasmtarget.PureApplicationBoundary
	ready    bool
	blocks   = map[uint32][]byte{}
	lastErr  []byte
)

func main() {}

//go:wasmexport pulp_alloc
func pulpAlloc(size uint32) uint32 {
	if size == 0 {
		return 0
	}
	b := make([]byte, size)
	p := uint32(uintptr(unsafe.Pointer(&b[0])))
	blocks[p] = b
	return p
}

//go:wasmexport pulp_free
func pulpFree(ptr, size uint32) {
	_ = size
	delete(blocks, ptr)
}

//go:wasmexport pulp_init
func pulpInit(configPtr, configLen uint32) int32 {
	ready = false
	lastErr = nil
	data, ok := guestBytes(configPtr, configLen)
	if !ok || len(data) == 0 {
		return fail("canonical_vm.graph_required")
	}
	g, err := wire.Decode(append([]byte(nil), data...))
	if err != nil {
		return fail("canonical_vm.graph_invalid")
	}
	b, err := wasmtarget.CertifyPureApplicationBoundary(g)
	if err != nil {
		return fail("canonical_vm.boundary_invalid")
	}
	program, boundary, ready = g, b, true
	return 0
}

//go:wasmexport pulp_step
func pulpStep(inputPtr, inputLen uint32) int32 {
	_, _ = inputPtr, inputLen
	return 0
}

//go:wasmexport pulp_shutdown
func pulpShutdown() int32 {
	program = wire.Envelope{}
	boundary = wasmtarget.PureApplicationBoundary{}
	ready = false
	return 0
}

//go:wasmexport pulp_on_call
func pulpOnCall(namePtr, nameLen, requestPtr, requestLen, responsePtrOut, responseLenOut uint32) int32 {
	lastErr = nil
	if !ready {
		return fail("canonical_vm.not_initialized")
	}
	name, ok := guestBytes(namePtr, nameLen)
	if !ok || string(name) != "seme.evaluate.v1" {
		return fail("canonical_vm.provider")
	}
	request, ok := guestBytes(requestPtr, requestLen)
	if !ok {
		return fail("canonical_vm.request_memory")
	}
	arguments, err := wasmtarget.DecodePureApplicationRequest(boundary, request)
	if err != nil {
		return fail("canonical_vm.request_invalid")
	}
	authorized := make(map[string]bool, len(boundary.RequiredCapabilities))
	for _, capability := range boundary.RequiredCapabilities {
		authorized[capability] = true
	}
	value, effects, err := canonicaleval.EvaluateAuthorized(program, arguments, authorized)
	if err != nil {
		return fail("canonical_vm.evaluate")
	}
	// The evaluator has already validated the complete graph and computed the
	// complete trace. Only now may observations cross the capability boundary.
	for _, effect := range effects {
		if effect.Capability != "observability.log" {
			return fail("canonical_vm.effect")
		}
		var bit uint32
		if effect.Value {
			bit = 1
		}
		if hostLogBool(bit) != 0 {
			return fail("canonical_vm.effect_denied")
		}
	}
	response, err := wasmtarget.EncodePureValue(boundary.Result, value)
	if err != nil {
		return fail("canonical_vm.response_invalid")
	}
	responsePtr := pulpAlloc(uint32(len(response)))
	if responsePtr == 0 && len(response) != 0 {
		return fail("canonical_vm.response_memory")
	}
	if len(response) != 0 {
		copy(blocks[responsePtr], response)
	}
	if !writeU32(responsePtrOut, responsePtr) || !writeU32(responseLenOut, uint32(len(response))) {
		pulpFree(responsePtr, uint32(len(response)))
		return fail("canonical_vm.response_pointer")
	}
	runtime.KeepAlive(response)
	return 0
}

//go:wasmexport pulp_on_call_error_ptr
func pulpOnCallErrorPtr() uint32 {
	if len(lastErr) == 0 {
		return 0
	}
	return uint32(uintptr(unsafe.Pointer(&lastErr[0])))
}

//go:wasmexport pulp_on_call_error_len
func pulpOnCallErrorLen() uint32 { return uint32(len(lastErr)) }

// Pulp exposes distinct optional diagnostic hooks for initialization and
// calls. Both report the cell's single deterministic last error buffer.
//
//go:wasmexport pulp_init_error_ptr
func pulpInitErrorPtr() uint32 { return pulpOnCallErrorPtr() }

//go:wasmexport pulp_init_error_len
func pulpInitErrorLen() uint32 { return pulpOnCallErrorLen() }

func fail(message string) int32 {
	lastErr = []byte(message)
	return 1
}

func guestBytes(ptr, size uint32) ([]byte, bool) {
	if size == 0 {
		return nil, true
	}
	if ptr == 0 || uint64(ptr)+uint64(size) > 1<<32 {
		return nil, false
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size), true
}

func writeU32(ptr, value uint32) bool {
	if ptr == 0 || uint64(ptr)+4 > 1<<32 {
		return false
	}
	b := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), 4)
	b[0], b[1], b[2], b[3] = byte(value), byte(value>>8), byte(value>>16), byte(value>>24)
	return true
}
