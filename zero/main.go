package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"time"
	gpslib "zero/gps"
	"zero/gyroscope"
	"zero/lora"
	"zero/pid"

	"periph.io/x/host/v3"
)

const (
	logFilename = "blackbox.log"

	mainFrequency = 433.36e6
	maxPacketSize = 1 << 8

	// how many degress to tilt pitch per one meter of altitude error
	pitchAdjustmentMultiplier = 10
	// how many degress to tilt roll per one degree of roll error
	rollAdjustmentMultiplier = 0.5
	// degrees
	pitchTargetMax = 15
	rollTargetMax  = 10

	flightUpdateInterval = 200 * time.Microsecond // 0.2ms
	radioUpdateInterval  = 366                    // symbols, ~12s with current settings
)

var (
	radio *lora.LoRa
	gyro  *gyroscope.BNO055
	gps   *gpslib.NEO6M

	radioAirtime time.Duration

	status   planeStatus
	statusMu sync.Mutex

	pitchPid, rollPid *pid.PID

	targetMu      sync.Mutex
	wpLat, wpLong float64
	targetAlt     float32

	mode rune = 'i'
)

func init() {
	if _, err := host.Init(); err != nil {
		log.Fatalln("error initializing host:", err)
	}

	// log
	file, err := os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
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

	// pid
	rollPid = pid.NewPID(0.03, 0, 0.01, 0)
	pitchPid = pid.NewPID(0.03, 0, 0.01, 0)

	status = planeStatus{
		status:    status_none,
		battery:   0,
		speed:     0,
		altitude:  0,
		latitude:  0,
		longitude: 0,
	}

	diagnostic()
}

func diagnostic() {
	log.Println("Starting system diagnostic")

	// double check
	if radio == nil {
		log.Fatalln("diagnostic: radio not initialized")
	}
	if gyro == nil {
		log.Fatalln("diagnostic: gyro not initialized")
	}
	if gps == nil {
		log.Fatalln("diagnostic: gps not initialized")
	}

	// print radio config
	radioConfig, err := radio.FormatConfig()
	if err != nil {
		log.Fatalln("diagnostic: could not format radio config:", err)
	}
	log.Println("LoRa:", radioConfig)

	// warm up
	time.Sleep(time.Second)
	// gyro
	_, _, _, err = gyro.ReadEuler()
	if err != nil {
		log.Fatalln("diagnostic: error reading euler:", err)
	}
	time.Sleep(10 * time.Millisecond)
	_, _, _, err = gyro.ReadLinearAccel()
	if err != nil {
		log.Fatalln("diagnostic: error reading accel:", err)
	}
	time.Sleep(10 * time.Millisecond)
	_, err = gyro.ReadTemperature()
	if err != nil {
		log.Fatalln("diagnostic: error reading temp:", err)
	}
	// gps
	_, _, _, err = gps.LatLongAlt()
	if err != nil {
		log.Fatalln("diagnostic: error reading lat/long:", err)
	}

	// verify lora is actually transmitting
	start := time.Now()
	err = radio.Transmit([]byte{0xFF})
	if err != nil {
		log.Fatalln("diagnostic: could not transmit:", err)
	}
	if time.Since(start) < time.Microsecond {
		log.Fatalln("diagnostic: test transmission took less than a microsecond (is the chip connected?)")
	}

	// test gyro
	temp, err := gyro.ReadTemperature()
	if err != nil {
		log.Fatalln("diagnostic: could not read temperature:", err)
	}
	if temp == 0 {
		log.Fatalln("diagnostic: incorrect temperature reading of 0 (unless it's winter)")
	}

	log.Println("Diagnostic successful")
}

func main() {
	go flightLoop()
	go radioLoop()

	select {}
}

func flightLoop() {
	for {
		switch mode {
		case 'i':
			time.Sleep(100 * time.Millisecond)
		case 'l':
			// TODO
		case 't':
			setThrust(100) // max
			x, y, z, err := gyro.ReadLinearAccel()
			if err != nil {
				setThrust(0)
				log.Fatalln("flightLoop: failed to read accel while taking off")
			}
			if (x+y+z)/3 > 8 { // 8m/s
				mode = 'f'
			}

		case 'f':
			yaw, roll, pitch, err := gyro.ReadEuler()
			if err != nil {
				log.Printf("flightLoop: gyro read error: %v", err)
				continue
			}

			lat, long, alt, err := gps.LatLongAlt()
			if err != nil {
				log.Printf("flightLoop: gps read error: %v", err)
			}

			statusMu.Lock()
			status.latitude = lat
			status.longitude = long
			status.altitude = float32(alt)
			statusMu.Unlock()

			targetMu.Lock()

			// update pitch target
			altError := float32(alt) - targetAlt
			pitchTarget := altError * pitchAdjustmentMultiplier
			pitchTarget = max(-pitchTargetMax, min(pitchTarget, pitchTargetMax))
			pitchPid.Setpoint = pitchTarget

			// update roll target
			yawError := bearingError(float64(yaw), lat, long, wpLat, wpLong)
			rollTarget := yawError * rollAdjustmentMultiplier
			rollTarget = max(-rollTargetMax, min(rollTarget, rollTargetMax))
			if rollTarget > -0.75 && rollTarget < 0.75 {
				rollTarget = 0
			}
			rollPid.Setpoint = pitchTarget

			targetMu.Unlock()

			rollControl := rollPid.Compute(roll, time.Now())
			pitchControl := pitchPid.Compute(pitch, time.Now())

			// actuate
			fmt.Println("roll:", rollControl)
			fmt.Println("pitch:", pitchControl)
		}
	}
}

func radioLoop() {
	for {
		if stat, err := os.Stat(logFilename); err == nil {
			// redirect log if it gets too large (1gb)
			if stat.Size() > 1<<30 {
				log.SetOutput(os.Stdout)
			}
		}

		statusMu.Lock()
		bytes := status.toBytes()
		statusMu.Unlock()

		start := time.Now()
		radio.Transmit(newPacket(payloadType_bulk, bytes[:]))
		radioAirtime += time.Since(start)

		fmt.Printf("transmit: %b\nairtime: %dms\n", bytes, radioAirtime.Milliseconds())

		data, err := radio.Receive(maxPacketSize, radioUpdateInterval)
		if err != nil {
			if err.Error() != "rx timeout" {
				log.Println("radioLoopL: rx error:", err)
			}
			continue
		}
		if len(data) < 2 {
			log.Printf("data length of %v with no error (<2)\n", len(data))
			continue
		}
		if len(data) >= 2 && int(data[1]) != len(data) {
			log.Println("data incorrectly stated its length")
			continue
		}

		switch data[0] {
		case payloadType_wpSet:
			wpLatNew := math.Float64frombits(binary.BigEndian.Uint64(data[2:10]))
			wpLongNew := math.Float64frombits(binary.BigEndian.Uint64(data[10:18]))

			// probably a good practice unless youre flying directly over null island
			if wpLatNew == 0 || wpLongNew == 0 {
				log.Printf("new lat/long was 0: %v/%v", wpLatNew, wpLongNew)
				continue
			}

			targetMu.Lock()
			wpLat, wpLong = wpLatNew, wpLongNew
			targetMu.Unlock()
		case payloadType_altSet:
			altNew := math.Float32frombits(binary.BigEndian.Uint32(data[2:6]))
			targetMu.Lock()
			targetAlt = altNew
			targetMu.Unlock()
		}
	}
}

func bearingError(yaw, lat, long, latWP, lonWP float64) float64 {
	const R = 6371000.0

	toRad := math.Pi / 180.0
	latRad := lat * toRad
	lonRad := long * toRad
	latWPRad := latWP * toRad
	lonWPRad := lonWP * toRad

	dLat := latWPRad - latRad
	dLon := lonWPRad - lonRad
	dX := dLon * math.Cos(latRad) * R
	dY := dLat * R

	yawRad := yaw * toRad
	Ax := math.Sin(yawRad)
	Ay := math.Cos(yawRad)

	magB := math.Hypot(dX, dY)
	if magB == 0 {
		return 0
	}

	dot := Ax*dX + Ay*dY
	cosTheta := dot / magB

	if cosTheta > 1 {
		cosTheta = 1
	} else if cosTheta < -1 {
		cosTheta = -1
	}

	theta := math.Acos(cosTheta) * 180.0 / math.Pi
	return theta
}

func setThrust(power int) {
	fmt.Printf("Thrust set to %d\n", power)
}
