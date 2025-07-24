package main

import (
	"encoding/binary"
	"math"
)

const (
	status_none = iota
	status_idle
	status_readyForTakeoff
	status_flying
	status_circling
	status_landing
)

type planeStatus struct {
	status   byte    // 1 byte
	battery  int     // 2 bytes
	speed    float32 // 4 bytes
	altitude float32 // 4 bytes
	// total: 11 bytes
}
type planeStatusCompressed [11]byte

func (p planeStatus) toBytes() planeStatusCompressed {
	var buf planeStatusCompressed
	buf[0] = p.status
	binary.BigEndian.PutUint16(buf[1:3], percentageToUint16(p.battery))
	binary.BigEndian.PutUint32(buf[3:7], math.Float32bits(p.speed))
	binary.BigEndian.PutUint32(buf[7:11], math.Float32bits(p.altitude))
	return buf
}

func planeStatusFromBytes(buf planeStatusCompressed) planeStatus {
	return planeStatus{
		status:   buf[0],
		battery:  percentageFromUint16(binary.BigEndian.Uint16(buf[1:3])),
		speed:    math.Float32frombits(binary.BigEndian.Uint32(buf[3:7])),
		altitude: math.Float32frombits(binary.BigEndian.Uint32(buf[7:11])),
	}
}

func percentageToUint16(i int) uint16 {
	if i < 0 {
		i = 0
	}
	if i > 100 {
		i = 100
	}
	return uint16(uint32(i) * math.MaxUint16 / 100)
}

func percentageFromUint16(u uint16) int {
	// reverse: multiply first, then divide
	return int(uint32(u) * 100 / math.MaxUint16)
}
