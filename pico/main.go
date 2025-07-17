package main

import (
	"machine"
	"pico/lora"
	"time"

	"tinygo.org/x/drivers/hd44780i2c"
)

func main() {
	onboardLed := machine.Pin(25)
	onboardLed.Configure(machine.PinConfig{
		Mode: machine.PinOutput,
	})
	onboardLed.High()

	machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
		SDA:       machine.Pin(0),
		SCL:       machine.Pin(1),
	})

	lcd := hd44780i2c.New(machine.I2C0, 0x27)
	lcd.Configure(hd44780i2c.Config{
		Width: 16, Height: 2, Font: hd44780i2c.FONT_5X8, CursorOn: true, CursorBlink: true,
	})

	onboardLed.Low()

	radio, err := lora.New(lora.LoRaConfig{
		SpiDev: *machine.SPI0,
		Sdi:    4,
		Sdo:    3,
		Sck:    2,
		Cs:     5,
		Reset:  6,
		FreqHz: 433e6,
	})
	if err != nil {
		lcd.Print([]byte(err.Error()))
		return
	}

	err = radio.Transmit([]byte("hello world"), 2*time.Second)
	if err != nil {
		lcd.Print([]byte(err.Error()))
		return
	}
	lcd.ClearDisplay()
	lcd.Print([]byte("finished"))
}
