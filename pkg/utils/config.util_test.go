package utils_test

import (
	"os"
	"testing"

	"github.com/loopinnovates/wallet-transfer-assignment/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetEnvInt_WithValidValue(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	result := utils.GetEnvInt("TEST_INT", 10)
	assert.Equal(t, 42, result)
}

func TestGetEnvInt_WithInvalidValue(t *testing.T) {
	os.Setenv("TEST_INT", "invalid")
	defer os.Unsetenv("TEST_INT")

	assert.Panics(t, func() {
		utils.GetEnvInt("TEST_INT", 10)
	})
}

func TestGetEnvInt_WithEmptyValue(t *testing.T) {
	os.Unsetenv("TEST_INT")

	result := utils.GetEnvInt("TEST_INT", 10)
	assert.Equal(t, 10, result)
}

func TestGetEnvStr_WithValidValue(t *testing.T) {
	os.Setenv("TEST_STR", "hello")
	defer os.Unsetenv("TEST_STR")

	result := utils.GetEnvStr("TEST_STR", "default")
	assert.Equal(t, "hello", result)
}

func TestGetEnvStr_WithEmptyValue(t *testing.T) {
	os.Unsetenv("TEST_STR")

	result := utils.GetEnvStr("TEST_STR", "default")
	assert.Equal(t, "default", result)
}
