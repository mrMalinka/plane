package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/hd44780i2c"
)

func main() {
	onboardLed := machine.Pin(25)
	onboardLed.Configure(machine.PinConfig{
		Mode: machine.PinOutput,
	})

	machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
		SDA:       machine.Pin(1),
		SCL:       machine.Pin(2),
	})

	lcd := hd44780i2c.New(machine.I2C0, 0x27)
	lcd.ClearDisplay()
	lcd.Print([]byte("hello world"))

	onboardLed.High()
	time.Sleep(3 * time.Second)
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
