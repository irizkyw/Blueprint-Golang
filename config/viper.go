package config

import (
	"errors"
	"os"

	"github.com/spf13/viper"
)

type EnvStructs struct {
	DB_HOST     string `mapstructure:"DB_HOST"`
	DB_PORT     string `mapstructure:"DB_PORT"`
	DB_DATABASE string `mapstructure:"DB_DATABASE"`
	DB_USER     string `mapstructure:"DB_USER"`
	DB_PASSWORD string `mapstructure:"DB_PASSWORD"`
	PORT        string `mapstructure:"PORT"`
}

func LoadConfig() (config EnvStructs, err error) {
	env := os.Getenv("GO_ENV")
	if env == "production" || env == "development" {
		return EnvStructs{
			DB_HOST:     os.Getenv("DB_HOST"),
			DB_PORT:     os.Getenv("DB_PORT"),
			DB_DATABASE: os.Getenv("DB_NAME"),
			DB_USER:     os.Getenv("DB_USER"),
			DB_PASSWORD: os.Getenv("DB_PASSWORD"),
			PORT:        os.Getenv("PORT"),
		}, nil
	}

	viper.AddConfigPath(".")
	viper.AddConfigPath("/app")

	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()
	err = viper.ReadInConfig()

	if err != nil {
		return EnvStructs{}, err
	}

	err = viper.Unmarshal(&config)

	if config.DB_HOST == "" {
		err = errors.New("DB_HOST is required")
		return
	}
	if config.DB_PORT == "" {
		err = errors.New("DB_PORT is required")
		return
	}
	if config.DB_DATABASE == "" {
		err = errors.New("DB_DATABASE is required")
		return
	}
	if config.DB_USER == "" {
		err = errors.New("DB_USER is required")
		return
	}
	if config.DB_PASSWORD == "" && env == "production" {
		err = errors.New("DB_PASSWORD is required")
		return
	}
	return
}
