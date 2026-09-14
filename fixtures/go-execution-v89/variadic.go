package variadic

import "fmt"

func Expand(values []any) string {
	return fmt.Sprint(values...)
}
