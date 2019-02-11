package osx

import "os"

func GetenvDefault(key string, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}
