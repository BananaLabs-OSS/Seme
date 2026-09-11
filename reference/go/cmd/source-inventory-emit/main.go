// Command source-inventory-emit binds a detached, language-neutral source
// snapshot to an existing semantic Project authority under Project v2.
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
	"seme.local/reference/projectsource"
	"seme.local/reference/sourceinventory"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var snapshotPath, projectPath, executionContract, packageContract, projectContract, foundationContract, dependencyContract, configurationContract, out string
	flag.StringVar(&snapshotPath, "snapshot", "", "detached source snapshot JSON")
	flag.StringVar(&projectPath, "project", "", "validated semantic Project artifact")
	flag.StringVar(&executionContract, "execution-contract", "", "Execution contract")
	flag.StringVar(&packageContract, "package-contract", "", "Package contract")
	flag.StringVar(&projectContract, "project-contract", "", "Project v2 contract")
	flag.StringVar(&foundationContract, "foundation-contract", "", "Foundation contract for Project v8 mode")
	flag.StringVar(&dependencyContract, "dependency-contract", "", "Dependency contract for Project v8 mode")
	flag.StringVar(&configurationContract, "configuration-contract", "", "Configuration contract for Project v8 mode")
	flag.StringVar(&out, "out", "", "new source inventory artifact")
	flag.Parse()
	for name, value := range map[string]string{"snapshot": snapshotPath, "project": projectPath, "execution-contract": executionContract, "package-contract": packageContract, "project-contract": projectContract, "out": out} {
		if value == "" {
			return fmt.Errorf("source_inventory_emit.missing:%s", name)
		}
	}
	read := func(path string) ([]byte, error) {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("source_inventory_emit.not_regular:%s", path)
		}
		return os.ReadFile(path)
	}
	raw, err := read(snapshotPath)
	if err != nil {
		return err
	}
	var snapshot projectsource.Snapshot
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&snapshot); err != nil {
		return fmt.Errorf("source_inventory_emit.snapshot:%w", err)
	}
	var extra any
	if trailing := decoder.Decode(&extra); trailing != io.EOF {
		return fmt.Errorf("source_inventory_emit.snapshot_trailing")
	}
	if err = projectsource.ValidateSnapshot(snapshot); err != nil {
		return err
	}
	project, err := read(projectPath)
	if err != nil {
		return err
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
	v8 := foundationContract != "" || dependencyContract != "" || configurationContract != ""
	var inventory []byte
	var projectAuthority contractcatalog.Contract
	if v8 {
		if foundationContract == "" || dependencyContract == "" || configurationContract == "" {
			return fmt.Errorf("source_inventory_emit.v8_contracts")
		}
		fc, e := read(foundationContract)
		if e != nil {
			return e
		}
		dc, e := read(dependencyContract)
		if e != nil {
			return e
		}
		cc, e := read(configurationContract)
		if e != nil {
			return e
		}
		contracts, e := contractcatalog.ResolveProjectContractSetV8(fc, ec, pc, dc, cc, prc)
		if e != nil {
			return e
		}
		projectAuthority = contracts.Project()
		inventory, err = sourceinventory.EmitV8(projectAuthority, project, snapshot)
	} else {
		contracts, e := contractcatalog.ResolveProjectContractSetV2(ec, pc, prc)
		if e != nil {
			return e
		}
		projectAuthority = contracts.Project()
		inventory, err = sourceinventory.Emit(projectAuthority, project, snapshot)
	}
	if err != nil {
		return err
	}
	if v8 {
		err = sourceinventory.ValidateV8(projectAuthority, project, inventory)
	} else {
		err = sourceinventory.Validate(projectAuthority, project, inventory)
	}
	if err != nil {
		return err
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	if _, err = os.Lstat(absOut); err == nil {
		return fmt.Errorf("source_inventory_emit.output_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(absOut), ".source-inventory-emit-*")
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
	if _, err = temp.Write(inventory); err != nil {
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
		return fmt.Errorf("source_inventory_emit.output_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err = os.Link(name, absOut); err != nil {
		return fmt.Errorf("source_inventory_emit.publish:%w", err)
	}
	published = true
	return os.Remove(name)
}
