package main

import (
	"encoding/json"
	"os"
	"strconv"

	control "example.test/go-uab-03"
)

func main() {
	if len(os.Args) != 2 {
		panic("usage: native-observations VECTORS.json")
	}
	var vectors struct {
		Valid []struct {
			Name, Start, Limit, AndIndex, OrIndex string
			Enabled                               bool
			Values                                []string
		} `json:"valid"`
		Malformed []struct {
			Name, Start, Limit, AndIndex, OrIndex string
			Enabled                               bool
			Values                                []string
		} `json:"malformed"`
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		panic(err)
	}
	observed := map[string]string{}
	for _, item := range vectors.Valid {
		start, err := strconv.ParseInt(item.Start, 10, 64)
		if err != nil {
			panic(err)
		}
		limit, err := strconv.ParseInt(item.Limit, 10, 64)
		if err != nil {
			panic(err)
		}
		andIndex, err := strconv.ParseInt(item.AndIndex, 10, 64)
		if err != nil {
			panic(err)
		}
		orIndex, err := strconv.ParseInt(item.OrIndex, 10, 64)
		if err != nil {
			panic(err)
		}
		values := make([]int64, len(item.Values))
		for i, raw := range item.Values {
			values[i], err = strconv.ParseInt(raw, 10, 64)
			if err != nil {
				panic(err)
			}
		}
		observed[item.Name] = strconv.FormatInt(control.Execute(start, limit, item.Enabled, values, andIndex, orIndex), 10)
	}
	for _, item := range vectors.Malformed {
		if item.Values == nil {
			continue
		}
		start, _ := strconv.ParseInt(item.Start, 10, 64)
		limit, _ := strconv.ParseInt(item.Limit, 10, 64)
		andIndex, _ := strconv.ParseInt(item.AndIndex, 10, 64)
		orIndex, _ := strconv.ParseInt(item.OrIndex, 10, 64)
		values := make([]int64, len(item.Values))
		for i, raw := range item.Values {
			values[i], _ = strconv.ParseInt(raw, 10, 64)
		}
		func() {
			accepted := true
			defer func() {
				if recover() != nil {
					accepted = false
				}
				if accepted {
					panic("forced runtime rejection accepted: " + item.Name)
				}
			}()
			control.Execute(start, limit, item.Enabled, values, andIndex, orIndex)
		}()
	}
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Valid     map[string]string `json:"valid"`
		Malformed int               `json:"malformed"`
	}{observed, len(vectors.Malformed)}); err != nil {
		panic(err)
	}
}
