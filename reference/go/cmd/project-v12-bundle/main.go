// Command project-v12-bundle atomically publishes a standard authenticated
// Project-v12 directory from an authenticated Project-v11 predecessor and a
// separately reproduced Project-v12 extension.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/goupb09bundle"
)

func main() {
	base := flag.String("bundle-v11", "", "authenticated Project-v11 bundle")
	extension := flag.String("extension-v12", "", "Project-v12 extension directory")
	out := flag.String("out", "", "new Project-v12 bundle")
	flag.Parse()
	if flag.NArg() != 0 || *base == "" || *extension == "" || *out == "" {
		fatal("arguments")
	}
	artifacts, _, blobs, err := goupb09bundle.ReadV11Directory(*base)
	if err != nil {
		fatal(err)
	}
	read := func(name string) []byte {
		path := filepath.Join(*extension, name)
		info, readErr := os.Lstat(path)
		if readErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 64<<20 {
			fatal("extension_artifact")
		}
		value, readErr := os.ReadFile(path)
		if readErr != nil {
			fatal(readErr)
		}
		return value
	}
	artifacts.ControlledEffects = read("controlled-effects-v1.seme")
	artifacts.ProjectV12 = read("project-v12.seme")
	artifacts.ReplayAuthority = read("controlled-replay-v1.json")
	if err = goupb09bundle.WriteDirectory(*out, artifacts, blobs); err != nil {
		fatal(err)
	}
}

func fatal(value any) { fmt.Fprintln(os.Stderr, "project-v12-bundle:", value); os.Exit(1) }
