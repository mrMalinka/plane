package main

import (
	"errors"
	"fmt"
)

const (
	payloadType_error byte = iota
	payloadType_bulk       // all plane status data
	payloadType_rssi       // exclusive to pico -> phone
	// for manual control only
	payloadType_joystick
	payloadType_throttle

	payloadType_errorInternal = 0xFF
)

func newPacket(header byte, payload []byte) []byte {
	// these packets are meant for everything from
	// lora to usb and as such do not have to be modified when forwarded
	// packet structure:
	//  header - 2 bytes
	//    first - length of the full packet including header
	//    second - data type of payload
	//
	//  payload - n bytes
	return append([]byte{byte(len(payload) + 2), header}, payload...)
}

func parsePacket(packet []byte) (length uint8, payloadType byte, payload []byte, err error) {
	if len(packet) == 0 {
		return 0xFF, 0, nil, errors.New("packet is empty")
	}
	length = packet[0]
	payloadType = packet[1]
	payload = packet[2:]
	return
}

func formatErrorPacket(while string, err error) []byte {
	return newPacket(
		payloadType_error,
		fmt.Appendf(nil, "error while %s: %v", while, err.Error()),
	)
}
