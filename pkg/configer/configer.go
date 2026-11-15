package configer

import "os"

func GetOrDefault(key, defaultValue string) string {
	val := os.Getenv(key)

	if val == "" {
		return defaultValue
	}

	return val
}
