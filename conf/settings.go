package conf

import (
	"fmt"
	"os"

	"github.com/go-yaml/yaml"
)

type Data struct {
	Device struct {
		URL      string `yaml:"-"`
		Host     string `yaml:"host"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"device"`

	Export struct {
		Password         string `yaml:"password"` // Configfile Backup req. a password
		Phonebooks       bool   `yaml:"phone_books"`
		PhoneAssets      bool   `yaml:"phone_assets"`
		PhoneBarringList bool   `yaml:"phone_barringlist"`
	} `yaml:"export"`

	Backup struct {
		TargetPath string `yaml:"target_path"`
	} `yaml:"backup"`
}

const (
	configFilename string = "backup-config.yml"
)

func New() (Data, error) {
	var Config Data
	// Read the file
	data, err := os.ReadFile(configFilename)
	if err != nil {
		return Config, err
	}

	// Unmarshal the YAML data into the struct
	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		return Config, err
	}

	Config.Device.URL = fmt.Sprintf("http://%s", Config.Device.Host)
	return Config, err
}
