// Command go-upb09-project creates an ordinary Go projection from authenticated
// Project-v12 authority. No Seme construction or execution artifact is copied.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"seme.local/reference/goresourcemanifest"
	"seme.local/reference/goupb09bundle"
	"seme.local/reference/goupb09cmdload"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb09-project:", err)
		os.Exit(65)
	}
}

func run(parent context.Context, args []string) error {
	f := flag.NewFlagSet("go-upb09-project", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var p goupb09cmdload.Paths
	p.Bind(f)
	to, module, goTool := f.String("to", "", "destination"), f.String("module", "", "module path"), f.String("go", "", "Go tool")
	if err := f.Parse(args); err != nil || f.NArg() != 0 || *to == "" || *module == "" || *goTool == "" || !cleanAbsolute(*to) || !cleanAbsolute(*goTool) {
		return fmt.Errorf("arguments")
	}
	if _, err := os.Lstat(*to); err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("destination_exists")
	}
	parentDir := filepath.Dir(*to)
	if !plainDirectory(parentDir) {
		return fmt.Errorf("destination_parent")
	}
	if i, err := os.Lstat(*goTool); err != nil || !i.Mode().IsRegular() || i.Mode()&0111 == 0 {
		return fmt.Errorf("go_tool")
	}
	ctx, cancel := context.WithTimeout(parent, 180*time.Second)
	defer cancel()
	loaded, err := goupb09cmdload.Load(ctx, p)
	if err != nil {
		return err
	}
	packages, err := goupb09bundle.Project(loaded.Bundle)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parentDir, ".seme-upb09-project-")
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(tmp)
		}
	}()
	write := func(rel string, data []byte) error {
		if !safeRelative(rel) || len(data) == 0 {
			return fmt.Errorf("projection_file")
		}
		path := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		return os.WriteFile(path, data, 0600)
	}
	for pkg, data := range packages {
		rel := ""
		if pkg != *module {
			if !strings.HasPrefix(pkg, *module+"/") {
				return fmt.Errorf("package_outside_module")
			}
			rel = strings.TrimPrefix(pkg, *module+"/")
		}
		name := "seme_projected.go"
		if rel != "" {
			name = rel + "/seme_projected.go"
		}
		if err = write(name, data); err != nil {
			return err
		}
	}
	// Detached resource paths and bytes are authenticated independently by the
	// resource plan and the bundle's closed blob inventory.
	for _, item := range loaded.Bundle.Base.Base.Base.Resource.Model.Resources {
		data, ok := loaded.Bundle.Blobs[item.SHA256]
		if !ok {
			return fmt.Errorf("resource_missing:%s", item.Identity)
		}
		if err = write(item.Path, data); err != nil {
			return err
		}
	}
	placements := map[string]string{}
	for _, item := range loaded.Bundle.Base.Base.Base.Resource.Model.Placements {
		if item.Resource == "" || item.Destination == "" || placements[item.Resource] != "" {
			return fmt.Errorf("resource_placement")
		}
		placements[item.Resource] = item.Destination
	}
	resourceSelection := make([]goresourcemanifest.Selection, 0, len(loaded.Bundle.Base.Base.Base.Resource.Model.Resources))
	for _, item := range loaded.Bundle.Base.Base.Base.Resource.Model.Resources {
		destination := placements[item.Identity]
		if destination == "" {
			return fmt.Errorf("resource_placement_missing:%s", item.Identity)
		}
		resourceSelection = append(resourceSelection, goresourcemanifest.Selection{Identity: item.Identity, Path: item.Path, Destination: destination, MediaType: item.MediaType, Size: item.Size, SHA256: item.SHA256})
	}
	resourceManifest, err := goresourcemanifest.Encode(resourceSelection)
	if err != nil {
		return err
	}
	if err = write("resources.json", resourceManifest); err != nil {
		return err
	}
	for name, data := range map[string][]byte{"configuration-selection.json": loaded.ConfigurationSelection, "durable-selection.json": loaded.DurableSelection, "ordered-transport-selection.json": loaded.TransportSelection, "controlled-effects-selection.json": loaded.EffectsSelection} {
		if err = write(name, data); err != nil {
			return err
		}
	}
	if err = write("go.mod", []byte("module "+*module+"\n\ngo 1.26\n")); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, *goTool, "test", "./...")
	cmd.Dir = tmp
	environment := make([]string, 0, len(os.Environ())+4)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "GOROOT=") && !strings.HasPrefix(value, "GOWORK=") && !strings.HasPrefix(value, "GOPROXY=") && !strings.HasPrefix(value, "GOSUMDB=") {
			environment = append(environment, value)
		}
	}
	// The caller selects an exact executable. Bind its matching standard
	// library as well so an ambient GOROOT cannot mix another Go edition into
	// native validation.
	goRoot := filepath.Dir(filepath.Dir(*goTool))
	cmd.Env = append(environment, "GOROOT="+goRoot, "GOWORK=off", "GOPROXY=off", "GOSUMDB=off")
	if out, testErr := cmd.CombinedOutput(); testErr != nil {
		return fmt.Errorf("native_test:%w:%s", testErr, out)
	}
	if err = os.Rename(tmp, *to); err != nil {
		return fmt.Errorf("projection_commit:%w", err)
	}
	keep = true
	return nil
}

func cleanAbsolute(p string) bool { return filepath.IsAbs(p) && filepath.Clean(p) == p }
func plainDirectory(p string) bool {
	i, e := os.Lstat(p)
	if e != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return false
	}
	r, e := filepath.EvalSymlinks(p)
	return e == nil && r == p
}
func safeRelative(p string) bool {
	return p != "" && filepath.IsLocal(filepath.FromSlash(p)) && filepath.ToSlash(filepath.Clean(filepath.FromSlash(p))) == p
}
