// Package durafmt formats time.Duration into a human readable format.
package durafmt

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	units = []string{"years", "months", "weeks", "days", "hours", "minutes", "seconds"}
)

// Durafmt holds the parsed duration and the original input duration.
type Durafmt struct {
	duration time.Duration
	input    string // Used as reference.
}

// Parse creates a new *Durafmt struct, returns error if input is invalid.
func Parse(dinput time.Duration) *Durafmt {
	input := dinput.String()
	return &Durafmt{dinput, input}
}

// ParseString creates a new *Durafmt struct from a string, returns error if input is invalid.
func ParseString(input string) (*Durafmt, error) {
	if input == "0" || input == "-0" {
		return nil, errors.New("durafmt: missing unit in duration " + input)
	}
	duration, err := time.ParseDuration(input)
	if err != nil {
		return nil, err
	}
	return &Durafmt{duration, input}, nil
}

// String parses d *Durafmt into a human readable duration.
func (d *Durafmt) String() string {
	var duration string

	// Check for minus durations.
	if string(d.input[0]) == "-" {
		duration += "-"
		d.duration = -d.duration
	}

	// Convert duration.
	seconds := int(d.duration.Seconds()) % 60
	minutes := int(d.duration.Minutes()) % 60
	hours := int(d.duration.Hours())
	days := (hours / 24)
	weeks := (days / 7)
	months := (weeks / 4)
	years := (months / 12)

	// Create a map of the converted duration time.
	convMap := map[string]int{
		"seconds": seconds,
		"minutes": minutes,
		"hours":   hours % 24,
		"days":    days % 7,
		"weeks":   weeks % 4,
		"months":  months % 12,
		"years":   years,
	}

	// Construct duration string.
	for _, k := range units {
		v := convMap[k]
		strval := strconv.Itoa(v)
		switch {
		// add to the duration string if v > 1.
		case v > 1:
			duration += strval + " " + k + " "
		// remove the plural 's', if v is 1.
		case v == 1:
			duration += strval + " " + strings.TrimRight(k, "s") + " "
		// omit any value with 0s or 0.
		case d.duration.String() == "0" || d.duration.String() == "0s":
			// disallow months.
			if k == "months" {
				continue
			}
			// check for suffix in input string and add the key.
			if strings.HasSuffix(d.input, string(k[0])) {
				duration += strval + " " + k
			}
			break
		// omit any value with 0.
		case v == 0:
			continue
		}
	}
	// trim any remaining spaces.
	duration = strings.TrimSpace(duration)
	return duration
}
