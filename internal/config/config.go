package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ysmood/goe"
)

const (
	DefaultJWTSecret = "change-me-in-production"
)

type Config struct {
	DataDir              string
	DBName               string
	Port                 int
	APIRequestLogEnabled bool
	JWTSecret            string
	JWTTTL               time.Duration
	RefreshIdleTTL       time.Duration
	RefreshAbsoluteTTL   time.Duration
}

func Load() error {
	err := goe.Load(false, true, goe.DOTENV)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("load .env: %w", err)
	}

	return nil
}

func LoadConfig() (Config, error) {
	dataDir, err := getEnv("DATA_DIR", ".data")
	if err != nil {
		return Config{}, err
	}

	dbName, err := getEnv("DB_NAME", "data.db")
	if err != nil {
		return Config{}, err
	}

	port, err := getEnv("PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	apiRequestLogEnabled, err := getEnv("API_REQUEST_LOG_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	jwtSecret, err := getEnv("JWT_SECRET", DefaultJWTSecret)
	if err != nil {
		return Config{}, err
	}

	jwtTTL, err := getEnv("JWT_TTL", 10*time.Minute)
	if err != nil {
		return Config{}, err
	}

	refreshIdleTTL, err := getEnv("REFRESH_IDLE_TTL", 7*24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	refreshAbsoluteTTL, err := getEnv("REFRESH_ABSOLUTE_TTL", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	return Config{
		DataDir:              dataDir,
		DBName:               dbName,
		Port:                 port,
		APIRequestLogEnabled: apiRequestLogEnabled,
		JWTSecret:            jwtSecret,
		JWTTTL:               jwtTTL,
		RefreshIdleTTL:       refreshIdleTTL,
		RefreshAbsoluteTTL:   refreshAbsoluteTTL,
	}, nil
}

func (c Config) UsesDefaultJWTSecret() bool {
	return c.JWTSecret == DefaultJWTSecret
}

func getEnv[T goe.EnvType](name string, fallback T) (T, error) {
	return getEnvAny([]string{name}, fallback)
}

func getEnvAny[T goe.EnvType](names []string, fallback T) (T, error) {
	for _, name := range names {
		raw, ok := os.LookupEnv(name)
		if !ok {
			continue
		}

		value, err := goe.Parse[T](raw)
		if err != nil {
			var zero T
			return zero, fmt.Errorf("config %s: %w", name, err)
		}

		return value, nil
	}

	return fallback, nil
}
