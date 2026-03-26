package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ysmood/goe"
)

const (
	jwtSecretFileName     = ".jwt_secret"
	jwtSecretEntropyBytes = 64
)

var errEmptyJWTSecretFile = errors.New("config JWT secret file is empty")

type Config struct {
	DataDir              string
	DBName               string
	Port                 int
	APIRequestLogEnabled bool
	JWTSecret            string
	JWTTTL               time.Duration
	RefreshIdleTTL       time.Duration
	RefreshAbsoluteTTL   time.Duration
	RefreshCookieName    string
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

	jwtSecret, err := loadJWTSecret(dataDir)
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

	refreshCookieName, err := getEnv("REFRESH_COOKIE_NAME", "")
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
		RefreshCookieName:    refreshCookieName,
	}, nil
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

func loadJWTSecret(dataDir string) (string, error) {
	if secret, ok := lookupNonEmptyEnv("JWT_SECRET"); ok {
		return secret, nil
	}

	secretPath := filepath.Join(dataDir, jwtSecretFileName)
	secret, err := readJWTSecretFile(secretPath)
	switch {
	case err == nil:
		return secret, nil
	case !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errEmptyJWTSecretFile):
		return "", fmt.Errorf("config JWT_SECRET: %w", err)
	}

	secret, err = generateJWTSecret()
	if err != nil {
		return "", fmt.Errorf("config JWT_SECRET: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", fmt.Errorf("config JWT_SECRET: create data dir: %w", err)
	}
	if err := os.WriteFile(secretPath, []byte(secret), 0o600); err != nil {
		return "", fmt.Errorf("config JWT_SECRET: write secret file: %w", err)
	}

	return secret, nil
}

func readJWTSecretFile(secretPath string) (string, error) {
	secret, err := os.ReadFile(secretPath)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(string(secret))
	if trimmed == "" {
		return "", errEmptyJWTSecretFile
	}

	return trimmed, nil
}

func generateJWTSecret() (string, error) {
	secret := make([]byte, jwtSecretEntropyBytes)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}

	return hex.EncodeToString(secret), nil
}

func lookupNonEmptyEnv(name string) (string, bool) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return "", false
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	return trimmed, true
}
