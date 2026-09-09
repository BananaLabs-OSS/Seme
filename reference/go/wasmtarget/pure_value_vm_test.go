package wasmtarget

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPureVMScalarExecutesModularArithmeticAndTypedControl(t *testing.T) {
	// select((a+b)*b, a-b, a<=b && flag)
	program := []PureVMInstruction{
		{Opcode: PureVMI64Parameter, Parameter: 0}, {Opcode: PureVMI64Parameter, Parameter: 1}, {Opcode: PureVMI64Add},
		{Opcode: PureVMI64Parameter, Parameter: 1}, {Opcode: PureVMI64Multiply},
		{Opcode: PureVMI64Parameter, Parameter: 0}, {Opcode: PureVMI64Parameter, Parameter: 1}, {Opcode: PureVMI64Subtract},
		{Opcode: PureVMI64Parameter, Parameter: 0}, {Opcode: PureVMI64Parameter, Parameter: 1}, {Opcode: PureVMI64LessEqual},
		{Opcode: PureVMBoolParameter, Parameter: 2}, {Opcode: PureVMBoolAnd}, {Opcode: PureVMSelectI64},
	}
	wasm, err := CompilePureVMScalar(3, program)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "vm.wasm")
	if err = os.WriteFile(path, wasm, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ args, want string }{{"2 3 1", "15"}, {"5 3 1", "2"}, {"9223372036854775807 1 0", "9223372036854775806"}} {
		parts := strings.Split(test.args, " ")
		command := exec.Command("node", append([]string{filepath.Join("..", "..", "js", "pure-vm-scalar-runner.mjs"), path}, parts...)...)
		output, runErr := command.CombinedOutput()
		if runErr != nil || strings.TrimSpace(string(output)) != test.want {
			t.Fatalf("run %s: %v %s", test.args, runErr, output)
		}
	}
}

func TestPureVMScalarRejectsIllTypedPrograms(t *testing.T) {
	bad := [][]PureVMInstruction{
		{{Opcode: PureVMI64Add}},
		{{Opcode: PureVMI64Constant, Immediate: 1}, {Opcode: PureVMBoolAnd}},
		{{Opcode: PureVMI64Parameter, Parameter: 1}},
		{{Opcode: 255}},
	}
	for _, program := range bad {
		if _, err := CompilePureVMScalar(1, program); err == nil {
			t.Fatal("ill-typed VM program accepted")
		}
	}
}
