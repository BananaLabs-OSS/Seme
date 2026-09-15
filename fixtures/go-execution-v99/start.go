package concurrentstart

func Work(value int64) {}

func Start(value int64) {
	go Work(value)
}
