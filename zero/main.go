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
	if err = radio.SetTxPower(9); err != nil {
		panic(err)
	}

	fmt.Println(radio.FormatConfig())

	for {
		data, err := radio.Receive(999, time.Second)
		if err != nil {
			fmt.Println(err.Error())
		}

		println(string(data))
	}
}
