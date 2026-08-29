package greeting

func Greeting(name string) string {
	return "Hello, " + name
}

func UseGreeting(name string) string {
	return Greeting(name)
}

