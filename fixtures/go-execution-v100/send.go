package channelsend

func Send(done chan struct{}) {
	done <- struct{}{}
}
