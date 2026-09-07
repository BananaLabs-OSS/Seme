package controltext

func Decide(enabled bool, value, limit int64) bool {
	if enabled || ("λ"+"!" == "never") {
		if value <= limit {
			return "exact"+"-text" == "exact-text"
		}
		return false
	}
	return false
}
