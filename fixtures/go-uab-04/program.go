package collections

import (
	"maps"
	"slices"
)

func Sum(values []int64) int64 {
	total := int64(0)
	for _, value := range values {
		total += value
	}
	return total
}

func Evaluate(values []int64, index, replacement, appended, removeIndex, keepKey, removeKey int64) int64 {
	built := []int64{values[0], values[1]}
	updated := slices.Replace(slices.Clone(built), int(index), int(index)+1, replacement)
	appendedValues := append(updated, appended)
	removed := slices.Delete(slices.Clone(appendedValues), int(removeIndex), int(removeIndex)+1)
	total := Sum(removed)
	empty := map[int64]int64{}
	withKeep := func(input map[int64]int64, key, value int64) map[int64]int64 {
		output := maps.Clone(input)
		output[key] = value
		return output
	}(empty, keepKey, total)
	withRemove := func(input map[int64]int64, key, value int64) map[int64]int64 {
		output := maps.Clone(input)
		output[key] = value
		return output
	}(withKeep, removeKey, removed[index])
	withoutRemove := func(input map[int64]int64, key int64) map[int64]int64 {
		output := maps.Clone(input)
		delete(output, key)
		return output
	}(withRemove, removeKey)
	return int64(len(removed)) + removed[index] + withoutRemove[keepKey]
}
