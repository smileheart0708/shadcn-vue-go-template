package users

import (
	"net/url"
	"path"
	"time"
)

const (
	RoleUser       = 0
	RoleAdmin      = 1
	RoleSuperAdmin = 2
)

type User struct {
	ID                      int64
	Username                string
	Email                   *string
	PasswordHash            string
	AvatarPath              *string
	Role                    int
	BootstrapPasswordActive bool
	AuthVersion             int64
	PasswordChangedAt       *time.Time
}

type PublicUser struct {
	ID                 int64   `json:"id"`
	Username           string  `json:"username"`
	Email              *string `json:"email"`
	AvatarURL          *string `json:"avatarUrl"`
	Role               int     `json:"role"`
	MustChangePassword bool    `json:"mustChangePassword"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:                 u.ID,
		Username:           u.Username,
		Email:              cloneStringPointer(u.Email),
		AvatarURL:          avatarURL(u.AvatarPath),
		Role:               u.Role,
		MustChangePassword: u.BootstrapPasswordActive,
	}
}

func avatarURL(avatarPath *string) *string {
	if avatarPath == nil || *avatarPath == "" {
		return nil
	}

	escaped := "/api/avatars/" + url.PathEscape(path.Base(*avatarPath))
	return &escaped
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	copy := *value
	return &copy
}
