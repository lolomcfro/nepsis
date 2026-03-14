package adb

import (
	"strings"
	"time"
)

// ParseDevices parses the output of `adb devices` and returns authorized device serials.
func ParseDevices(output string) []string {
	var devices []string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "device" {
			devices = append(devices, fields[0])
		}
	}
	return devices
}

// Poller polls for connected ADB devices on a fixed interval.
type Poller struct {
	runner   Executor
	interval time.Duration
	onChange func(connected bool, serial string)
	stop     chan struct{}
}

// NewPoller creates a Poller that calls onChange when connection state changes.
func NewPoller(r Executor, interval time.Duration, onChange func(bool, string)) *Poller {
	return &Poller{
		runner:   r,
		interval: interval,
		onChange: onChange,
		stop:     make(chan struct{}),
	}
}

// Start begins polling in a background goroutine.
func (p *Poller) Start() {
	go func() {
		var lastSerial string
		for {
			select {
			case <-p.stop:
				return
			case <-time.After(p.interval):
				out, err := p.runner.Run("devices")
				if err != nil {
					if lastSerial != "" {
						lastSerial = ""
						p.onChange(false, "")
					}
					continue
				}
				devices := ParseDevices(out)
				if len(devices) > 0 && devices[0] != lastSerial {
					lastSerial = devices[0]
					p.onChange(true, lastSerial)
				} else if len(devices) == 0 && lastSerial != "" {
					lastSerial = ""
					p.onChange(false, "")
				}
			}
		}
	}()
}

// Stop halts the polling goroutine.
func (p *Poller) Stop() {
	close(p.stop)
}
