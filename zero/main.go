package main

import (
	"fmt"
	"os"
	"time"
	"zero/lora"
)

func main() {
	radio, err := lora.New("", "GPIO25", 433e6)
	if err != nil {
		fmt.Println("error while creating new lora:", err)
		os.Exit(1)
	}
	if err = radio.SetTxPower(true, 0, 8); err != nil {
		panic(err)
	}

	fmt.Println(radio.FormatConfig())

	for {
		data, err := radio.Receive(999, time.Second)
		if err != nil {
			if err.Error() == "rx timeout" {
				continue
			} else {
				fmt.Println(err.Error())
			}
		}

		println(radio.GetSignalStrength())
		println(string(data))
	}
}
