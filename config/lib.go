package config

import (
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type options struct {
	delimiter        string
	jsonConfig       string
	jsonWatchEnabled bool
	yamlConfig       string
	yamlWatchEnabled bool
}

type Config koanf.Koanf

func WithDelimiter(delim string) func(*options) {
	return func(o *options) {
		o.delimiter = delim
	}
}

func WithJsonConfig(filename string) func(*options) {
	return func(o *options) {
		o.jsonConfig = filename
	}
}

func WithYamlConfig(filename string) func(*options) {
	return func(o *options) {
		o.yamlConfig = filename
	}
}

func WithoutJsonWatch() func(*options) {
	return func(o *options) {
		o.jsonWatchEnabled = false
	}
}

func WithoutYamlWatch() func(*options) {
	return func(o *options) {
		o.yamlWatchEnabled = false
	}
}

func NewConfig(opts ...func(*options)) (k *Config, err error) {
	o := &options{
		delimiter:        ".",
		jsonWatchEnabled: true,
		yamlWatchEnabled: true,
	}

	for _, opt := range opts {
		opt(o)
	}

	k = koanf.New(o.delimiter)

	err = loadConfig(k, o)
	if err != nil {
		return nil, err
	}

	return k, nil
}

func loadConfig(k *koanf.Koanf, o *options) (err error) {
	if o.jsonConfig != "" {
		_, err := os.Stat(o.jsonConfig)
		if err != nil {
			return err
		}

		f := file.Provider(o.jsonConfig)

		// TODO: provide watch functionality

		err = k.Load(f, json.Parser())
		if err != nil {
			return err
		}
	}

	if o.yamlConfig != "" {
		_, err := os.Stat(o.yamlConfig)
		if err != nil {
			return err
		}

		err = k.Load(file.Provider(o.yamlConfig), yaml.Parser())
		if err != nil {
			return err
		}
	}

	err = k.Load(env.Provider("", "_", func(s string) string {
		return strings.ToLower(strings.ReplaceAll(s, "_", "."))
	}), nil)
	if err != nil {
		return err
	}

	return nil
}
