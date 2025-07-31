package main

import (
	"fmt"
	"log"
	"os"
	"time"
	gpslib "zero/gps"
	"zero/gyroscope"
	"zero/lora"

	"periph.io/x/host/v3"
)

const (
	mainFrequency = 433.36e6
	maxPacketSize = 1 << 8
)

var (
	radio *lora.LoRa
	gyro  *gyroscope.BNO055
	gps   *gpslib.NEO6M

	status       planeStatus
	radioAirtime time.Duration
)

func init() {
	if _, err := host.Init(); err != nil {
		log.Fatalln("error initializing host:", err)
	}

	// log
	file, err := os.OpenFile("blackbox.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalln("failed to open log file:", err)
	}
	log.SetOutput(file)

	// radio
	radio, err = lora.New("", "GPIO25", mainFrequency)
	if err != nil {
		log.Fatalln("error while creating new lora:", err)
	}
	if err = radio.SetTxPower(true, 0, 9); err != nil {
		log.Fatalln("error while setting tx power on init:", err)
	}

	// gyro
	gyro, err = gyroscope.New(
		"1",

		gyroscope.OrientAndroid|
			gyroscope.EulerDeg|
			gyroscope.TempC|
			gyroscope.GyrDPS|
			gyroscope.AccMS2,
	)
	if err != nil {
		log.Fatalln("error while creating new gyro:", err)
	}

	// gps
	gps, err = gpslib.New("/dev/serial0", 9600, 2*time.Second)
	if err != nil {
		log.Fatalln("error while creating new gps:", err)
	}

	status = planeStatus{
		status:    status_none,
		battery:   0,
		speed:     0,
		altitude:  0,
		latitude:  0,
		longitude: 0,
	}
}

func main() {
	log.Println(radio.FormatConfig())

	// give chips time to warm up
	time.Sleep(2 * time.Second)

	go sensorLoop()
	go radioLoop()

	select {}
}

func sensorLoop() {
	const updateInterval = 20 * time.Second

	for {
		println("-----")
		temp, err := gyro.ReadTemperature()
		if err != nil {
			println(err.Error())
			continue
		} else {
			fmt.Printf("Temperature: %v\n", temp)
			println()
		}

		heading, roll, pitch, err := gyro.ReadEuler()
		if err != nil {
			println(err.Error())
			continue
		} else {
			fmt.Printf("Heading: %v\n", heading)
			fmt.Printf("Roll: %v\n", roll)
			fmt.Printf("Pitch: %v\n", pitch)
			println()
		}

		ax, ay, az, err := gyro.ReadLinearAccel()
		if err != nil {
			println(err.Error())
			continue
		} else {
			fmt.Printf("AX: %v\n", ax)
			fmt.Printf("AY: %v\n", ay)
			fmt.Printf("AZ: %v\n", az)
			println()
		}

		latitude, longitude, err := gps.LatitudeLongitude()
		if err != nil {
			println(err.Error())
			continue
		} else {
			fmt.Printf("LAT: %v\n", latitude)
			fmt.Printf("LONG: %v\n", longitude)
			println()
		}

		status.latitude = latitude
		status.longitude = longitude
		status.status = status_flying

		time.Sleep(updateInterval)
	}
}

func radioLoop() {
	const updateInterval = 1200 * time.Millisecond

	for {
		bytes := status.toBytes()

		start := time.Now()
		radio.Transmit(newPacket(payloadType_bulk, bytes[:]))
		radioAirtime += time.Since(start)

		fmt.Printf("transmit: %b\nairtime: %dms\n", bytes, radioAirtime.Milliseconds())
		time.Sleep(updateInterval)
	}
}
