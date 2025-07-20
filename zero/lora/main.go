package lora

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

// SX127x
type LoRa struct {
	antennaMutex sync.Mutex

	spiConn  spi.Conn
	resetPin gpio.PinOut

	frequency uint32

	spreadingF byte
	bandwidth  uint32
}

// spiDev: default can be ""; reset: GPIO name; freqHz: e.g. 433e6
func New(spiDev, reset string, freqHz uint32) (*LoRa, error) {
	if _, err := host.Init(); err != nil {
		return nil, err
	}
	port, err := spireg.Open(spiDev)
	if err != nil {
		return nil, err
	}

	resetPin := gpioreg.ByName(reset)
	if resetPin == nil {
		return nil, errors.New("invalid GPIO pin name")
	}

	resetPin.Out(gpio.High)
	// 10MHz clock speed, mode 0, 8 bits per word
	conn, err := port.Connect(10*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, err
	}

	l := &LoRa{spiConn: conn, resetPin: resetPin, frequency: freqHz}
	if err := l.Reset(); err != nil {
		return nil, err
	}
	if err := l.Init(); err != nil {
		return nil, err
	}
	return l, nil
}

// pulses reset pin
func (l *LoRa) Reset() error {
	if err := l.resetPin.Out(gpio.Low); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)
	if err := l.resetPin.Out(gpio.High); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (l *LoRa) Init() error {
	ver, err := l.readReg(RegVersion)
	if err != nil || ver != 0x12 {
		return errors.New("LoRa module not found or unsupported version")
	}
	// enter sleep mode so 7th bit (mode) can be set to 1 (LoRa mode)
	l.writeReg(RegOpMode, ModeSleep)
	// write LoRa mode (included in all mode constants here)
	l.writeReg(RegOpMode, ModeSleep)
	// go into standby to set other registers
	l.writeReg(RegOpMode, ModeStandby)

	l.SetBandwidth(125000)
	l.SetCodingRate("4/8")
	l.SetSpreadingFactor(12)
	l.SetLowDataRateOptimize(true)
	l.SetLnaGain(LNA_G1, LNA_Boost3) // max range
	l.SetPreambleLength(12)
	l.SetAgc(true)
	l.SetCRC(true)

	// set frequency
	frf := uint64(l.frequency) * (1 << 19) / 32_000_000
	l.writeReg(RegFrMsb, byte(frf>>16))
	l.writeReg(RegFrMid, byte(frf>>8))
	l.writeReg(RegFrLsb, byte(frf))

	// set fifo memory spaces
	// tx is first 128 bytes, rx is last
	l.writeReg(RegFifoTxBaseAddr, 0)
	l.writeReg(RegFifoRxBaseAddr, 128)

	// set pa ramp to 50us
	l.writeReg(RegPaRamp, 0x08)

	// set payload length to almost half to be safe (because we split fifo in half)
	l.writeReg(RegMaxPayloadLength, 127)

	// reset irq
	l.writeReg(RegIrqFlags, 0xFF)

	return nil
}

// config

func (l *LoRa) SetBandwidth(bw uint32) error {
	reg, ok := BwValues[bw]
	if !ok {
		return errors.New("unsupported bandwidth")
	}
	l.bandwidth = bw
	existing, _ := l.readReg(RegModemConfig1)
	newVal := (existing & 0x0F) | reg
	return l.writeReg(RegModemConfig1, newVal)
}

func (l *LoRa) SetCodingRate(cr string) error {
	rVal, ok := CrValues[cr]
	if !ok {
		return errors.New("unsupported coding rate")
	}
	existing, _ := l.readReg(RegModemConfig1)
	newVal := (existing & 0xF1) | rVal
	return l.writeReg(RegModemConfig1, newVal)
}

// (6..12)
func (l *LoRa) SetSpreadingFactor(sf int) error {
	if sf < 6 || sf > 12 {
		return errors.New("spreading factor out of range")
	}
	val := byte((sf << 4) & 0xF0)
	existing, _ := l.readReg(RegModemConfig2)
	newVal := (existing & 0x0F) | val
	if err := l.writeReg(RegModemConfig2, newVal); err != nil {
		return err
	}
	l.spreadingF = byte(sf)
	return nil
}

// refer to page 109 of the sx127x datasheet (or search for RegPaConfig)
func (l *LoRa) SetTxPower(paBoost bool, maxPower byte, outputPower byte) error {
	if maxPower > 0b111 {
		maxPower = 0b111
	}
	if outputPower > 0b1111 {
		outputPower = 0b1111
	}
	val := byte(0b0_000_1000)
	if paBoost {
		val |= 0b1_000_0000
	}
	val |= maxPower
	val |= outputPower
	return l.writeReg(RegPaConfig, val)
}

func (l *LoRa) SetPreambleLength(len uint16) error {
	msb := byte(len >> 8)
	lsb := byte(len)
	if err := l.writeReg(RegPreambleMsb, msb); err != nil {
		return err
	}
	return l.writeReg(RegPreambleLsb, lsb)
}

func (l *LoRa) SetSyncWord(sw byte) error {
	return l.writeReg(RegSyncWord, sw)
}

func (l *LoRa) SetAgc(enable bool) error {
	existing, err := l.readReg(RegModemConfig3)
	if err != nil {
		return err
	}
	if enable {
		existing |= 0x04
	} else {
		existing &^= 0x04
	}
	return l.writeReg(RegModemConfig3, existing)
}

func (l *LoRa) SetLowDataRateOptimize(enable bool) error {
	existing, err := l.readReg(RegModemConfig3)
	if err != nil {
		return err
	}
	if enable {
		existing |= 0x08
	} else {
		existing &^= 0x08
	}
	return l.writeReg(RegModemConfig3, existing)
}

func (l *LoRa) SetLnaGain(gain, boost byte) error {
	validGains := []byte{LNA_G1, LNA_G2, LNA_G3, LNA_G4, LNA_G5}
	validGain := false
	for _, v := range validGains {
		if gain == v {
			validGain = true
			break
		}
	}
	if !validGain {
		return errors.New("invalid LNA gain")
	}
	if boost > 3 {
		return errors.New("invalid LNA boost")
	}

	ex := gain | boost
	return l.writeReg(RegLna, ex)
}

func (l *LoRa) SetCRC(enable bool) error {
	existing, err := l.readReg(RegModemConfig2)
	if err != nil {
		return err
	}
	if enable {
		existing |= 0x04
	} else {
		existing &^= 0x04
	}
	return l.writeReg(RegModemConfig2, existing)
}

// enables/disabled overcurrent protection
func (l *LoRa) SetOcp(enable bool) error {
	val := byte(0x20) // default OCP on, trim ~100mA
	if !enable {
		val |= 0x0F // disable
	}
	return l.writeReg(RegOcp, val)
}

func (l *LoRa) SetReceiveTimeout(d time.Duration) error {
	timeoutSec := d.Seconds()
	Ts := math.Pow(2, float64(l.spreadingF)) / float64(l.bandwidth)
	symbols := uint16(math.Ceil(timeoutSec / Ts))
	if symbols > 0x3FF {
		symbols = 0x3FF
	}
	return l.SetSymbolTimeout(symbols)
}

func (l *LoRa) SetSymbolTimeout(timeout uint16) error {
	// write lower 8 bits
	if err := l.writeReg(RegSymbTimeoutLsb, byte(timeout&0xFF)); err != nil {
		return err
	}
	// write upper bits (for some reason theyre in the 2nd modem config?)
	existing, err := l.readReg(RegModemConfig2)
	if err != nil {
		return err
	}
	msb := byte((timeout >> 8) & 0x07)
	newVal := (existing & 0xF8) | msb
	return l.writeReg(RegModemConfig2, newVal)
}

func (l *LoRa) GetSignalStrength() (int, error) {
	rssiRaw, err := l.readReg(RegPktRssiValue)
	return int(rssiRaw) - 164, err
}

func (l *LoRa) FormatConfig() (string, error) {
	frfMsb, err := l.readReg(RegFrMsb)
	if err != nil {
		return "", err
	}
	frfMid, err := l.readReg(RegFrMid)
	if err != nil {
		return "", err
	}
	frfLsb, err := l.readReg(RegFrLsb)
	if err != nil {
		return "", err
	}
	frf := uint64(frfMsb)<<16 | uint64(frfMid)<<8 | uint64(frfLsb)
	actualFreq := uint32(frf * 32_000_000 >> 19)

	modemCfg1, err := l.readReg(RegModemConfig1)
	if err != nil {
		return "", err
	}
	bwReg := modemCfg1 & 0xF0
	bw := uint32(0)
	for hz, reg := range BwValues {
		if reg == bwReg {
			bw = hz
			break
		}
	}

	modemCfg2, err := l.readReg(RegModemConfig2)
	if err != nil {
		return "", err
	}
	sf := (modemCfg2 & 0xF0) >> 4

	crReg := modemCfg1 & 0x0E
	cr := "?"
	for ratio, reg := range CrValues {
		if reg == crReg {
			cr = ratio
			break
		}
	}

	syncWord, err := l.readReg(RegSyncWord)
	if err != nil {
		return "", err
	}

	preambleMsb, err := l.readReg(RegPreambleMsb)
	if err != nil {
		return "", err
	}
	preambleLsb, err := l.readReg(RegPreambleLsb)
	if err != nil {
		return "", err
	}
	preamble := uint16(preambleMsb)<<8 | uint16(preambleLsb)

	crcEnabled := (modemCfg2 & 0x04) != 0

	modemCfg3, err := l.readReg(RegModemConfig3)
	if err != nil {
		return "", err
	}
	ldro := (modemCfg3 & 0x08) != 0

	lnaReg, err := l.readReg(RegLna)
	if err != nil {
		return "", err
	}
	lnaGain := (lnaReg & 0xE0) >> 5
	lnaBoost := lnaReg & 0x03

	return fmt.Sprintf(
		"Freq:%d|BW:%d|SF:%d|CR:%s|SW:0x%02X|PL:%d|CRC:%t|LDRO:%t|LNA:%d/%d",
		actualFreq, bw, sf, cr, syncWord, preamble, crcEnabled, ldro, lnaGain, lnaBoost,
	), nil
}

func (l *LoRa) writeReg(reg, val byte) error {
	return l.spiConn.Tx([]byte{reg | 0x80, val}, nil)
}

func (l *LoRa) readReg(reg byte) (byte, error) {
	buf := make([]byte, 2)
	if err := l.spiConn.Tx([]byte{reg & 0x7F, 0x00}, buf); err != nil {
		return 0, err
	}
	return buf[1], nil
}
