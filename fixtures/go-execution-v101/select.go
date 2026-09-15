package nativeselect

func Wait(done, tick chan struct{}) int64 {
	for {
		select {
		case <-done:
			return 1
		case <-tick:
			return 2
		}
	}
}
