package main

const (
	defaultType = "GAUGE"
)

type MuninType interface {
	RequiresPrevious() bool
	Calculate(float64, float64) float64
}

type MuninTypeGauge struct {
}

func (t *MuninTypeGauge) RequiresPrevious() bool {
	return false
}

func (t *MuninTypeGauge) Calculate(value float64, previous float64) float64 {
	return value
}

type MuninTypeCounter struct {
}

func (t *MuninTypeCounter) RequiresPrevious() bool {
	return true
}

func (t *MuninTypeCounter) Calculate(value float64, previous float64) float64 {
	if previous <= value {
		return value - previous
	}

	// A wrap occurred and we're supported to detect whether it was at 32-bit or
	// 64-bit boundary - a quick search didn't bring up the exact RRDTool
	// calculations and I've yet to inspect the RRDTool source, so I'm going to
	// guess here

	// Is the previous value less than 2^31? Then treat it as a 32-bit wrap
	if previous <= 2^31 {
		return 2 ^ 31 - previous + value
	}

	return 2 ^ 63 - previous + value
}

type MuninTypeDerive struct {
}

func (t *MuninTypeDerive) RequiresPrevious() bool {
	return true
}

func (t *MuninTypeDerive) Calculate(value float64, previous float64) float64 {
	return value - previous
}

func init() {
	registeredTypes["GAUGE"] = &MuninTypeGauge{}
	registeredTypes["COUNTER"] = &MuninTypeCounter{}
	registeredTypes["DERIVE"] = &MuninTypeDerive{}
}
