package utils

import (
	"encoding/json"
	"strings"
)

// Clamp constrains v to [min, max].
func Clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// SafeParseJSON attempts to unmarshal data into T; returns zero value on error.
func SafeParseJSON[T any](data []byte) (T, error) {
	var result T
	err := json.Unmarshal(data, &result)
	return result, err
}

// IsTruthyEnvValue returns true if the value is "1", "true", or "yes" (case-insensitive).
func IsTruthyEnvValue(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "1" || v == "true" || v == "yes"
}
