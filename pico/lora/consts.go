package lora

// register addresses
const (
	RegFifo             = 0x00
	RegOpMode           = 0x01
	RegFrMsb            = 0x06
	RegFrMid            = 0x07
	RegFrLsb            = 0x08
	RegPaConfig         = 0x09
	RegOcp              = 0x0B
	RegLna              = 0x0C
	RegFifoAddrPtr      = 0x0D
	RegFifoTxBaseAddr   = 0x0E
	RegFifoRxBaseAddr   = 0x0F
	RegFifoRxCurrent    = 0x10
	RegIrqFlagsMask     = 0x11
	RegIrqFlags         = 0x12
	RegRxNbBytes        = 0x13
	RegPktRssiValue     = 0x1A
	RegRssiValue        = 0x1B
	RegHopChannel       = 0x1C
	RegModemConfig1     = 0x1D
	RegModemConfig2     = 0x1E
	RegSymbTimeoutLsb   = 0x1F
	RegPreambleMsb      = 0x20
	RegPreambleLsb      = 0x21
	RegPayloadLength    = 0x22
	RegMaxPayloadLength = 0x23
	RegSyncWord         = 0x39
	RegDioMapping1      = 0x40
	RegVersion          = 0x42
)

// mode constants
// all of them have LoRa mode on because the other mode is unused
const (
	ModeSleep        = 0b10000000
	ModeStandby      = 0b10000001
	ModeFSTx         = 0b10000010
	ModeTx           = 0b10000011
	ModeFSRx         = 0b10000100
	ModeRxContinuous = 0b10000101
	ModeRxSingle     = 0b10000110
	ModeCad          = 0b10000111
)

// dio0 mapping options
const (
	DIO0_RxDone  = 0b00
	DIO0_TxDone  = 0b01
	DIO0_CadDone = 0b10
)

// IRQ masks
const (
	IrqRxTimeout       = 0b10000000
	IrqRxDone          = 0b01000000
	IrqPayloadCrcError = 0b00100000
	IrqValidHeader     = 0b00010000
	IrqTxDone          = 0b00001000
	IrqCadDone         = 0b00000100
	IrqFhssChangeChan  = 0b00000010
	IrqCadDetected     = 0b00000001
)

// LNA gain settings
const (
	LNA_G1 = 0x20
	LNA_G2 = 0x40
	LNA_G3 = 0x60
	LNA_G4 = 0x80
	LNA_G5 = 0xA0

	LNA_Boost1 = 0x01
	LNA_Boost2 = 0x02
	LNA_Boost3 = 0x03
)

// bandwidth settings (Hz)
var BwValues = map[uint32]byte{
	125000: 0x70,
	250000: 0x80,
	500000: 0x90,
}

// coding rate options (4/5..4/8)
var CrValues = map[string]byte{
	"4/5": 0x02,
	"4/6": 0x04,
	"4/7": 0x06,
	"4/8": 0x08,
}
