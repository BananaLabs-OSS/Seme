// Command provider-patch-fixture composes one imported Provider Contract v1
// graph with Patch Module v1 declarations and a canonical rename transaction.
// It is conformance-fixture construction, not runtime authority.
package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type graph struct {
	revision string
	count    int
	entities []entity
}
type entity struct{ id, text string }

func main() {
	if len(os.Args) != 6 {
		fatal("usage: provider-patch-fixture PATCH_MODULE_G1 PROGRAM_G1 TARGET EXPECTED REPLACEMENT")
	}
	patch := read(os.Args[1])
	program := read(os.Args[2])
	target := os.Args[3]
	if len(target) != 32 {
		fatal("target must be 16-byte hex identity")
	}
	if _, err := hex.DecodeString(target); err != nil {
		fatal("invalid target identity")
	}
	entities := append(append([]entity{}, patch.entities...), program.entities...)
	entities = append(entities,
		entity{id(0x6000), fmt.Sprintf("en %s %s 1 0\n", id(0x6000), id(0x6100))},
		entity{id(0x6001), fmt.Sprintf("en %s %s 1 3\nfi %s rf %s\nfi %s by %s\nfi %s li 1\nrf %s\n", id(0x6001), id(0x5010), id(0x5100), id(0x6000), id(0x5101), program.revision, id(0x5102), id(0x6002))},
		entity{id(0x6002), fmt.Sprintf("en %s %s 1 4\nfi %s by %s\nfi %s by %s\nfi %s by %s\nfi %s by %s\n", id(0x6002), id(0x5011), id(0x5110), target, id(0x5111), id(0x7130), id(0x5112), text(os.Args[4]), id(0x5113), text(os.Args[5]))},
	)
	sort.Slice(entities, func(i, j int) bool { return entities[i].id < entities[j].id })
	for i := 1; i < len(entities); i++ {
		if entities[i-1].id == entities[i].id {
			fatal("duplicate composed identity " + entities[i].id)
		}
	}
	fmt.Printf("# Generated Provider Contract v1 Patch workspace.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n", id(0x5000), program.revision, len(entities))
	for _, entity := range entities {
		fmt.Println()
		fmt.Print(entity.text)
	}
}
func read(path string) graph {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var result graph
	var current *entity
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "rv ") {
			result.revision = strings.TrimPrefix(line, "rv ")
		}
		if strings.HasPrefix(line, "ec ") {
			parsed, parseErr := strconv.Atoi(strings.TrimPrefix(line, "ec "))
			if parseErr != nil {
				panic(parseErr)
			}
			result.count = parsed
		}
		if strings.HasPrefix(line, "en ") {
			fields := strings.Fields(line)
			result.entities = append(result.entities, entity{id: fields[1]})
			current = &result.entities[len(result.entities)-1]
		}
		if current != nil {
			current.text += line + "\n"
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	if result.revision == "" {
		fatal("missing revision in " + path)
	}
	return result
}
func id(n uint64) string { return fmt.Sprintf("%032x", n) }
func text(value string) string {
	if value == "" {
		return "-"
	}
	return hex.EncodeToString([]byte(value))
}
func fatal(message string) { fmt.Fprintln(os.Stderr, "provider-patch-fixture:", message); os.Exit(64) }
