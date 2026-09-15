package channelreceive

func Wait(done chan struct{}) {
	<-done
}
