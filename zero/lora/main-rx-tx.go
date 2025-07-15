package lora

import (
	"errors"
	"time"
)

// Transmit sends data and blocks until transmission is done or timeout occurs
func (l *LoRa) Transmit(data []byte, timeout time.Duration) error {
	// set standby mode
	if err := l.writeReg(RegOpMode, ModeStandby); err != nil {
		return err
	}
	// reset IRQ flags
	l.writeReg(RegIrqFlags, 0xFF)

	// set FIFO address pointer to TX base
	l.writeReg(RegFifoAddrPtr, l.readRegOrZero(RegFifoTxBaseAddr))

	// write payload to FIFO
	for _, b := range data {
		if err := l.writeReg(RegFifo, b); err != nil {
			return err
		}
	}
	// set payload length
	if err := l.writeReg(RegPayloadLength, byte(len(data))); err != nil {
		return err
	}

	// set DIO0 to TxDone
	if err := l.writeReg(RegDioMapping1, DIO0_TxDone); err != nil {
		return err
	}

	// enter TX mode
	if err := l.writeReg(RegOpMode, ModeTx); err != nil {
		return err
	}

	// wait for DIO0 rising edge
	if !l.dio0Pin.WaitForEdge(timeout) {
		return errors.New("transmit timeout")
	}

	// clear IRQ flags
	l.writeReg(RegIrqFlags, 0xFF)
	return nil
}

// reads a register or returns 0 on error
func (l *LoRa) readRegOrZero(reg byte) byte {
	b, _ := l.readReg(reg)
	return b
}

func (l *LoRa) Receive(maxLen int, timeout time.Duration) ([]byte, error) {
	// reset IRQ flags
	l.writeReg(RegIrqFlags, 0xFF)

	// set FIFO address pointer to RX base
	l.writeReg(RegFifoAddrPtr, l.readRegOrZero(RegFifoRxBaseAddr))

	// map DIO0 to RxDone (00)
	if err := l.writeReg(RegDioMapping1, RegFifo); err != nil {
		return nil, err
	}

	// enter single RX mode
	if err := l.writeReg(RegOpMode, ModeRxSingle); err != nil {
		return nil, err
	}

	// wait for DIO0 rising edge
	if !l.dio0Pin.WaitForEdge(timeout) {
		return nil, errors.New("receive timeout")
	}

	// read IRQ flags
	irq, err := l.readReg(RegIrqFlags)
	if err != nil {
		return nil, err
	}

	// check CRC error
	if irq&0x20 != 0 {
		l.writeReg(RegIrqFlags, 0xFF)
		return nil, errors.New("CRC error")
	}

	// read number of received bytes
	nb, err := l.readReg(RegRxNbBytes)
	if err != nil {
		return nil, err
	}
	if int(nb) > maxLen {
		return nil, errors.New("packet too large")
	}

	// set FIFO address pointer to current RX address
	addr, _ := l.readReg(RegFifoRxCurrent)
	l.writeReg(RegFifoAddrPtr, addr)

	// read payload
	buf := make([]byte, nb)
	for i := range buf {
		b, err := l.readReg(RegFifo)
		if err != nil {
			return nil, err
		}
		buf[i] = b
	}

	// clear IRQ flags
	l.writeReg(RegIrqFlags, 0xFF)
	return buf, nil
}
