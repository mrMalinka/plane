package gps

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/adrianmo/go-nmea"
	"github.com/tarm/serial"
)

type NEO6M struct {
	port   *serial.Port
	reader *bufio.Reader
}

func New(portName string, baud int, readTimeout time.Duration) (*NEO6M, error) {
	p, err := serial.OpenPort(&serial.Config{
		Name:        portName,
		Baud:        baud,
		ReadTimeout: readTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("opening serial port: %w", err)
	}
	return &NEO6M{
		port:   p,
		reader: bufio.NewReader(p),
	}, nil
}

func (n *NEO6M) Sentence() (nmea.Sentence, error) {
	line, err := n.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, fmt.Errorf("reading NMEA: %w", err)
	}
	s, err := nmea.Parse(line)
	if err != nil {
		return nil, fmt.Errorf("parsing NMEA: %w", err)
	}
	return s, nil
}

func (n *NEO6M) LatitudeLongitude() (float64, float64, error) {
	for {
		sentence, err := n.Sentence()
		if err != nil {
			return 0, 0, err
		}
		switch sentence := sentence.(type) {
		case nmea.GGA:
			if sentence.FixQuality == "0" {
				return 0, 0, errors.New("fix not available")
			}
			return sentence.Latitude, sentence.Longitude, nil
		}
	}
}

func (n *NEO6M) Close() error {
	return n.port.Close()
}
