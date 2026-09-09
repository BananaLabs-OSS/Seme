package effects

import "log"

func Observe(first, second bool) bool {
	log.Print(first)
	log.Print(second)
	return second
}
