package method

type Runner struct{}

func (r *Runner) Sum(values ...int64) int64 {
	var total int64
	for _, value := range values {
		total += value
	}
	return total
}

func Run(r *Runner, values []int64) int64 {
	return r.Sum(values...)
}
