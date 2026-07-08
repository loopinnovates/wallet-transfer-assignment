package config_test

import (
	"os"
	"testing"

	"github.com/loopinnovates/wallet-transfer-assignment/internal/config"
)

func TestAppConfig_Success(t *testing.T) {
	expectedPort := "9090"
	err := os.Setenv(config.PortKey, expectedPort)
	if err != nil {
		t.Fatal(err)
	}

	config := config.AppConfig()

	if config.Port != expectedPort {
		t.Errorf("Expected port: %s, but got: %s", expectedPort, config.Port)
	}
}

func TestAppConfig_DefaultValues(t *testing.T) {
	err := os.Unsetenv(config.PortKey)
	if err != nil {
		t.Fatal(err)
	}

	config := config.AppConfig()
	if config.Port != "8080" {
		t.Errorf("Expected default port: 8080, but got: %s", config.Port)
	}
}
