// Command patch-fixture emits canonical-construction G1 workspaces used to
// prove Patch Module v1. It is a differential fixture generator, not authority.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func id(n uint64) string { return fmt.Sprintf("%032x", n) }

func main() {
	mode := "valid"
	if len(os.Args) == 2 {
		mode = os.Args[1]
	} else if len(os.Args) != 1 {
		fatal("usage: patch-fixture [valid|stale|precondition|duplicate]")
	}
	if mode != "valid" && mode != "stale" && mode != "precondition" && mode != "duplicate" {
		fatal("unknown mode")
	}

	f, err := os.Open("../../modules/patch/v1/module.g1")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var declarations []string
	inEntities := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "en ") {
			inEntities = true
		}
		if inEntities {
			declarations = append(declarations, line)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	base := id(0x5500)
	if mode == "stale" {
		base = id(0x5501)
	}
	expected := "4772656574696e67" // Greeting
	if mode == "precondition" {
		expected = "57726f6e67" // Wrong
	}
	operations := "li 1\nrf " + id(0x5202)
	if mode == "duplicate" {
		operations = "li 2\nrf " + id(0x5202) + "\nrf " + id(0x5202)
	}

	fmt.Printf("# Generated Patch Module v1 %s workspace.\nve 1\nmo %s\nrv %s\npc 0\nec 16\n\n", mode, id(0x5000), id(0x5500))
	for _, line := range declarations {
		fmt.Println(line)
	}
	fmt.Printf(`
en %s %s 1 0

en %s %s 1 3
fi %s rf %s
fi %s by %s
fi %s %s

en %s %s 1 4
fi %s rf %s
fi %s by %s
fi %s by %s
fi %s by %s

en %s %s 1 2
fi %s by 52656e616d6561626c65
fi %s li 1
rf %s

en %s %s 1 4
fi %s by 6e616d65
fi %s rc 1
fi %s uu 4
fi %s uu 0
fi %s uu 1

en %s %s 1 1
fi %s by 4772656574696e67
`, id(0x5200), id(0x6000),
		id(0x5201), id(0x5010), id(0x5100), id(0x5200), id(0x5101), base, id(0x5102), operations,
		id(0x5202), id(0x5011), id(0x5110), id(0x5400), id(0x5111), id(0x5310), id(0x5112), expected, id(0x5113), "57656c636f6d65",
		id(0x5300), id(0x10), id(0x100), id(0x101), id(0x5310),
		id(0x5310), id(0x11), id(0x110), id(0x111), id(0x2000), id(0x112), id(0x113),
		id(0x5400), id(0x5300), id(0x5310))
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "patch-fixture:", message)
	os.Exit(64)
}
