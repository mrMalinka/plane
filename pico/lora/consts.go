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
	RegIrqFlags         = 0x12
	RegRxNbBytes        = 0x13
	RegPktRssiValue     = 0x1A
	RegRssiValue        = 0x1B
	RegHopChannel       = 0x1C
	RegModemConfig1     = 0x1D
	RegModemConfig2     = 0x1E
	RegSymbTimeoutLsb   = 0x20
	RegPreambleMsb      = 0x20
	RegPreambleLsb      = 0x21
	RegPayloadLength    = 0x22
	RegMaxPayloadLength = 0x23
	RegSyncWord         = 0x39
	RegDioMapping1      = 0x40
	RegVersion          = 0x42
)

// mode constants
const (
	ModeSleep        = 0x00
	ModeStandby      = 0x01
	ModeFSTx         = 0x02
	ModeTx           = 0x03
	ModeFSRx         = 0x04
	ModeRxContinuous = 0x05
	ModeRxSingle     = 0x06
)

// dio0 mapping options
const (
	DIO0_RxDone          = 0x00
	DIO0_TxDone          = 0x40
	DIO0_PayloadCrcError = 0x80
	DIO0_ValidHeader     = 0xC0
)

// IRQ masks
const (
	IrqTxDone         = 0x08
	IrqRxDone         = 0x40
	IrqPayloadCrcErr  = 0x20
	IrqValidHeader    = 0x10
	IrqRxTimeout      = 0x80
	IrqFhssChangeChan = 0x02
	IrqCadDone        = 0x04
	IrqCadDetected    = 0x01
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
