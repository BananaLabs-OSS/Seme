package main

import (
	"os"
	"seme.local/reference/languageservice"
)

func main() { languageservice.Emit(os.Stdout) }
