package collectionupdate

import "slices"

func UpdateAndAppend(values []int64, index int64, replacement int64, appended int64) []int64 {
	return append(slices.Replace(slices.Clone(values), int(index), int(index)+1, replacement), appended)
}
