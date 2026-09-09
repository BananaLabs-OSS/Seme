// Command project-module emits a versioned Project Contract.
package main

import (
	"flag"
	"log"
	"os"
	"seme.local/reference/projectmodule"
)

func main() {
	outPath := flag.String("out", "", "optional output path")
	version := flag.Int("version", 1, "Project Contract version")
	flag.Parse()
	out := os.Stdout
	if *outPath != "" {
		file, err := os.Create(*outPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		out = file
	}
	if err := projectmodule.EmitVersion(out, *version); err != nil {
		log.Fatal(err)
	}
}
