package main

import (
	"errors"
	"fmt"
)

const (
	header_error byte = iota
	header_bulk       // all plane status data
	// for manual control only
	header_joystick
	header_throttle
)

func newPacket(header byte, payload []byte) []byte {
	return append([]byte{header}, payload...)
}

func parsePacket(packet []byte) (header byte, payload []byte, err error) {
	if len(packet) == 0 {
		return 0xFF, nil, errors.New("packet is empty")
	}
	header = packet[0]
	payload = packet[1:]
	return header, payload, nil
}

func formatErrorPacket(while string, err error) []byte {
	return newPacket(
		header_error,
		[]byte(fmt.Sprintf("error while %s: %v", while, err.Error())),
	)
}
