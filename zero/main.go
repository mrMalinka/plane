package main

import (
	"fmt"
	"os"
	"time"
	"zero/lora"
)

func main() {
	radio, err := lora.New("", "GPIO25", "GPIO24", 433e6)
	if err != nil {
		fmt.Println("error while creating new lora:", err)
		os.Exit(1)
	}
	radio.Init()
	radio.SetTxPower(9)

	err = radio.Transmit([]byte("hello world"), 10*time.Millisecond)
	if err != nil {
		panic(err)
	}
}
