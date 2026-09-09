package main

import (
	"encoding/json"
	"os"
	"strconv"

	aggregate "example.test/go-uab-02-aggregate"
)

func main() {
	observe := func(bias int64, fixed [2]int64, slice []int64, entries map[int64]int64, key int64) string {
		return strconv.FormatInt(aggregate.Observe(aggregate.Bounds{Bias: bias}, fixed, slice, entries, key), 10)
	}
	values := map[string]string{
		"present":  observe(5, [2]int64{2, 7}, []int64{1, 2, 3}, map[int64]int64{-2: 4, 9: 11}, 9),
		"missing":  observe(5, [2]int64{2, 7}, []int64{1, 2, 3}, map[int64]int64{-2: 4, 9: 11}, 8),
		"wrapping": observe(9223372036854775807, [2]int64{0, 1}, nil, nil, 0),
	}
	if err := json.NewEncoder(os.Stdout).Encode(values); err != nil {
		panic(err)
	}
}
