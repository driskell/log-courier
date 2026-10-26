package pipelinebench

import (
	"sort"
	"time"
)

// percentile returns the p-th percentile of sorted durations
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(p * float64(len(sorted)))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func sortDurations(durations []time.Duration) {
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
}

// jainIndex returns Jain's fairness index of the values, 1 being perfectly equal
func jainIndex(values []float64) float64 {
	var sum, sumSquares float64
	for _, value := range values {
		sum += value
		sumSquares += value * value
	}
	if sumSquares == 0 {
		return 0
	}
	return sum * sum / (float64(len(values)) * sumSquares)
}
