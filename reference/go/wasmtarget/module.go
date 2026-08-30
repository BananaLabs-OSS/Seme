package wasmtarget

import (
	"bytes"
	"fmt"
)

// module derives every request/response size and field offset from the
// validated canonical RecordType graph. The remaining instructions are the
// deliberately scoped quota target profile, not a source-language template.
func module(layout applicationLayout) ([]byte, error) {
	if layout.requestHeaderSize > 65535 || layout.responseHeaderSize > 65535 {
		return nil, fmt.Errorf("wasm.application_layout_too_large")
	}
	var wasm bytes.Buffer
	wasm.Write([]byte{'\x00', 'a', 's', 'm', '\x01', 0, 0, 0})
	section(&wasm, 1, []byte{
		4,
		0x60, 1, 0x7f, 1, 0x7f,
		0x60, 2, 0x7f, 0x7f, 1, 0x7f,
		0x60, 0, 1, 0x7f,
		0x60, 6, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 1, 0x7f,
	})
	var imports bytes.Buffer
	uleb(&imports, 1)
	name(&imports, "pulp")
	name(&imports, "log_bool")
	imports.Write([]byte{0, 0})
	section(&wasm, 2, imports.Bytes())
	section(&wasm, 3, []byte{5, 0, 1, 1, 2, 3})
	section(&wasm, 5, []byte{1, 0, 1})
	var globals bytes.Buffer
	globals.Write([]byte{1, 0x7f, 1, 0x41})
	sleb(&globals, 1024)
	globals.WriteByte(0x0b)
	section(&wasm, 6, globals.Bytes())
	var exports bytes.Buffer
	uleb(&exports, 6)
	export(&exports, "memory", 2, 0)
	export(&exports, "pulp_alloc", 0, 1)
	export(&exports, "pulp_init", 0, 2)
	export(&exports, "pulp_step", 0, 3)
	export(&exports, "pulp_shutdown", 0, 4)
	export(&exports, "pulp_on_call", 0, 5)
	section(&wasm, 7, exports.Bytes())

	bodies := [][]byte{
		{1, 1, 0x7f, 0x23, 0, 0x22, 1, 0x20, 0, 0x6a, 0x41, 8, 0x6a, 0x24, 0, 0x20, 1, 0x0b},
		{0, 0x41, 0, 0x0b},
		{0, 0x41, 0, 0x0b},
		{0, 0x41, 0, 0x0b},
		providerBody(layout),
	}
	var code bytes.Buffer
	uleb(&code, uint64(len(bodies)))
	for _, body := range bodies {
		uleb(&code, uint64(len(body)))
		code.Write(body)
	}
	section(&wasm, 10, code.Bytes())
	return wasm.Bytes(), nil
}

func providerBody(layout applicationLayout) []byte {
	var body bytes.Buffer
	body.Write([]byte{1, 4, 0x7f}) // accepted, subject length, evidence length, response length
	body.Write([]byte{0x20, 3, 0x41})
	sleb(&body, int64(layout.requestHeaderSize))
	body.Write([]byte{0x49, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b})
	loadRecordI32(&body, 2, layout.requestHeaderSize-8)
	body.Write([]byte{0x22, 7, 0x41})
	sleb(&body, 4096)
	body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b})
	loadRecordI32(&body, 2, layout.requestHeaderSize-4)
	body.Write([]byte{0x22, 8, 0x41})
	sleb(&body, 4096)
	body.Write([]byte{0x4b, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b})
	body.Write([]byte{0x20, 3, 0x41})
	sleb(&body, int64(layout.requestHeaderSize))
	body.Write([]byte{0x20, 7, 0x6a, 0x20, 8, 0x6a, 0x47, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b})
	loadRecordI64(&body, 2, layout.requestOffsets[0])
	loadRecordI64(&body, 2, layout.requestOffsets[1])
	body.WriteByte(0x7c) // i64.add
	loadRecordI64(&body, 2, layout.requestOffsets[2])
	body.Write([]byte{0x57, 0x22, 6, 0x10, 0, 0x04, 0x40, 0x00, 0x0b})
	constI32(&body, 8192) // Result tag: Ok
	body.Write([]byte{0x41, 0, 0x3a, 0, 0})
	constI32(&body, 8192+layout.responseBoolOffset)
	body.Write([]byte{0x20, 6, 0x3a, 0, 0}) // i32.store8 align=0 offset=0
	constI32(&body, 8192+layout.responseBoolOffset+1)
	body.Write([]byte{0x20, 7, 0x36, 0, 0})
	constI32(&body, 8192+layout.responseBoolOffset+5)
	body.Write([]byte{0x20, 8, 0x36, 0, 0})
	constI32(&body, 8192+layout.responseHeaderSize)
	body.Write([]byte{0x20, 2, 0x41})
	sleb(&body, int64(layout.requestHeaderSize))
	body.Write([]byte{0x6a, 0x20, 7, 0xfc, 0x0a, 0, 0})
	constI32(&body, 8192+layout.responseHeaderSize)
	body.Write([]byte{0x20, 7, 0x6a, 0x20, 2, 0x41})
	sleb(&body, int64(layout.requestHeaderSize))
	body.Write([]byte{0x6a, 0x20, 7, 0x6a, 0x20, 8, 0xfc, 0x0a, 0, 0})
	body.Write([]byte{0x41})
	sleb(&body, int64(layout.responseHeaderSize))
	body.Write([]byte{0x20, 7, 0x6a, 0x20, 8, 0x6a, 0x21, 9})
	body.Write([]byte{0x20, 4})
	constI32(&body, 8192)
	body.Write([]byte{0x36, 2, 0, 0x20, 5, 0x20, 9, 0x36, 2, 0, 0x41, 0, 0x0b})
	return body.Bytes()
}

func loadRecordI32(body *bytes.Buffer, pointerLocal byte, offset uint64) {
	body.Write([]byte{0x20, pointerLocal, 0x28, 2})
	uleb(body, offset)
}

func loadRecordI64(body *bytes.Buffer, pointerLocal byte, offset uint64) {
	body.Write([]byte{0x20, pointerLocal, 0x29, 3})
	uleb(body, offset)
}
func constI32(body *bytes.Buffer, value uint64) {
	body.WriteByte(0x41)
	sleb(body, int64(value))
}
func export(output *bytes.Buffer, value string, kind byte, index uint64) {
	name(output, value)
	output.WriteByte(kind)
	uleb(output, index)
}
func section(output *bytes.Buffer, id byte, payload []byte) {
	output.WriteByte(id)
	uleb(output, uint64(len(payload)))
	output.Write(payload)
}
func name(output *bytes.Buffer, value string) {
	uleb(output, uint64(len(value)))
	output.WriteString(value)
}
func uleb(output *bytes.Buffer, value uint64) {
	for {
		part := byte(value & 0x7f)
		value >>= 7
		if value != 0 {
			part |= 0x80
		}
		output.WriteByte(part)
		if value == 0 {
			return
		}
	}
}
func sleb(output *bytes.Buffer, value int64) {
	for {
		part := byte(value & 0x7f)
		value >>= 7
		done := (value == 0 && part&0x40 == 0) || (value == -1 && part&0x40 != 0)
		if !done {
			part |= 0x80
		}
		output.WriteByte(part)
		if done {
			return
		}
	}
}
