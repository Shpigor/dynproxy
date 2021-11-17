package dynproxy

import (
	"log"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	yamlConfig := LoadConfig("./cmd/config.yaml")
	log.Printf("%+v", yamlConfig)
	tomlConfig := LoadConfig("./cmd/config.toml")
	log.Printf("%+v", tomlConfig)
}
