package main

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"strconv"

	dispatch "example.test/go-uab-05"
)

func main() {
	if len(os.Args) != 2 {
		panic("usage: native-observations VECTORS.json")
	}
	var vectors struct {
		Valid []struct {
			Name, Amount, Value, Result string
			Scale                       bool
		}
		Malformed []struct{ Name, Category string }
	}
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
		amount, err := strconv.ParseInt(item.Amount, 10, 64)
		if err != nil {
			panic(err)
		}
		value, err := strconv.ParseInt(item.Value, 10, 64)
		if err != nil {
			panic(err)
		}
		observed[item.Name] = strconv.FormatInt(dispatch.Dispatch(item.Scale, amount, value), 10)
	}
	base := make([]byte, 17)
	base[0] = 0
	binary.LittleEndian.PutUint64(base[1:9], 3)
	binary.LittleEndian.PutUint64(base[9:17], 7)
	for _, item := range vectors.Malformed {
		request := append([]byte(nil), base...)
		switch item.Category {
		case "empty":
			request = nil
		case "short":
			request = request[:16]
		case "long":
			request = append(request, 0)
		case "selector":
			request[0] = 2
		default:
			panic("unknown malformed category")
		}
		if _, ok := invokeBoundary(request); ok {
			panic("native accepted " + item.Name)
		}
		rejected[item.Name] = true
	}
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Valid    map[string]string `json:"valid"`
		Rejected map[string]bool   `json:"rejected"`
	}{observed, rejected}); err != nil {
		panic(err)
	}
}

func invokeBoundary(request []byte) (int64, bool) {
	if len(request) != 17 || request[0] > 1 {
		return 0, false
	}
	amount := int64(binary.LittleEndian.Uint64(request[1:9]))
	value := int64(binary.LittleEndian.Uint64(request[9:17]))
	return dispatch.Dispatch(request[0] == 1, amount, value), true
}
