package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Environment string `default:"dev" split_words:"true"`

	// Server
	Port string `default:"8080" split_words:"true"`

	// Application
	GracefulTimeout int `default:"10" split_words:"true"` // seconds

}

func Load() *Config {
	c := &Config{}
	err := envconfig.Process("", c)

	if err != nil {
		log.Fatal(err.Error())
	}

	return c
}
