package users

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	passwordSaltBytes    = 16
	passwordHashBytes    = 32
	passwordArgonTime    = 3
	passwordArgonMemory  = 64 * 1024
	passwordArgonThreads = 2
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("users: failed to generate password salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, passwordArgonTime, passwordArgonMemory, passwordArgonThreads, passwordHashBytes)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		passwordArgonMemory,
		passwordArgonTime,
		passwordArgonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(password string, encodedHash string) (bool, error) {
	params, salt, hash, err := parsePasswordHash(encodedHash)
	if err != nil {
		return false, err
	}

	comparison := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(hash)))
	return subtle.ConstantTimeCompare(hash, comparison) == 1, nil
}

type passwordParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

func parsePasswordHash(encodedHash string) (passwordParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return passwordParams{}, nil, nil, errors.New("users: invalid encoded password hash")
	}
	if parts[1] != "argon2id" {
		return passwordParams{}, nil, nil, errors.New("users: unsupported password hash algorithm")
	}
	if parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return passwordParams{}, nil, nil, errors.New("users: unsupported password hash version")
	}

	params, err := parsePasswordParams(parts[3])
	if err != nil {
		return passwordParams{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return passwordParams{}, nil, nil, fmt.Errorf("users: invalid password hash salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return passwordParams{}, nil, nil, fmt.Errorf("users: invalid password hash value: %w", err)
	}

	return params, salt, hash, nil
}

func parsePasswordParams(raw string) (passwordParams, error) {
	var params passwordParams

	for _, part := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return passwordParams{}, errors.New("users: invalid password hash parameters")
		}

		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return passwordParams{}, fmt.Errorf("users: invalid password hash parameter %s: %w", key, err)
		}

		switch key {
		case "m":
			params.memory = uint32(parsed)
		case "t":
			params.time = uint32(parsed)
		case "p":
			params.threads = uint8(parsed)
		default:
			return passwordParams{}, errors.New("users: unsupported password hash parameter")
		}
	}

	if params.memory == 0 || params.time == 0 || params.threads == 0 {
		return passwordParams{}, errors.New("users: incomplete password hash parameters")
	}

	return params, nil
}
