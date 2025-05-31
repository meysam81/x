package config

import (
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
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
	defaultValues    map[string]interface{}
	defaultProvided  bool
	unmarshalTo      interface{}
	unmarshalConf    *UnmarshalConf
}

type Config = koanf.Koanf
type UnmarshalConf = koanf.UnmarshalConf

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

func WithDefaults(defaults map[string]interface{}) func(*options) {
	return func(o *options) {
		o.defaultValues = defaults
		o.defaultProvided = true
	}
}

func WithUnmarshalTo(obj interface{}) func(*options) {
	return func(o *options) {
		o.unmarshalTo = obj
	}
}

func WithUnmarshalConf(c *UnmarshalConf) func(*options) {
	return func(o *options) {
		o.unmarshalConf = c
	}
}

func NewConfig(opts ...func(*options)) (*Config, error) {
	o := &options{
		delimiter:        ".",
		jsonWatchEnabled: true,
		yamlWatchEnabled: true,
		defaultProvided:  false,
		unmarshalTo:      nil,
		unmarshalConf:    &UnmarshalConf{Tag: "koanf", FlatPaths: true},
	}

	for _, opt := range opts {
		opt(o)
	}

	k := koanf.New(o.delimiter)

	if o.defaultProvided {
		err := k.Load(confmap.Provider(o.defaultValues, o.delimiter), nil)
		if err != nil {
			return nil, err
		}
	}

	if o.jsonConfig != "" {
		_, err := os.Stat(o.jsonConfig)
		if err != nil {
			return nil, err
		}

		f := file.Provider(o.jsonConfig)

		// TODO: provide watch functionality

		err = k.Load(f, json.Parser())
		if err != nil {
			return nil, err
		}
	}

	if o.yamlConfig != "" {
		_, err := os.Stat(o.yamlConfig)
		if err != nil {
			return nil, err
		}

		err = k.Load(file.Provider(o.yamlConfig), yaml.Parser())
		if err != nil {
			return nil, err
		}
	}

	err := k.Load(env.ProviderWithValue(o.envPrefix, o.delimiter, func(key, value string) (string, interface{}) {
		k := strings.TrimPrefix(key, o.envPrefix)
		k = strings.ToLower(k)
		k = strings.ReplaceAll(k, "__", "-")        // BASE__URL => base-url
		k = strings.ReplaceAll(k, "_", o.delimiter) // SERVE_PORT => serve.port

		if strings.Contains(value, " ") {
			return k, strings.Split(value, " ")
		}

		return k, value
	}), nil)

	if err != nil {
		return nil, err
	}

	if o.unmarshalTo != nil {
		err := k.UnmarshalWithConf("", o.unmarshalTo, *o.unmarshalConf)
		if err != nil {
			return nil, err
		}
	}

	return k, nil
}
