package main

import (
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// Config describes the config format in config.yml.
type Config struct {
	GoogleOauth   []byte `yaml:"google_oauth"`
	Since         time.Time
	SpreadSheetID string `yaml:"spreadsheet_id"`
	Banks         struct {
		N26 *struct {
			User     string
			Password string
		}
		DB *struct {
			Branch  string
			Account string
			PIN     string
		}
	}
}

// loadConfig loads config from path.
func loadConfig(path string) (*Config, error) {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(d, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
