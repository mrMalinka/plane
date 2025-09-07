package pid

import "time"

type PID struct {
	Kp, Ki, Kd float32

	Setpoint float32

	prevError     float32
	integral      float32
	lastTimestamp time.Time
}

func NewPID(kp, ki, kd, setpoint float32) *PID {
	return &PID{
		Kp:            kp,
		Ki:            ki,
		Kd:            kd,
		Setpoint:      setpoint,
		lastTimestamp: time.Now(),
	}
}

func (p *PID) Compute(measurement float32, now time.Time) float32 {
	dt := float32(now.Sub(p.lastTimestamp).Seconds())
	if dt <= 0 {
		dt = 1e-3
	}

	err := p.Setpoint - measurement

	P := p.Kp * err

	p.integral += err * dt
	I := p.Ki * p.integral

	derivative := (err - p.prevError) / dt
	D := p.Kd * derivative

	p.prevError = err
	p.lastTimestamp = now

	return P + I + D
}
