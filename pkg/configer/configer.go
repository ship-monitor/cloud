package configer

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func GetOrDefault(key, defaultValue string) string {
	val := os.Getenv(key)

	if val == "" {
		return defaultValue
	}

	return val
}

func GetDuration(key string) (time.Duration, error) {
	val := os.Getenv(key)

	duration, err := time.ParseDuration(val)
	if err != nil {
		return time.Second * 0, fmt.Errorf("parse duration: %w", err)
	}

	return duration, nil
}

func GetDurationOrDefault(
	key string,
	defaultValue time.Duration,
) time.Duration {
	val, err := GetDuration(key)
	if err != nil {
		return defaultValue
	}

	return val
}

func GetStrings(key, separator string) []string {
	val := os.Getenv(key)

	splitVal := strings.Split(val, separator)

	valStrings := []string{}

	for _, s := range splitVal {
		s = strings.TrimSpace(s)
		if s != "" {
			valStrings = append(valStrings, s)
		}
	}

	return valStrings
}
