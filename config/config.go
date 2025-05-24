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
	envPrefix        string
}

type Config = koanf.Koanf
type unloadedConfig struct {
	k *Config
	o *options
}

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

func WithEnvPrefix(p string) func(*options) {
	return func(o *options) {
		o.envPrefix = p
	}
}

func NewConfig(opts ...func(*options)) (*unloadedConfig, error) {
	o := &options{
		delimiter:        ".",
		jsonWatchEnabled: true,
		yamlWatchEnabled: true,
	}

	for _, opt := range opts {
		opt(o)
	}

	k := koanf.New(o.delimiter)

	return &unloadedConfig{
		k,
		o,
	}, nil
}

func (u *unloadedConfig) Build() (err error) {
	err = u.k.Load(env.ProviderWithValue(u.o.envPrefix, ".", func(key, value string) (string, interface{}) {
		k := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(key, u.o.envPrefix), "_", "."))

		if strings.Contains(value, " ") {
			return k, strings.Split(value, " ")
		}

		return k, value
	}), nil)

	if err != nil {
		return err
	}

	if u.o.jsonConfig != "" {
		_, err := os.Stat(u.o.jsonConfig)
		if err != nil {
			return err
		}

		f := file.Provider(u.o.jsonConfig)

		// TODO: provide watch functionality

		err = u.k.Load(f, json.Parser())
		if err != nil {
			return err
		}
	}

	if u.o.yamlConfig != "" {
		_, err := os.Stat(u.o.yamlConfig)
		if err != nil {
			return err
		}

		err = u.k.Load(file.Provider(u.o.yamlConfig), yaml.Parser())
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *unloadedConfig) Koanf() *Config {
	return u.k
}
