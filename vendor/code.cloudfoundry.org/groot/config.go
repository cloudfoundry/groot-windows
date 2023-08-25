package groot

import (
	"io/ioutil"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type config struct {
	LogLevel           string   `yaml:"log_level"`
	LogFormat          string   `yaml:"log_format"`
	InsecureRegistries []string `yaml:"insecure_registries"`
}

func parseConfig(configFilePath string) (conf config, err error) {
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
		return config{}, errors.Wrap(err, "reading config file")
	}

	if err := yaml.Unmarshal(contents, &conf); err != nil {
		return config{}, errors.Wrap(err, "parsing config file")
	}

	return conf, nil
}

func applyDefaults(conf config) config {
	if conf.LogLevel == "" {
		conf.LogLevel = "info"
	}
	if conf.LogFormat == "" {
		conf.LogFormat = "rfc3339"
	}
	return conf
}
