package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	env := getEnv("TEST_DUMMY_ENV", "defaultVal")
	assert.Equal(t, "defaultVal", env)

	os.Setenv("TEST_DUMMY_ENV", "newVal")
	env = getEnv("TEST_DUMMY_ENV", "defaultVal")
	assert.Equal(t, "newVal", env)
}
