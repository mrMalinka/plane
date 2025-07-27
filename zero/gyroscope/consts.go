package gyroscope

const (
	BNO055_Address = 0x29
	BNO055_Id      = 0xA0
)

const (
	regChipID     = 0x00
	regPageID     = 0x07
	regOprMode    = 0x3D
	regPwrMode    = 0x3E
	regSysTrigger = 0x3F
	regUnitSel    = 0x3B
	regTemp       = 0x34

	regEulerStart  = 0x1A
	regEulerLength = 6

	regLinearAccelStart  = 0x28
	regLinearAccelLength = 6
)

const (
	modeConfig = 0x00
	modeNDOF   = 0x0C
)

const (
	pwrNormal = 0x00
)

const (
	OrientAndroid byte = 0 << 7
	OrientWindows byte = 1 << 7

	TempC byte = 0 << 4
	TempF byte = 1 << 4

	EulerDeg byte = 0 << 2
	EulerRad byte = 1 << 2

	GyrDPS byte = 0 << 1
	GyrRPS byte = 1 << 1

	AccMS2 byte = 0 << 0
	AccMG  byte = 1 << 0
)
