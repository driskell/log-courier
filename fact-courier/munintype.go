package main

import "time"

const (
	defaultType = "GAUGE"
)

type MuninType interface {
	Name() string
	RequiresPrevious() bool
	Calculate(float64, float64, float64) float64
}

type MuninTypeGauge struct {
}

func (t *MuninTypeGauge) Name() string {
	return "GAUGE"
}

func (t *MuninTypeGauge) RequiresPrevious() bool {
	return false
}

func (t *MuninTypeGauge) Calculate(value float64, previous float64, duration float64) float64 {
	return value
}

type MuninTypeCounter struct {
}

func (t *MuninTypeCounter) Name() string {
	return "COUNTER"
}

func (t *MuninTypeCounter) RequiresPrevious() bool {
	return true
}

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

type MuninTypeDerive struct {
}

func (t *MuninTypeDerive) Name() string {
	return "DERIVE"
}

func (t *MuninTypeDerive) RequiresPrevious() bool {
	return true
}

func (t *MuninTypeDerive) Calculate(value float64, previous float64, duration float64) float64 {
	return (value - previous) / (duration / float64(time.Second))
}

func registerType(typeHandler MuninType) {
	registeredTypes[typeHandler.Name()] = typeHandler
}

func init() {
	registerType(&MuninTypeGauge{})
	registerType(&MuninTypeCounter{})
	registerType(&MuninTypeDerive{})
}
