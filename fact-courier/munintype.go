package main

import "time"

const (
	defaultType = "GAUGE"
)

// MuninType is implemented by RRDTool DST (Data Source Type) structures that
// are responsible for converting raw values into real values based on previous
// values and the duration since the last value
type MuninType interface {
	Name() string
	RequiresPrevious() bool
	Calculate(float64, float64, float64) float64
}

// MuninTypeGauge calculates the real value for the GAUGE DST
type MuninTypeGauge struct {
}

// Name returns "GAUGE"
func (t *MuninTypeGauge) Name() string {
	return "GAUGE"
}

// RequiresPrevious returns false for GAUGE as it does not require a previous
// value
func (t *MuninTypeGauge) RequiresPrevious() bool {
	return false
}

// Calculate returns the value as-is for GAUGE
func (t *MuninTypeGauge) Calculate(value float64, previous float64, duration float64) float64 {
	return value
}

// MuninTypeCounter calculates the real value for the COUNTER DST
type MuninTypeCounter struct {
}

// Name returns "COUNTER"
func (t *MuninTypeCounter) Name() string {
	return "COUNTER"
}

// RequiresPrevious returns true for COUNTER as it requires a previous value
func (t *MuninTypeCounter) RequiresPrevious() bool {
	return true
}

// Calculate for COUNTER returns the difference between the previous value and
// the current value, and returns the per-second rate change
// It assumes it will never decrease unless a counter overflow occurs, at which
// point it attempts to work out whether it happened at a 32-bit boundary or a
// 64-bit boundary and return the correct rate of change
func (t *MuninTypeCounter) Calculate(value float64, previous float64, duration float64) float64 {
	if previous <= value {
		return (value - previous) / (duration / float64(time.Second))
	}

	// A wrap occurred and we're supported to detect whether it was at 32-bit or
	// 64-bit boundary - a quick search didn't bring up the exact RRDTool
	// calculations and I've yet to inspect the RRDTool source, so I'm going to
	// guess here

	// Is the previous value less than 2^31? Then treat it as a 32-bit wrap
	if previous <= 2^31 {
		return (2 ^ 31 - previous + value) / (duration / float64(time.Second))
	}

	return (2 ^ 63 - previous + value) / (duration / float64(time.Second))
}

// MuninTypeDerive calculates the real value for the DERIVE DST
type MuninTypeDerive struct {
}

// Name returns the DST name
func (t *MuninTypeDerive) Name() string {
	return "DERIVE"
}

// RequiresPrevious returns true for MuninTypeDerive as it requires a previous
// value
func (t *MuninTypeDerive) RequiresPrevious() bool {
	return true
}

// Calculate for DERIVE returns the rate of change from the previous value to
// the current value
// It acts exactly like COUNTER but without the overflow checks, and it can
// return negative values representing a decrease of the value
func (t *MuninTypeDerive) Calculate(value float64, previous float64, duration float64) float64 {
	return (value - previous) / (duration / float64(time.Second))
}

// registerType registers an available DST handler
func registerType(typeHandler MuninType) {
	registeredTypes[typeHandler.Name()] = typeHandler
}

// init registers the DST handlers
func init() {
	registerType(&MuninTypeGauge{})
	registerType(&MuninTypeCounter{})
	registerType(&MuninTypeDerive{})
}
