package wasmtarget

import (
	"bytes"
	"fmt"
)

// PureVMOpcode is the language-neutral target instruction set used after
// canonical graph certification. It describes meaning, never source syntax or
// application identity.
type PureVMOpcode byte

const (
	PureVMI64Parameter PureVMOpcode = iota + 1
	PureVMBoolParameter
	PureVMI64Constant
	PureVMI64Add
	PureVMI64Subtract
	PureVMI64Multiply
	PureVMI64LessEqual
	PureVMBoolAnd
	PureVMBoolOr
	PureVMSelectI64
)

type PureVMInstruction struct {
	Opcode    PureVMOpcode
	Parameter uint32
	Immediate int64
}

type pureVMStackType byte

const (
	vmI64 pureVMStackType = iota + 1
	vmBool
)

// CompilePureVMScalar compiles the typed scalar subset of the reusable VM to
// an executable Wasm function `run`. It is intentionally independent of the
// cumulative application; later value-handle instructions share this exact
// checked instruction stream and module builder.
func CompilePureVMScalar(parameterCount uint32, program []PureVMInstruction) ([]byte, error) {
	if parameterCount > 32 || len(program) == 0 || len(program) > 4096 {
		return nil, fmt.Errorf("wasm.vm_program_size")
	}
	stack := []pureVMStackType{}
	var code bytes.Buffer
	pop := func(want pureVMStackType) error {
		if len(stack) == 0 || stack[len(stack)-1] != want {
			return fmt.Errorf("wasm.vm_stack_type")
		}
		stack = stack[:len(stack)-1]
		return nil
	}
	for _, instruction := range program {
		switch instruction.Opcode {
		case PureVMI64Parameter, PureVMBoolParameter:
			if instruction.Parameter >= parameterCount {
				return nil, fmt.Errorf("wasm.vm_parameter")
			}
			code.WriteByte(0x20) // local.get
			uleb(&code, uint64(instruction.Parameter))
			if instruction.Opcode == PureVMBoolParameter {
				// Parameters use i64 at this operation-level seam; require 0/1.
				code.Write([]byte{0x50, 0x45}) // i64.eqz; i32.eqz
				stack = append(stack, vmBool)
			} else {
				stack = append(stack, vmI64)
			}
		case PureVMI64Constant:
			code.WriteByte(0x42)
			sleb(&code, instruction.Immediate)
			stack = append(stack, vmI64)
		case PureVMI64Add, PureVMI64Subtract, PureVMI64Multiply:
			if err := pop(vmI64); err != nil {
				return nil, err
			}
			if err := pop(vmI64); err != nil {
				return nil, err
			}
			op := byte(0x7c)
			if instruction.Opcode == PureVMI64Subtract {
				op = 0x7d
			}
			if instruction.Opcode == PureVMI64Multiply {
				op = 0x7e
			}
			code.WriteByte(op)
			stack = append(stack, vmI64)
		case PureVMI64LessEqual:
			if err := pop(vmI64); err != nil {
				return nil, err
			}
			if err := pop(vmI64); err != nil {
				return nil, err
			}
			code.WriteByte(0x57) // i64.le_s
			stack = append(stack, vmBool)
		case PureVMBoolAnd, PureVMBoolOr:
			if err := pop(vmBool); err != nil {
				return nil, err
			}
			if err := pop(vmBool); err != nil {
				return nil, err
			}
			if instruction.Opcode == PureVMBoolAnd {
				code.WriteByte(0x71)
			} else {
				code.WriteByte(0x72)
			}
			stack = append(stack, vmBool)
		case PureVMSelectI64:
			if err := pop(vmBool); err != nil {
				return nil, err
			}
			if err := pop(vmI64); err != nil {
				return nil, err
			}
			if err := pop(vmI64); err != nil {
				return nil, err
			}
			code.WriteByte(0x1b) // select
			stack = append(stack, vmI64)
		default:
			return nil, fmt.Errorf("wasm.vm_opcode")
		}
	}
	if len(stack) != 1 || stack[0] != vmI64 {
		return nil, fmt.Errorf("wasm.vm_result_type")
	}
	code.WriteByte(0x0b)
	return scalarVMModule(parameterCount, code.Bytes()), nil
}

func scalarVMModule(parameterCount uint32, expression []byte) []byte {
	var wasm bytes.Buffer
	wasm.Write([]byte{'\x00', 'a', 's', 'm', '\x01', 0, 0, 0})
	var types bytes.Buffer
	uleb(&types, 1)
	parameters := bytes.Repeat([]byte{0x7e}, int(parameterCount))
	functionType(&types, parameters, []byte{0x7e})
	section(&wasm, 1, types.Bytes())
	section(&wasm, 3, []byte{1, 0})
	var exports bytes.Buffer
	uleb(&exports, 1)
	export(&exports, "run", 0, 0)
	section(&wasm, 7, exports.Bytes())
	var code bytes.Buffer
	uleb(&code, 1)
	body := append([]byte{0}, expression...) // zero local groups
	uleb(&code, uint64(len(body)))
	code.Write(body)
	section(&wasm, 10, code.Bytes())
	return wasm.Bytes()
}
