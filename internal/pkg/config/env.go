package config

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Env struct {
	Port   int    `env:"PORT"`
	DBPath string `env:"DB_PATH"`

	Log Log `env:", prefix=LOG_"`
	DB  DB  `env:", prefix=DB_"`
}

type DB struct {
	URL string `env:"URL"`
}
type Log struct {
	Level string `env:"LEVEL"`
}

func NewEnv() (Env, error) {
	var env Env

	envPath := os.Getenv("CONFPATH")
	if envPath != "" {
		_ = godotenv.Load(envPath)
	}

	if err := envconfig.Process(context.Background(), &env); err != nil {
		return env, err
	}

	if err := validateEnv(&env); err != nil {
		return env, err
	}

	return env, nil
}

// validateEnv checks for required env vars
// set up default values
func validateEnv(env *Env) error {
	if env.DBPath == "" {
		env.DBPath = "karden.db"
	}
	if env.Port == 0 {
		env.Port = 8080
	}
	return nil
}
