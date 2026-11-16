package configer

import (
	"os"
	"strings"
)

func GetOrDefault(key, defaultValue string) string {
	val := os.Getenv(key)

	if val == "" {
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
