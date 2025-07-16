package main

import (
	"machine"

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
	lcd.ClearDisplay()
	lcd.Print([]byte("hello world"))

	onboardLed.Low()

	/*
		radio, err := lora.New(
			machine.SPI0, 3, 4, 2, 5, 7, 6, 433e6,
		)
		radio.Init()
		if err != nil {
			return
		}
	*/
}
