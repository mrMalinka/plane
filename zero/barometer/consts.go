package barometer

// random unrelated constants
const (
	BMP390_Address = 0x76
	BMP390_ID      = 0x3C

	otpStart  = 0x31
	otpLength = 21
)

const (
	Cmd_fifoFlush = 0xB0
	Cmd_softReset = 0xB6
)

// registers
const (
	RegChipId = 0x00
	RegRevId  = 0x01
	RegErrReg = 0x02
	RegStatus = 0x03

	RegPressureData1 = 0x04
	RegPressureData2 = 0x05
	RegPressureData3 = 0x06

	RegTemperatureData1 = 0x07
	RegTemperatureData2 = 0x08
	RegTemperatureData3 = 0x09

	RegSensorTime1 = 0x0C
	RegSensorTime2 = 0x0D
	RegSensorTime3 = 0x0E

	RegEvent     = 0x10
	RegIntStatus = 0x11

	RegFifoLength1 = 0x12
	RegFifoLength2 = 0x13

	RegFifoData = 0x14

	RegFifoWatermark1 = 0x15
	RegFifoWatermark2 = 0x16

	RegFifoConfig1 = 0x17
	RegFifoConfig2 = 0x18

	RegIntCtrl = 0x19
	RegIfConf  = 0x1A
	RegPwrCtrl = 0x1B
	RegOsr     = 0x1C
	RegOdr     = 0x1D
	RegConfig  = 0x1F
	RegCmd     = 0x7E
)
