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
	IsBanned                bool
	BannedAt                *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type PublicUser struct {
	ID                 int64   `json:"id"`
	Username           string  `json:"username"`
	Email              *string `json:"email"`
	AvatarURL          *string `json:"avatarUrl"`
	Role               int     `json:"role"`
	MustChangePassword bool    `json:"mustChangePassword"`
}

type AdminUser struct {
	ID                 int64      `json:"id"`
	Username           string     `json:"username"`
	Email              *string    `json:"email"`
	AvatarURL          *string    `json:"avatarUrl"`
	Role               int        `json:"role"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"createdAt"`
	BannedAt           *time.Time `json:"bannedAt"`
	MustChangePassword bool       `json:"mustChangePassword"`
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

func (u User) Admin() AdminUser {
	status := "active"
	if u.IsBanned {
		status = "banned"
	}

	return AdminUser{
		ID:                 u.ID,
		Username:           u.Username,
		Email:              cloneStringPointer(u.Email),
		AvatarURL:          avatarURL(u.AvatarPath),
		Role:               u.Role,
		Status:             status,
		CreatedAt:          u.CreatedAt,
		BannedAt:           cloneTimePointer(u.BannedAt),
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

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	copy := *value
	return &copy
}
