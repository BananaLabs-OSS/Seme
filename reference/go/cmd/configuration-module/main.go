package main

import (
	"flag"
	"log"
	"os"
	"seme.local/reference/configurationmodule"
)

func main() {
	version := flag.Int("version", 1, "Configuration Contract version")
	flag.Parse()
	if err := configurationmodule.EmitVersion(os.Stdout, *version); err != nil {
		log.Fatal(err)
	}
}
