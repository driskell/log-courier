package event

import (
	"strconv"
	"strings"
)

type FloatValue64 float64

func (f FloatValue64) MarshalJSON() ([]byte, error) {
	result := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(result, '.') {
		result += ".0"
	}
	return []byte(result), nil
}

type FloatValue32 float32

func (f FloatValue32) MarshalJSON() ([]byte, error) {
	result := strconv.FormatFloat(float64(f), 'f', -1, 32)
	if !strings.ContainsRune(result, '.') {
		result += ".0"
	}
	return []byte(result), nil
}
