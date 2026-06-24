// Package env provides runtime environment variable helpers for generated
// plugin code. Values are read at application startup and fall back to the
// defaults that were baked in at build time from Polafile.hcl.
package env

import (
	"os"
	"strconv"
)

// String returns the value of the environment variable named by key,
// or fallback if the variable is not set or is empty.
func String(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Bool returns the boolean value of the environment variable named by key,
// or fallback if the variable is not set.
func Bool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return v == "true" || v == "1"
}

// Int returns the integer value of the environment variable named by key,
// or fallback if the variable is not set or cannot be parsed.
func Int(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// Float64 returns the float64 value of the environment variable named by key,
// or fallback if the variable is not set or cannot be parsed.
func Float64(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
