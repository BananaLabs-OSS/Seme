package ordinary

// This comment and spacing must survive a semantic rename.
func Greeting(name string) string {
	return "Hello, " + name
}

func Message() string {
	return Greeting("Seme")
}
