package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MQTT   MQTTConfig    `yaml:"mqtt"`
	HTTP   HTTPConfig    `yaml:"http"`
	Meters []MeterConfig `yaml:"meters"`
}

type MQTTConfig struct {
	Broker   string `yaml:"broker"`
	ClientID string `yaml:"client_id"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type HTTPConfig struct {
	Listen string `yaml:"listen"`
}

type MeterConfig struct {
	Name   string        `yaml:"name"`
	Device string        `yaml:"device"`
	Values []ValueConfig `yaml:"values"`
}

type ValueConfig struct {
	OBIS        string  `yaml:"obis"`
	Name        string  `yaml:"name"`
	DeviceClass string  `yaml:"device_class"`
	StateClass  string  `yaml:"state_class"`
	Unit        string  `yaml:"unit"`
	Factor      float64 `yaml:"factor"`
}

func (v ValueConfig) OBISBytes() ([]byte, error) {
	parts := strings.Split(v.OBIS, ".")
	result := make([]byte, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return nil, fmt.Errorf("invalid OBIS code byte: %s", p)
		}
		result[i] = byte(n)
	}
	return result, nil
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.MQTT.Username == "CHANGE_ME" || cfg.MQTT.Password == "CHANGE_ME" {
		return nil, fmt.Errorf("MQTT username/password still set to 'CHANGE_ME' — copy config.example.yaml to config.yaml and set real credentials")
	}
	if cfg.MQTT.ClientID == "" {
		cfg.MQTT.ClientID = "zaehler2mqtt"
	}
	if cfg.HTTP.Listen == "" {
		cfg.HTTP.Listen = ":8080"
	}
	for i := range cfg.Meters {
		for j := range cfg.Meters[i].Values {
			if cfg.Meters[i].Values[j].Factor == 0 {
				cfg.Meters[i].Values[j].Factor = 1.0
			}
		}
	}
	return &cfg, nil
}
