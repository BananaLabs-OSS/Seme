package method

type Router struct {
	Base int64
}

func (r *Router) Resolve(value int64) int64 {
	return r.Base + value
}

func Bind(r *Router) func(int64) int64 {
	return r.Resolve
}
