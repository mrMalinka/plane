package main

import (
	"fmt"
	"os"
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

	data, err := radio.Receive(999)
	if err != nil {
		panic(err)
	}

	println(string(data))
}
