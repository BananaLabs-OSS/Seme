package main

import (
	"encoding/json"
	"os"
	"strconv"

	collections "example.test/go-uab-04"
)

type vector struct {
	Name, Index, Replacement, Appended, RemoveIndex, KeepKey, RemoveKey, Result string
	Values                                                                      []string
}

func parse(item vector) ([]int64, [6]int64) {
	values := make([]int64, len(item.Values))
	for index, raw := range item.Values {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			panic(err)
		}
		values[index] = value
	}
	raw := [6]string{item.Index, item.Replacement, item.Appended, item.RemoveIndex, item.KeepKey, item.RemoveKey}
	var args [6]int64
	for index, value := range raw {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			panic(err)
		}
		args[index] = parsed
	}
	return values, args
}

func evaluate(item vector) int64 {
	values, args := parse(item)
	return collections.Evaluate(values, args[0], args[1], args[2], args[3], args[4], args[5])
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: native-observations VECTORS.json")
	}
	var vectors struct{ Valid, Malformed []vector }
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		panic(err)
	}
	observed := map[string]string{}
	rejected := map[string]bool{}
	for _, item := range vectors.Valid {
		observed[item.Name] = strconv.FormatInt(evaluate(item), 10)
	}
	for _, item := range vectors.Malformed {
		func() {
			accepted := true
			defer func() {
				if recover() != nil {
					accepted = false
					rejected[item.Name] = true
				}
				if accepted {
					panic("forced runtime rejection accepted: " + item.Name)
				}
			}()
			evaluate(item)
		}()
	}
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Valid     map[string]string `json:"valid"`
		Malformed int               `json:"malformed"`
		Rejected  map[string]bool   `json:"rejected"`
	}{observed, len(vectors.Malformed), rejected}); err != nil {
		panic(err)
	}
}
