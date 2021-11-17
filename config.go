package dynproxy

import (
	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"strings"
)

type Global struct {
	LogLevel string `yaml:"log_level" toml:"log_level"`
}

type FrontendConfig struct {
	Name          string `yaml:"name" toml:"name"`
	Net           string `yaml:"net" toml:"net"`
	Address       string `yaml:"address" toml:"address"`
	TlsSkipVerify bool   `yaml:"tls_skip_verify" toml:"tls_skip_verify"`
	TlsCACertPath string `yaml:"tls_ca_cert_path" toml:"tls_ca_cert_path"`
	TlsCertPath   string `yaml:"tls_cert_path" toml:"tls_cert_path"`
	TlsPkPath     string `yaml:"tls_pk_path" toml:"tls_pk_path"`
	BackendGroup  string `yaml:"backend_group" toml:"backend_group"`
}

type BackendGroup struct {
	Name     string          `yaml:"name" toml:"name"`
	Backends []BackendConfig `yaml:"servers" toml:"servers"`
}

type BackendConfig struct {
	Name              string `yaml:"name" toml:"name"`
	Net               string `yaml:"net" toml:"net"`
	Address           string `yaml:"address" toml:"address"`
	HealthCheckPeriod int    `yaml:"health_check_period_sec" toml:"health_check_period_sec"`
}

type Config struct {
	Global    Global           `yaml:"global" toml:"global"`
	Frontends []FrontendConfig `yaml:"frontends" toml:"frontends"`
	Backends  []BackendGroup   `yaml:"backends" toml:"backends"`
}

func LoadConfig(filePath string) *Config {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	config := &Config{}
	if strings.HasSuffix(filePath, ".toml") {
		err = toml.Unmarshal(file, config)
	} else if strings.HasSuffix(filePath, ".yaml") {
		err = yaml.Unmarshal(file, config)
	}
	if err != nil {
		log.Fatalf("%+v", err)
	}
	validateConfig(config)
	return config
}

func validateConfig(config *Config) {

}
