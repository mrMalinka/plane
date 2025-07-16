package lora

import (
	"errors"
	"machine"
	"time"
)

// SX127x
type LoRa struct {
	spiConn    machine.SPI
	csPin      machine.Pin // chip select out
	resetPin   machine.Pin // out
	dio0Pin    machine.Pin // in
	frequency  uint32      // in Hz
	bandwidth  uint32
	codingRate string
	spreadingF byte
	txPower    int // in dBm, NOT mW
}

// spiDev: SPI0 or SPI1; sdo, sdi, sck, cs, reset, dio0: GPIO indexes; freqHz: e.g. 433e6
func New(spiDev machine.SPI, sdo, sdi, sck, cs, reset, dio0 uint8, freqHz uint32) (*LoRa, error) {
	csPin := machine.Pin(cs)
	csPin.Configure(machine.PinConfig{
		Mode: machine.PinOutput,
	})
	resetPin := machine.Pin(reset)
	resetPin.Configure(machine.PinConfig{
		Mode: machine.PinOutput,
	})
	dio0Pin := machine.Pin(dio0)
	dio0Pin.Configure(machine.PinConfig{
		Mode: machine.PinInputPulldown,
	})

	resetPin.High()
	// rising edge not supported?
	// dio0Pin.In(gpio.PullDown, gpio.RisingEdge)

	spiDev.Configure(machine.SPIConfig{
		Frequency: 10 * machine.MHz,
		Mode:      0,
		SDO:       machine.Pin(sdo),
		SDI:       machine.Pin(sdi),
		SCK:       machine.Pin(sck),
	})

	l := &LoRa{spiConn: spiDev, csPin: csPin, resetPin: resetPin, dio0Pin: dio0Pin, frequency: freqHz}
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
	l.resetPin.Low()
	time.Sleep(10 * time.Millisecond)
	l.resetPin.High()
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (l *LoRa) Init() error {
	ver, err := l.readReg(RegVersion)
	if err != nil || ver != 0x12 {
		return errors.New("LoRa module not found or unsupported version")
	}
	// sleep to configure
	l.writeReg(RegOpMode, ModeSleep)

	// set default modem: BW=125k, CR=4/5, SF=7
	l.SetBandwidth(125000)
	l.SetCodingRate("4/5")
	l.SetSpreadingFactor(7)
	l.SetLowDataRateOptimize(false)

	// enable crc
	l.SetCRC(true)

	// set frequency
	frf := uint64(l.frequency) * (1 << 19) / 32_000_000
	l.writeReg(RegFrMsb, byte(frf>>16))
	l.writeReg(RegFrMid, byte(frf>>8))
	l.writeReg(RegFrLsb, byte(frf))

	// set fifo memory spaces
	// tx is first 128 bytes, rx is last
	l.writeReg(RegFifoTxBaseAddr, 0x00)
	l.writeReg(RegFifoRxBaseAddr, 0x80)

	// standby
	l.writeReg(RegOpMode, ModeStandby)
	return nil
}

// config

func (l *LoRa) SetBandwidth(bw uint32) error {
	reg, ok := BwValues[bw]
	if !ok {
		return errors.New("unsupported bandwidth")
	}
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

func (l *LoRa) SetTxPower(pwr int) error {
	if pwr < 2 || pwr > 17 {
		return errors.New("power out of range")
	}
	sel := byte(0x00)
	val := sel | byte(pwr+1)
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

func (l *LoRa) SetLowDataRateOptimize(enable bool) error {
	existing1, _ := l.readReg(RegModemConfig1)
	existing2, _ := l.readReg(RegModemConfig2)
	if enable {
		existing1 |= 0x01
		existing2 |= 0x08
	} else {
		existing1 &^= 0x01
	}
	if err := l.writeReg(RegModemConfig1, existing1); err != nil {
		return err
	}
	return l.writeReg(RegModemConfig2, existing2)
}

func (l *LoRa) SetLnaGain(g byte) error {
	if g > 5 {
		return errors.New("invalid LNA gain")
	}
	ex := byte(0x20) | (g & 0x07)
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

// returns signal strength in dBm ( -120dBm (low) -> -30dBm (high) )
func (l *LoRa) GetSignalStrength() (int, error) {
	rssiRaw, err := l.readReg(RegPktRssiValue)
	// datasheet specifies offset of 137 dBm
	return int(rssiRaw) - 137, err
}

func (l *LoRa) writeReg(reg, val byte) error {
	l.csPin.Low()
	defer l.csPin.High()
	return l.spiConn.Tx([]byte{reg | 0x80, val}, nil)
}

func (l *LoRa) readReg(reg byte) (byte, error) {
	buf := make([]byte, 2)
	l.csPin.Low()
	defer l.csPin.High()
	if err := l.spiConn.Tx([]byte{reg & 0x7F, 0x00}, buf); err != nil {
		return 0, err
	}
	return buf[1], nil
}
