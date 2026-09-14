package packagebinding

import "os"

func Output() *os.File {
	return os.Stdout
}
