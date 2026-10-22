package domain

import (
	"errors"
	"math"
	"time"
)

type Schedule struct {
	Initial     time.Duration
	Factor      float64
	Cap         time.Duration
	MaxAttempts int
}

func (s Schedule) Validate() error {
	switch {
	case s.Initial <= 0:
		return errors.New("initial delay must be greater than zero")
	case s.Factor < 1:
		return errors.New("backoff factor must be at least 1")
	case s.Cap < s.Initial:
		return errors.New("maximum delay must not be below the initial delay")
	case s.MaxAttempts < 1:
		return errors.New("maximum attempts must be at least 1")
	}
	return nil
}

func (s Schedule) Exhausted(attempts int) bool {
	return attempts >= s.MaxAttempts
}

// Delay is the wait after the given number of failed attempts (1-based); jitter in [0,1) selects a point in [0, ceiling) (full jitter).
func (s Schedule) Delay(failedAttempts int, jitter float64) time.Duration {
	if failedAttempts < 1 {
		failedAttempts = 1
	}
	ceiling := float64(s.Initial) * math.Pow(s.Factor, float64(failedAttempts-1))
	if ceiling > float64(s.Cap) || math.IsInf(ceiling, 0) {
		ceiling = float64(s.Cap)
	}
	return time.Duration(ceiling * jitter)
}
