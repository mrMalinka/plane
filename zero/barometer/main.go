package barometer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
)

type BMP390 struct {
	dev    i2c.Dev
	calib  calibrationData
	config Config
}

type Config struct {
	PressureOversampling    uint8
	TemperatureOversampling uint8
	IIRFilter               uint8
	OutputDataRate          uint8
}

// hardcoded corrections written to non volatile memory in the factory because not all chips are identical
type calibrationData struct {
	PAR_T1  float64
	PAR_T2  float64
	PAR_T3  float64
	PAR_P1  float64
	PAR_P2  float64
	PAR_P3  float64
	PAR_P4  float64
	PAR_P5  float64
	PAR_P6  float64
	PAR_P7  float64
	PAR_P8  float64
	PAR_P9  float64
	PAR_P10 float64
	PAR_P11 float64
	T_LIN   float64
}

type Measurement struct {
	Pressure    physic.Pressure
	Temperature physic.Temperature
}

func New(bus i2c.BusCloser, config Config) (*BMP390, error) {
	dev := i2c.Dev{Bus: bus, Addr: BMP390_Address}

	bmp := &BMP390{
		dev:    dev,
		config: config,
	}

	if err := bmp.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize BMP390: %w", err)
	}

	return bmp, nil
}

func (b *BMP390) init() error {
	chipIDBytes := []byte{0}
	if err := b.dev.Tx([]byte{RegChipId}, chipIDBytes); err != nil {
		return fmt.Errorf("failed to read chip ID: %w", err)
	}

	if chipIDBytes[0] != BMP390_ID {
		return fmt.Errorf("invalid chip ID: expected 0x%02X, got 0x%02X", BMP390_ID, chipIDBytes[0])
	}

	if err := b.softReset(); err != nil {
		return fmt.Errorf("failed to reset sensor: %w", err)
	}

	time.Sleep(10 * time.Millisecond)

	if err := b.readCalibrationData(); err != nil {
		return fmt.Errorf("failed to read calibration data: %w", err)
	}

	if err := b.configure(); err != nil {
		return fmt.Errorf("failed to configure sensor: %w", err)
	}

	return nil
}

func (b *BMP390) readCalibrationData() error {
	calibData := make([]byte, otpLength)
	if err := b.dev.Tx([]byte{otpStart}, calibData); err != nil {
		return fmt.Errorf("failed to read calibration data: %w", err)
	}

	nvm_par_t1 := binary.LittleEndian.Uint16(calibData[0:2])
	nvm_par_t2 := binary.LittleEndian.Uint16(calibData[2:4])
	nvm_par_t3 := int8(calibData[4])
	nvm_par_p1 := int16(binary.LittleEndian.Uint16(calibData[5:7]))
	nvm_par_p2 := int16(binary.LittleEndian.Uint16(calibData[7:9]))
	nvm_par_p3 := int8(calibData[9])
	nvm_par_p4 := int8(calibData[10])
	nvm_par_p5 := binary.LittleEndian.Uint16(calibData[11:13])
	nvm_par_p6 := binary.LittleEndian.Uint16(calibData[13:15])
	nvm_par_p7 := int8(calibData[15])
	nvm_par_p8 := int8(calibData[16])
	nvm_par_p9 := int16(binary.LittleEndian.Uint16(calibData[17:19]))
	nvm_par_p10 := int8(calibData[19])
	nvm_par_p11 := int8(calibData[20])

	b.calib.PAR_T1 = float64(nvm_par_t1) / math.Pow(2, -8)
	b.calib.PAR_T2 = float64(nvm_par_t2) / math.Pow(2, 30)
	b.calib.PAR_T3 = float64(nvm_par_t3) / math.Pow(2, 48)
	b.calib.PAR_P1 = (float64(nvm_par_p1) - math.Pow(2, 14)) / math.Pow(2, 20)
	b.calib.PAR_P2 = (float64(nvm_par_p2) - math.Pow(2, 14)) / math.Pow(2, 29)
	b.calib.PAR_P3 = float64(nvm_par_p3) / math.Pow(2, 32)
	b.calib.PAR_P4 = float64(nvm_par_p4) / math.Pow(2, 37)
	b.calib.PAR_P5 = float64(nvm_par_p5) / math.Pow(2, -3)
	b.calib.PAR_P6 = float64(nvm_par_p6) / math.Pow(2, 6)
	b.calib.PAR_P7 = float64(nvm_par_p7) / math.Pow(2, 8)
	b.calib.PAR_P8 = float64(nvm_par_p8) / math.Pow(2, 15)
	b.calib.PAR_P9 = float64(nvm_par_p9) / math.Pow(2, 48)
	b.calib.PAR_P10 = float64(nvm_par_p10) / math.Pow(2, 48)
	b.calib.PAR_P11 = float64(nvm_par_p11) / math.Pow(2, 65)

	return nil
}

func (b *BMP390) configure() error {
	osrValue := (b.config.PressureOversampling << 3) | b.config.TemperatureOversampling
	if err := b.writeReg(RegOsr, osrValue); err != nil {
		return fmt.Errorf("failed to configure oversampling: %w", err)
	}

	if err := b.writeReg(RegOdr, b.config.OutputDataRate); err != nil {
		return fmt.Errorf("failed to configure ODR: %w", err)
	}

	if err := b.writeReg(RegConfig, b.config.IIRFilter<<1); err != nil {
		return fmt.Errorf("failed to configure IIR filter: %w", err)
	}

	if err := b.writeReg(RegPwrCtrl, 0x33); err != nil {
		return fmt.Errorf("failed to enable measurements: %w", err)
	}

	return nil
}

func (b *BMP390) softReset() error {
	return b.writeReg(RegCmd, Cmd_softReset)
}

func (b *BMP390) ReadMeasurement() (*Measurement, error) {
	status := []byte{0}
	if err := b.dev.Tx([]byte{RegStatus}, status); err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	if status[0]&0x60 != 0x60 {
		return nil, errors.New("sensor data not ready")
	}

	data := make([]byte, 6)
	if err := b.dev.Tx([]byte{RegPressureData1}, data); err != nil {
		return nil, fmt.Errorf("failed to read sensor data: %w", err)
	}

	rawPressure := uint32(data[2])<<16 | uint32(data[1])<<8 | uint32(data[0])
	rawTemperature := uint32(data[5])<<16 | uint32(data[4])<<8 | uint32(data[3])

	temperature := b.compensateTemperature(float64(rawTemperature))

	pressure := b.compensatePressure(float64(rawPressure))

	return &Measurement{
		Pressure:    physic.Pressure(pressure * float64(physic.Pascal)),
		Temperature: physic.Temperature(temperature * float64(physic.Celsius)),
	}, nil
}

func (b *BMP390) compensateTemperature(rawTemp float64) float64 {
	pd1 := rawTemp - b.calib.PAR_T1
	pd2 := pd1 * b.calib.PAR_T2
	b.calib.T_LIN = pd2 + pd1*pd1*b.calib.PAR_T3
	return b.calib.T_LIN
}

func (b *BMP390) compensatePressure(rawPres float64) float64 {
	pd1 := b.calib.PAR_P6 * b.calib.T_LIN
	pd2 := b.calib.PAR_P7 * b.calib.T_LIN * b.calib.T_LIN
	pd3 := b.calib.PAR_P8 * b.calib.T_LIN * b.calib.T_LIN * b.calib.T_LIN
	po1 := b.calib.PAR_P5 + pd1 + pd2 + pd3

	pd1 = b.calib.PAR_P2 * b.calib.T_LIN
	pd2 = b.calib.PAR_P3 * b.calib.T_LIN * b.calib.T_LIN
	pd3 = b.calib.PAR_P4 * b.calib.T_LIN * b.calib.T_LIN * b.calib.T_LIN
	po2 := rawPres * (b.calib.PAR_P1 + pd1 + pd2 + pd3)

	pd1 = rawPres * rawPres
	pd2 = b.calib.PAR_P9 + b.calib.PAR_P10*b.calib.T_LIN
	pd3 = pd1 * pd2
	pd4 := pd3 + rawPres*rawPres*rawPres*b.calib.PAR_P11

	return po1 + po2 + pd4
}

func (b *BMP390) writeReg(reg, value uint8) error {
	return b.dev.Tx([]byte{reg, value}, nil)
}

func DefaultConfig() Config {
	return Config{
		PressureOversampling:    0x05, // 32x oversampling
		TemperatureOversampling: 0x05, // 32x oversampling
		IIRFilter:               0x03, // IIR filter coefficient 7
		OutputDataRate:          0x00, // 200 Hz
	}
}
