package users

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	defaultSuperAdminUsername = "admin"
	bootstrapPasswordLength   = 8
	bootstrapPasswordFileName = "superadmin-password.txt"
)

type BootstrapManager struct {
	store   *Store
	dataDir string
	logger  *slog.Logger
}

func NewBootstrapManager(store *Store, dataDir string, logger *slog.Logger) *BootstrapManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &BootstrapManager{
		store:   store,
		dataDir: dataDir,
		logger:  logger,
	}
}

func (m *BootstrapManager) Ensure(ctx context.Context) error {
	if err := os.MkdirAll(AvatarDir(m.dataDir), 0o755); err != nil {
		return fmt.Errorf("users: failed to ensure avatar dir: %w", err)
	}
	if err := os.MkdirAll(BootstrapDir(m.dataDir), 0o700); err != nil {
		return fmt.Errorf("users: failed to ensure bootstrap dir: %w", err)
	}

	superAdmin, err := m.store.GetFirstSuperAdmin(ctx)
	if errors.Is(err, ErrUserNotFound) {
		return m.createBootstrapSuperAdmin(ctx)
	}
	if err != nil {
		return err
	}

	if !superAdmin.BootstrapPasswordActive {
		return nil
	}

	password, err := os.ReadFile(BootstrapPasswordPath(m.dataDir))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("users: failed to read bootstrap password file: %w", err)
		}

		return m.rotateBootstrapPassword(ctx, superAdmin)
	}

	m.logger.Warn("bootstrap super admin password is still active", "username", superAdmin.Username, "password", string(password))
	return nil
}

func (m *BootstrapManager) createBootstrapSuperAdmin(ctx context.Context) error {
	password, err := randomPassword(bootstrapPasswordLength)
	if err != nil {
		return err
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return err
	}

	superAdmin, err := m.store.Create(ctx, CreateParams{
		Username:                defaultSuperAdminUsername,
		PasswordHash:            passwordHash,
		Role:                    RoleSuperAdmin,
		BootstrapPasswordActive: true,
	})
	if err != nil {
		return err
	}

	if err := writeBootstrapPasswordFile(m.dataDir, password); err != nil {
		return err
	}

	m.logger.Warn("bootstrap super admin password is still active", "username", superAdmin.Username, "password", password)
	return nil
}

func (m *BootstrapManager) rotateBootstrapPassword(ctx context.Context, user User) error {
	password, err := randomPassword(bootstrapPasswordLength)
	if err != nil {
		return err
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return err
	}

	if _, err := m.store.UpdatePassword(ctx, user.ID, passwordHash, true); err != nil {
		return err
	}

	if err := writeBootstrapPasswordFile(m.dataDir, password); err != nil {
		return err
	}

	m.logger.Warn("bootstrap super admin password is still active", "username", user.Username, "password", password)
	return nil
}

func AvatarDir(dataDir string) string {
	return filepath.Join(dataDir, "avatars")
}

func BootstrapDir(dataDir string) string {
	return filepath.Join(dataDir, "bootstrap")
}

func BootstrapPasswordPath(dataDir string) string {
	return filepath.Join(BootstrapDir(dataDir), bootstrapPasswordFileName)
}

func ClearBootstrapPasswordFile(dataDir string) error {
	err := os.Remove(BootstrapPasswordPath(dataDir))
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("users: failed to delete bootstrap password file: %w", err)
}

func writeBootstrapPasswordFile(dataDir string, password string) error {
	if err := os.WriteFile(BootstrapPasswordPath(dataDir), []byte(password), 0o600); err != nil {
		return fmt.Errorf("users: failed to write bootstrap password file: %w", err)
	}
	return nil
}

func randomPassword(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

	buffer := make([]byte, length)
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("users: failed to generate random password: %w", err)
	}

	for index := range buffer {
		buffer[index] = alphabet[int(randomBytes[index])%len(alphabet)]
	}

	return string(buffer), nil
}
