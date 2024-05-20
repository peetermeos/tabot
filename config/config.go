package config

import (
	"os"
	"reflect"

	"github.com/pkg/errors"
)

type Config struct {
	LogLevel     string `env:"LOGLEVEL"`
	AWSRegion    string `env:"AWS_REGION"`
	KrakenKey    string `env:"KRAKEN_API_KEY"`
	KrakenSecret string `env:"KRAKEN_API_SECRET"`
}

var ErrFieldNotDefined = errors.New("environment variable name tag for field is not defined")

func Load() (*Config, error) {
	config := Config{
		LogLevel:  "debug",
		AWSRegion: "us-east-1",
	}

	typeOf := reflect.TypeOf(config)
	valueOf := reflect.Indirect(reflect.ValueOf(&config))

	// using reflection iterate over config struct fields
	for i := 0; i < typeOf.NumField(); i++ {
		// get field name
		field := typeOf.Field(i)

		// get field "env" tag value
		tag, ok := field.Tag.Lookup("env")
		if !ok {
			return nil, errors.Wrap(ErrFieldNotDefined, field.Name)
		}

		// override default config field value with environment variable value if set
		value, ok := os.LookupEnv(tag)
		if ok {
			if field.Type.Kind() == reflect.String {
				valueOf.FieldByName(field.Name).SetString(value)
			}

			if field.Type.Kind() == reflect.Bool {
				valueOf.FieldByName(field.Name).SetBool(true)
			}
		}
	}

	return &config, nil
}
