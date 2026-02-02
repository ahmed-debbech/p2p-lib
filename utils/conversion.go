package utils

import (
	"bytes"
	"encoding/binary"
)

func BytesToUInt64(bin []byte) uint64 {
	var ret uint64
	buf := bytes.NewBuffer(bin)
	binary.Read(buf, binary.BigEndian, &ret)
	return ret
}

func UInt64ToBinary(i uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

func BytesToUInt8(bin []byte) uint8 {
	var ret uint8
	buf := bytes.NewBuffer(bin)
	binary.Read(buf, binary.BigEndian, &ret)
	return ret
}

func BytesToUInt16(bin []byte) uint16 {
	var ret uint16
	buf := bytes.NewBuffer(bin)
	binary.Read(buf, binary.BigEndian, &ret)
	return ret
}

func UInt8ToBinary(i uint8) []byte {
	return []byte{byte(i)}
}

func UInt16ToBinary(i uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(i))
	return b
}
