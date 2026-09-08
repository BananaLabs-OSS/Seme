package interfacedispatch

type Adjuster interface {
	Adjust(value int64) int64
}

type OffsetAdjuster struct {
	Offset int64
}

func (adjuster OffsetAdjuster) Adjust(value int64) int64 {
	return value + adjuster.Offset
}

type ScaleAdjuster struct {
	Factor int64
}

func (adjuster ScaleAdjuster) Adjust(value int64) int64 {
	return value * adjuster.Factor
}

func Apply(adjuster Adjuster, value int64) int64 {
	return adjuster.Adjust(value)
}

func Dispatch(useScale bool, amount, value int64) int64 {
	if useScale {
		return Apply(ScaleAdjuster{Factor: amount}, value)
	}
	return Apply(OffsetAdjuster{Offset: amount}, value)
}
