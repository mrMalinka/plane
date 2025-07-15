package lora

import (
	"errors"
	"time"
)

// sends data and blocks until done or timeout
func (l *LoRa) Transmit(data []byte, timeout time.Duration) error {
	return errors.New("transmit not implemented")
}

// listens and returns next packet
func (l *LoRa) Receive(maxLen int, timeout time.Duration) ([]byte, error) {
	return nil, errors.New("receive not implemented")
}
