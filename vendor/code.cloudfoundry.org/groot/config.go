package groot

import (
	"io/ioutil"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	LogLevel string `yaml:"log_level"`
	Store    string `yaml:"store"`
}

func parseConfig(configFilePath string) (conf Config, err error) {
	defer func() {
		if err == nil {
			conf = applyDefaults(conf)
		}
	}()

	if configFilePath == "" {
		return conf, nil
	}

	contents, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return Config{}, errors.Wrap(err, "reading config file")
	}

	if err := yaml.Unmarshal(contents, &conf); err != nil {
		return Config{}, errors.Wrap(err, "parsing config file")
	}

	return conf, nil
}

func applyDefaults(conf Config) Config {
	if conf.LogLevel == "" {
		conf.LogLevel = "info"
	}
	return conf
}
