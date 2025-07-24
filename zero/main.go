package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
	"zero/lora"
)

const (
	mainFrequency = 433.36e6
	maxPacketSize = 1 << 10
)

var (
	radio *lora.LoRa

	status planeStatus
)

func init() {
	file, err := os.OpenFile("blackbox.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalln("failed to open log file:", err)
	}
	log.SetOutput(file)

	radio, err = lora.New("", "GPIO25", mainFrequency)
	if err != nil {
		log.Fatalln("error while creating new lora:", err)
	}
	if err = radio.SetTxPower(true, 0, 9); err != nil {
		log.Fatalln("error while setting tx power on init:", err)
	}

	status = planeStatus{
		status:   status_none,
		battery:  0,
		speed:    0,
		altitude: 0,
	}
}

func main() {
	log.Println(radio.FormatConfig())

	go sensorLoop()
	go radioLoop()

	select {}
}

func sensorLoop() {
	const updateInterval = 129 * time.Millisecond

	for {
		if status.battery > 50 {
			status.battery = 0
		} else {
			status.battery++
		}

		status.status = byte(rand.Intn(7))

		if status.speed > 10 {
			status.speed = 0
		} else {
			status.speed++
		}

		if status.altitude > 100 {
			status.altitude = 0
		} else {
			status.altitude += 8
		}

		time.Sleep(updateInterval)
	}
}

func radioLoop() {
	const updateInterval = 200 * time.Millisecond

	for {
		bytes := status.toBytes()
		radio.Transmit(newPacket(header_bulk, bytes[:]))
		fmt.Printf("transmit: %b\n", bytes)
		time.Sleep(updateInterval)
	}
}
