// Command project-assemble binds an already canonical Execution artifact and
// explicit language-provider ownership evidence into neutral Project v1.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectemitter"
	"seme.local/reference/wire"
)

type manifest struct {
	Identity    string            `json:"identity"`
	RootPackage string            `json:"root_package"`
	Packages    []manifestPackage `json:"packages"`
}
type manifestPackage struct {
	Name         string                      `json:"name"`
	Interfaces   []manifestInterface         `json:"interfaces"`
	Dependencies []projectemitter.Dependency `json:"dependencies"`
	Effects      []string                    `json:"effects"`
}
type manifestInterface struct {
	Name       string   `json:"name"`
	Function   string   `json:"function"`
	Parameters []string `json:"parameters"`
	Result     string   `json:"result"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var executionPath, manifestPath, executionContract, packageContract, projectContract, out string
	flag.StringVar(&executionPath, "execution", "", "canonical Execution artifact")
	flag.StringVar(&manifestPath, "manifest", "", "explicit package ownership JSON")
	flag.StringVar(&executionContract, "execution-contract", "", "Execution contract")
	flag.StringVar(&packageContract, "package-contract", "", "Package v1 contract")
	flag.StringVar(&projectContract, "project-contract", "", "Project v1 contract")
	flag.StringVar(&out, "out", "", "new Project artifact destination")
	flag.Parse()
	for name, value := range map[string]string{"execution": executionPath, "manifest": manifestPath, "execution-contract": executionContract, "package-contract": packageContract, "project-contract": projectContract, "out": out} {
		if value == "" {
			return fmt.Errorf("project_assemble.missing:%s", name)
		}
	}
	read := func(path string) ([]byte, error) {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("project_assemble.not_regular:%s", path)
		}
		return os.ReadFile(path)
	}
	executionBytes, err := read(executionPath)
	if err != nil {
		return err
	}
	execution, err := wire.Decode(executionBytes)
	if err != nil {
		return fmt.Errorf("project_assemble.execution:%w", err)
	}
	canonical, err := wire.Encode(execution)
	if err != nil || string(canonical) != string(executionBytes) {
		return fmt.Errorf("project_assemble.execution_noncanonical")
	}
	manifestBytes, err := read(manifestPath)
	if err != nil {
		return err
	}
	var m manifest
	decoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&m); err != nil {
		return fmt.Errorf("project_assemble.manifest:%w", err)
	}
	var extra any
	if trailing := decoder.Decode(&extra); trailing != io.EOF {
		return fmt.Errorf("project_assemble.manifest_trailing")
	}
	input := projectemitter.Input{Identity: m.Identity, RootPackage: m.RootPackage, Execution: execution}
	for _, p := range m.Packages {
		outPackage := projectemitter.Package{Name: p.Name, Dependencies: p.Dependencies}
		for _, raw := range p.Effects {
			id, parseErr := wire.ParseID(raw)
			if parseErr != nil {
				return fmt.Errorf("project_assemble.effect:%w", parseErr)
			}
			outPackage.Effects = append(outPackage.Effects, id)
		}
		for _, raw := range p.Interfaces {
			fn, parseErr := wire.ParseID(raw.Function)
			if parseErr != nil {
				return fmt.Errorf("project_assemble.function:%w", parseErr)
			}
			result, parseErr := wire.ParseID(raw.Result)
			if parseErr != nil {
				return fmt.Errorf("project_assemble.result:%w", parseErr)
			}
			item := projectemitter.Interface{Name: raw.Name, Function: fn, Result: result}
			for _, value := range raw.Parameters {
				id, e := wire.ParseID(value)
				if e != nil {
					return fmt.Errorf("project_assemble.parameter:%w", e)
				}
				item.Parameters = append(item.Parameters, id)
			}
			outPackage.Interfaces = append(outPackage.Interfaces, item)
		}
		input.Packages = append(input.Packages, outPackage)
	}
	ec, err := read(executionContract)
	if err != nil {
		return err
	}
	pc, err := read(packageContract)
	if err != nil {
		return err
	}
	prc, err := read(projectContract)
	if err != nil {
		return err
	}
	contracts, err := contractcatalog.ResolveProjectContractSet(ec, pc, prc)
	if err != nil {
		return err
	}
	artifact, err := projectemitter.Emit(contracts, input)
	if err != nil {
		return err
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	if _, err = os.Lstat(absOut); err == nil {
		return fmt.Errorf("project_assemble.output_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(absOut), ".project-assemble-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	published := false
	defer func() {
		if !published {
			_ = os.Remove(name)
		}
	}()
	if _, err = temp.Write(artifact); err != nil {
		_ = temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if _, err = os.Lstat(absOut); err == nil {
		return fmt.Errorf("project_assemble.output_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err = os.Link(name, absOut); err != nil {
		return fmt.Errorf("project_assemble.publish:%w", err)
	}
	published = true
	return os.Remove(name)
}
