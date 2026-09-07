package main

import (
	"os"
	"seme.local/reference/executionmodule"
)

func main() { if err := executionmodule.Emit(os.Stdout, 13); err != nil { panic(err) } }
