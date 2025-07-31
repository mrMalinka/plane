package main

import (
	"machine"
	"pico/lora"
	"time"

	"tinygo.org/x/drivers/hd44780i2c"
)

/*
func formatErrorPacket(while string, err error) []byte {
	const header_error byte = 0
	return append(
		[]byte{header_error},
		[]byte(fmt.Sprintf("error while %s: %v", while, err.Error()))...,
	)
}
*/

func readUsbPacket() ([]byte, error) {
	buf := make([]byte, maxPacketSize)
	start := time.Now()
	n := 0

	for {
		if n >= len(buf) {
			break
		}
		if machine.USBCDC.Buffered() > 0 {
			b, err := machine.USBCDC.ReadByte()
			if err != nil {
				return buf[:n], err
			}
			buf[n] = b
			n++
			start = time.Now()
			continue
		}
		if time.Since(start) >= usbByteTimeout {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	return buf[:n], nil
}

const (
	mainFrequency  = 433.36e6
	maxPacketSize  = 1 << 8
	usbByteTimeout = 25 * time.Millisecond
)

var (
	onboardLed machine.Pin
	lcd        hd44780i2c.Device
	radio      *lora.LoRa
)

func init() {
	var err error

	// led
	onboardLed = machine.Pin(25)
	onboardLed.Configure(machine.PinConfig{
		Mode: machine.PinOutput,
	})

	// lcd
	machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
		SDA:       machine.Pin(0),
		SCL:       machine.Pin(1),
	})
	lcd = hd44780i2c.New(machine.I2C0, 0x27)
	lcd.Configure(hd44780i2c.Config{
		Width: 16, Height: 2, Font: hd44780i2c.FONT_5X8, CursorOn: true, CursorBlink: true,
	})

	lcd.ClearDisplay()
	lcd.Print([]byte("one"))
	// lora
	radio, err = lora.New(lora.LoRaConfig{
		SpiDev: *machine.SPI0, // lsp error but builds fine and breaks if i try to fix it
		Sdi:    4,
		Sdo:    3,
		Sck:    2,
		Cs:     5,
		Reset:  6,
		FreqHz: mainFrequency,
	})
	lcd.ClearDisplay()
	lcd.Print([]byte("two"))
	if err != nil {
		lcd.ClearDisplay()
		lcd.Print([]byte(err.Error()))
		machine.USBCDC.Write(formatErrorPacket("initializing lora", err))

		time.Sleep(time.Second)
		panic(err)
	}
	lcd.ClearDisplay()
	lcd.Print([]byte("three"))
	// set tx power to 9dBm
	if err = radio.SetTxPower(true, 0, 9); err != nil {
		lcd.ClearDisplay()
		lcd.Print([]byte(err.Error()))
		machine.USBCDC.Write(formatErrorPacket("setting tx power on init", err))

		time.Sleep(time.Second)
		panic(err)
	}
	lcd.ClearDisplay()
	lcd.Print([]byte("four"))
	time.Sleep(time.Second)
}

func main() {
	go usbReceiveLoop()
	go radioLoop()

	select {}
}

func usbReceiveLoop() {
	for {
		data, err := readUsbPacket()
		if len(data) == 0 {
			time.Sleep(3 * time.Second)
			continue
		}
		if err != nil {
			lcd.ClearDisplay()
			lcd.Print([]byte(err.Error()))
			machine.USBCDC.Write(formatErrorPacket("receiving usb", err))
			continue
		}
		lcd.ClearDisplay()
		lcd.Print([]byte("USB:"))
		lcd.SetCursor(0, 1)
		lcd.Print(data)

		/*
			err = radio.Transmit(data)
			if err != nil {
				lcd.ClearDisplay()
				lcd.Print([]byte(err.Error()))
				machine.USBCDC.Write(formatErrorPacket("forwarding data", err))
				continue
			}
		*/
	}
}

func radioLoop() {
	for {
		data, err := radio.Receive(maxPacketSize, 800)
		if err != nil {
			lcd.ClearDisplay()
			lcd.Print([]byte(err.Error()))
			// TODO: do something when timed out
			if err.Error() != "rx timeout" {
				machine.USBCDC.Write(formatErrorPacket("receiving lora", err))
			}
			continue
		}
		lcd.ClearDisplay()
		lcd.Print([]byte("forwarding"))
		lcd.SetCursor(0, 1)
		lcd.Print(data[13:29])
		// forward the data
		machine.USBCDC.Write(data)
	}
}
