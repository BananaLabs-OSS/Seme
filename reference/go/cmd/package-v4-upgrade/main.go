// Command package-v4-upgrade upgrades authenticated Package-v2 record
// ownership into Package v4 without consulting a language provider.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/packagedeclarationadapter"
	"seme.local/reference/packagev3instance"
)

func main() {
	paths := map[string]*string{}
	for _, name := range []string{"package", "foundation-contract", "execution-contract", "package-contract", "dependency-contract", "configuration-contract", "project-contract", "out"} {
		var value string
		paths[name] = &value
		flag.StringVar(paths[name], name, "", name)
	}
	flag.Parse()
	if flag.NArg() != 0 {
		fail("arguments")
	}
	for name, value := range paths {
		if *value == "" || !filepath.IsAbs(*value) || filepath.Clean(*value) != *value {
			fail("flag:" + name)
		}
	}
	read := func(name string) []byte {
		value, err := os.ReadFile(*paths[name])
		if err != nil {
			fail(name + ":" + err.Error())
		}
		return value
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV8(read("foundation-contract"), read("execution-contract"), read("package-contract"), read("dependency-contract"), read("configuration-contract"), read("project-contract"))
	if err != nil {
		fail(err.Error())
	}
	base := read("package")
	declarations, err := packagedeclarationadapter.DataTypes(base)
	if err != nil {
		fail(err.Error())
	}
	out, err := packagev3instance.EmitV4(packagev3instance.InputsV4{Contracts: contracts, PackageV2: base, Declarations: declarations})
	if err != nil {
		fail(err.Error())
	}
	if _, err = os.Lstat(*paths["out"]); !os.IsNotExist(err) {
		fail("output_exists")
	}
	tmp, err := os.CreateTemp(filepath.Dir(*paths["out"]), ".package-v4-upgrade-*")
	if err != nil {
		fail(err.Error())
	}
	name := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(name)
		}
	}()
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(out)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, *paths["out"])
	}
	if err != nil {
		fail(err.Error())
	}
	ok = true
}
func fail(message string) { fmt.Fprintln(os.Stderr, "package-v4-upgrade:", message); os.Exit(1) }
