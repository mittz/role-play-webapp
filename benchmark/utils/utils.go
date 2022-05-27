package utils

import (
	"log"
	"os"
	"strconv"
)

func GetMin(x, y int) int {
	if x < y {
		return x
	}

	return y
}

func getEnv(key, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultVal
}

func GetEnvPortalPort() int {
	val := getEnv("PORTAL_PORT", "8080")
	port, err := strconv.Atoi(val)
	if err != nil {
		log.Fatalf("PORTAL_PORT: %s should be integer: %v", val, err)
	}

	return port
}

func GetEnvDataStudioURL() string {
	return getEnv("DS_URL", "")
}

func GetEnvDBHostname() string {
	return getEnv("DB_HOSTNAME", "score-database")
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
	return getEnv("DB_USERNAME", "score")
}

func GetEnvDBPassword() string {
	return getEnv("DB_PASSWORD", "score")
}

func GetEnvDBName() string {
	return getEnv("DB_NAME", "score")
}

func GetEnvSAKey() string {
	return getEnv("GOOGLE_APPLICATION_CREDENTIALS", "")
}

func GetEnvProjectID() string {
	return getEnv("PROJECT_ID", "")
}
