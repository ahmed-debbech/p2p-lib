package protocol

import (
	"errors"
	"log"

	"github.com/ahmed-debbech/p2p-lib/utils"
)

func ParseFrame(binFrame []byte) (Frame, error) {

	frame := Frame{}
	//check version
	if binFrame[0] != uint8(1) {
		return Frame{}, errors.New("protocol version mismatch!")
	}

	frame.Version = 1
	frame.PayloadLength = utils.BytesToUInt16(binFrame[1:3])
	frame.SequenceNumber = utils.BytesToUInt64(binFrame[3:11])

	if frame.PayloadLength > 0 && len(binFrame) <= 10 {
		log.Println("frame has payload length > 0 but payload is empty.")
		return Frame{}, errors.New("frame has payload length > 0 but payload is empty.")
	}
	if frame.PayloadLength > 0 {
		frame.Payload = binFrame[11:frame.PayloadLength]
	} else {
		frame.Payload = nil
	}
	return frame, nil
}

func BuildFrameToBinary(sequenceNumber uint64, payload []byte) ([]byte, error) {
	frame := Frame{}
	frame.Version = uint8(1)
	if len(payload) >= 65535 {
		return nil, errors.New("payload len is too big")
	}
	frame.PayloadLength = uint16(len(payload))
	frame.SequenceNumber = sequenceNumber
	frame.Payload = payload

	b := serializeToBinary(frame)
	return b, nil
}

func serializeToBinary(frame Frame) []byte {
	bin := make([]byte, 0)
	bin = append(bin, utils.UInt8ToBinary(frame.Version)...)
	bin = append(bin, utils.UInt16ToBinary(frame.PayloadLength)...)
	bin = append(bin, utils.UInt64ToBinary(frame.SequenceNumber)...)
	bin = append(bin, frame.Payload...)
	return bin
}
