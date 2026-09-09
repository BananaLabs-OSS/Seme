package main

import (
	"encoding/binary"
	"encoding/json"
	closures "example.test/go-uab-06"
	"os"
	"strconv"
)

type item struct {
	Name      string   `json:"name"`
	Arguments []string `json:"arguments"`
	Result    string   `json:"result"`
}

func main() {
	var v struct {
		Immutable, Mutable []item
		Malformed          []struct{ Name, Category string }
	}
	b, e := os.ReadFile(os.Args[1])
	if e != nil {
		panic(e)
	}
	if json.Unmarshal(b, &v) != nil {
		panic("vectors")
	}
	type observation struct {
		Valid    map[string]string `json:"valid"`
		Rejected map[string]bool   `json:"rejected"`
	}
	out := map[string]observation{"immutable": {map[string]string{}, map[string]bool{}}, "mutable": {map[string]string{}, map[string]bool{}}}
	for _, x := range v.Immutable {
		a := parse(x.Arguments)
		out["immutable"].Valid[x.Name] = strconv.FormatInt(closures.ImmutableRun(a[0], a[1]), 10)
	}
	for _, x := range v.Mutable {
		a := parse(x.Arguments)
		out["mutable"].Valid[x.Name] = strconv.FormatInt(closures.MutableRun(a[0], a[1], a[2]), 10)
	}
	for _, x := range v.Malformed {
		for _, entry := range []string{"immutable", "mutable"} {
			size := map[string]int{"immutable": 16, "mutable": 24}[entry]
			if x.Category == "short" {
				size--
			} else {
				size++
			}
			if _, ok := boundary(entry, make([]byte, size)); ok {
				panic("accepted " + x.Name)
			}
			o := out[entry]
			o.Rejected[x.Name] = true
			out[entry] = o
		}
	}
	json.NewEncoder(os.Stdout).Encode(out)
}
func boundary(entry string, request []byte) (int64, bool) {
	want := 16
	if entry == "mutable" {
		want = 24
	}
	if len(request) != want {
		return 0, false
	}
	args := make([]int64, want/8)
	for i := range args {
		args[i] = int64(binary.LittleEndian.Uint64(request[i*8 : i*8+8]))
	}
	if entry == "immutable" {
		return closures.ImmutableRun(args[0], args[1]), true
	}
	return closures.MutableRun(args[0], args[1], args[2]), true
}
func parse(v []string) []int64 {
	r := make([]int64, len(v))
	for i, x := range v {
		n, e := strconv.ParseInt(x, 10, 64)
		if e != nil {
			panic(e)
		}
		r[i] = n
	}
	return r
}
