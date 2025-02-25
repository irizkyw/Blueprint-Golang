package config

import (
	"errors"
	"os"

	"github.com/spf13/viper"
)

type EnvStructs struct {
	MYSQL_HOST     string `mapstructure:"MYSQL_HOST"`
	MYSQL_PORT     string `mapstructure:"MYSQL_PORT"`
	MYSQL_DB       string `mapstructure:"MYSQL_DB"`
	MYSQL_USER     string `mapstructure:"MYSQL_USER"`
	MYSQL_PASSWORD string `mapstructure:"MYSQL_PASSWORD"`
	PORT           string `mapstructure:"PORT"`
}

func LoadConfig() (config EnvStructs, err error) {
	env := os.Getenv("GO_ENV")
	if env == "production" {
		return EnvStructs{
			MYSQL_HOST:     os.Getenv("MYSQL_HOST"),
			MYSQL_PORT:     os.Getenv("MYSQL_PORT"),
			MYSQL_DB:       os.Getenv("MYSQL_DB"),
			MYSQL_USER:     os.Getenv("MYSQL_USER"),
			MYSQL_PASSWORD: os.Getenv("MYSQL_PASSWORD"),
			PORT:           os.Getenv("PORT"),
		}, nil
	}

	viper.AddConfigPath(".")
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)

	if config.MYSQL_HOST == "" {
		err = errors.New("MYSQL_HOST is required")
		return
	}
	if config.MYSQL_PORT == "" {
		err = errors.New("MYSQL_PORT is required")
		return
	}
	if config.MYSQL_DB == "" {
		err = errors.New("MYSQL_DB is required")
		return
	}
	if config.MYSQL_USER == "" {
		err = errors.New("MYSQL_USER is required")
		return
	}
	if config.MYSQL_PASSWORD == "" && env == "production" {
		err = errors.New("MYSQL_PASSWORD is required")
		return
	}
	return
}
