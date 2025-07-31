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
	battery  float32 // 4 bytes, here it's a float32 [0..100], but compressed, it's an uint32 [0..2^32]
	speed    float32 // 4 bytes
	altitude float32 // 4 bytes

	latitude, longitude float64 // 16 bytes
	// total: 29 bytes
}
type planeStatusCompressed [29]byte

func (p planeStatus) toBytes() planeStatusCompressed {
	var buf planeStatusCompressed
	buf[0] = p.status
	binary.BigEndian.PutUint32(buf[1:5], percentageToUint32(p.battery))
	binary.BigEndian.PutUint32(buf[5:9], math.Float32bits(p.speed))
	binary.BigEndian.PutUint32(buf[9:13], math.Float32bits(p.altitude))
	binary.BigEndian.PutUint64(buf[13:21], math.Float64bits(p.latitude))
	binary.BigEndian.PutUint64(buf[21:29], math.Float64bits(p.longitude))
	return buf
}

func planeStatusFromBytes(buf planeStatusCompressed) planeStatus {
	return planeStatus{
		status:    buf[0],
		battery:   percentageFromUint32(binary.BigEndian.Uint32(buf[1:5])),
		speed:     math.Float32frombits(binary.BigEndian.Uint32(buf[5:9])),
		altitude:  math.Float32frombits(binary.BigEndian.Uint32(buf[9:13])),
		latitude:  math.Float64frombits(binary.BigEndian.Uint64(buf[13:21])),
		longitude: math.Float64frombits(binary.BigEndian.Uint64(buf[21:29])),
	}
}

func percentageToUint32(f float32) uint32 {
	// clamp [0..100]
	f = min(max(f, 0), 100)
	return uint32(float64(f/100) * math.MaxUint32)
}

func percentageFromUint32(u uint32) float32 {
	return float32((float64(u) / math.MaxUint32) * 100)
}
