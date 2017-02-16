package bot

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// Auth checks the current user's ID to match the admin's ID
func Auth(admin int) HandleFunc {
	return func(c Context) error {
		if c.User().ID != admin {
			return fmt.Errorf("unauthorized")
		}
		return nil
	}
}

// Instrument the handlers and create prometheus metrics for those
func Instrument(counter *prometheus.CounterVec) HandleFunc {
	return func(c Context) error {
		counter.WithLabelValues(c.Raw()).Inc()
		return nil
	}
}
