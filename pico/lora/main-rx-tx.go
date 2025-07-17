package lora

import (
	"errors"
	"time"
)

const pollRate = 5 * time.Millisecond

func (l *LoRa) Transmit(data []byte, timeout time.Duration) error {
	if len(data) > 127 {
		return errors.New("payload too large")
	}

	// make sure were in standby mode
	if err := l.writeReg(RegOpMode, ModeStandby); err != nil {
		return err
	}

	// clear irq flags
	l.writeReg(RegIrqFlags, 0xFF)
	defer l.writeReg(RegIrqFlags, 0xFF)

	// set FIFO address pointer to where tx starts
	fifoTxBaseAddr, _ := l.readReg(RegFifoTxBaseAddr)
	l.writeReg(RegFifoAddrPtr, fifoTxBaseAddr)

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

	// set to tx single mode
	if err := l.writeReg(RegOpMode, ModeTx); err != nil {
		return err
	}
	defer l.writeReg(RegOpMode, ModeStandby)

	// poll irq until tx is done
	for {
		irq, err := l.readReg(RegIrqFlags)
		if err != nil {
			return err
		}
		if irq&IrqTxDone != 0 {
			break
		}
		time.Sleep(pollRate)
	}

	// clear irq flags
	l.writeReg(RegIrqFlags, 0xFF)
	return nil
}

func (l *LoRa) Receive(maxLen int, timeout time.Duration) ([]byte, error) {
	// make sure were in standby mode
	if err := l.writeReg(RegOpMode, ModeStandby); err != nil {
		return nil, err
	}

	// clear irq flags
	l.writeReg(RegIrqFlags, 0xFF)
	defer l.writeReg(RegIrqFlags, 0xFF)

	l.SetReceiveTimeout(timeout)

	// set to rx single mode
	if err := l.writeReg(RegOpMode, ModeRxSingle); err != nil {
		return nil, err
	}
	defer l.writeReg(RegOpMode, ModeStandby)

poll: // poll irq flags register
	for {
		irq, err := l.readReg(RegIrqFlags)
		if err != nil {
			return nil, err
		}

		switch {
		case irq&IrqRxDone != 0 && irq&IrqPayloadCrcError == 0:
			break poll // valid packet
		case irq&IrqPayloadCrcError != 0:
			return nil, errors.New("crc error")
		case irq&IrqRxTimeout != 0:
			return nil, errors.New("rx timeout")
		}
		time.Sleep(pollRate)
	}

	nb, err := l.readReg(RegRxNbBytes)
	if err != nil {
		return nil, err
	}
	if int(nb) > maxLen {
		return nil, errors.New("packet too large")
	}

	// set FIFO address pointer to where the packet begins
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
	return buf, nil
}
