package gyroscope

import (
	"encoding/binary"
	"errors"
	"time"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
)

type BNO055 struct {
	dev i2c.Dev
}

func New(busName string, unitSel byte) (*BNO055, error) {
	bus, err := i2creg.Open(busName)
	if err != nil {
		return nil, err
	}
	dev := i2c.Dev{
		Bus:  bus,
		Addr: BNO055_Address,
	}

	bno055 := &BNO055{
		dev: dev,
	}
	bno055.Init(unitSel)

	return bno055, nil
}

func (b *BNO055) Init(unitSel byte) error {
	id, err := b.readReg(regChipID)
	if err != nil {
		return err
	}
	if id != BNO055_Id {
		return errors.New("Chip ID not found or unsupported")
	}

	// reset chip
	if err := b.writeReg(regSysTrigger, 1<<5); err != nil {
		return err
	}
	time.Sleep(650 * time.Millisecond)

	// config mode
	if err := b.writeReg(regOprMode, modeConfig); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)

	// unit outputs
	if err := b.writeReg(regUnitSel, unitSel); err != nil {
		return err
	}

	// power mode to normal
	if err := b.writeReg(regPwrMode, pwrNormal); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)

	// config mode again
	if err := b.writeReg(regOprMode, modeConfig); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)

	// switch to ndof fused mode
	if err := b.writeReg(regOprMode, modeNDOF); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)

	return nil
}

func (b *BNO055) ReadEuler() (heading, roll, pitch float32, err error) {
	buf := make([]byte, regEulerLength)
	if err = b.dev.Tx([]byte{regEulerStart}, buf); err != nil {
		return
	}
	// each is a signed 16 bit lsb/msb
	// scale factor = 16 lsb per degree
	heading = float32(int16(binary.LittleEndian.Uint16(buf[0:2]))) / 16.0
	roll = float32(int16(binary.LittleEndian.Uint16(buf[2:4]))) / 16.0
	pitch = float32(int16(binary.LittleEndian.Uint16(buf[4:6]))) / 16.0
	return
}

func (b *BNO055) ReadTemperature() (int8, error) {
	val, err := b.readReg(regTemp)
	return int8(val), err
}

func (b *BNO055) ReadLinearAccel() (x, y, z float32, err error) {
	buf := make([]byte, regLinearAccelLength)
	if err = b.dev.Tx([]byte{regLinearAccelStart}, buf); err != nil {
		return
	}
	// each is a signed 16 bit lsb/msb
	// scale factor = 100 lsb per m/s^2 (or 1 lsb per mg)
	x = float32(int16(binary.LittleEndian.Uint16(buf[0:2]))) / 100.0
	y = float32(int16(binary.LittleEndian.Uint16(buf[2:4]))) / 100.0
	z = float32(int16(binary.LittleEndian.Uint16(buf[4:6]))) / 100.0
	return
}

func (b *BNO055) writeReg(reg byte, value byte) error {
	return b.dev.Tx([]byte{reg, value}, nil)
}

func (b *BNO055) readReg(reg byte) (byte, error) {
	buf := make([]byte, 1)
	if err := b.dev.Tx([]byte{reg}, buf); err != nil {
		return 0x00, err
	}
	return buf[0], nil
}
