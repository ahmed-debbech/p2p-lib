package protocol

type Frame struct {
	Version        uint8  //1
	PayloadLength  uint16 //payload length without sequenceNumber, if 0 then it is just a heartbeat
	SequenceNumber uint64
	Payload        []byte
}
