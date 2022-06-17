package utils

import (
	"log"
	"os"
	"strconv"
)

func GetEnvDBHostname() string {
	return getEnv("DB_HOSTNAME", "scstore-database")
}

func GetEnvDBPort() int {
	val := getEnv("DB_PORT", "5432")
	port, err := strconv.Atoi(val)
	if err != nil {
		log.Fatalf("DB_PORT should be integer, but %s: %v", val, err)
	}

	return port
}

func GetEnvDBUsername() string {
	return getEnv("DB_USERNAME", "scstore")
}

func GetEnvDBPassword() string {
	return getEnv("DB_PASSWORD", "scstore")
}

func GetEnvDBName() string {
	return getEnv("DB_NAME", "scstore")
}

func getEnv(key, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultVal
}
