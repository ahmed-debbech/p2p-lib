package model

type Peer struct {
	ID            string
	IP            string
	Port          string
	TimeoutInSec  int
	FirstAppeared int64 // a timestamp unix representing when the peer first appeared on stun to start the time out based on TimeoutInSec
}
