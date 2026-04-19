package authorization

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"sort"
	"time"

	"main/internal/database"
)

const (
	RoleOwner = "owner"
	RoleAdmin = "admin"
	RoleUser  = "user"
)

const (
	CapabilitySystemSettingsRead       = "system.settings.read"
	CapabilitySystemSettingsUpdate     = "system.settings.update"
	CapabilityManagementUsersRead      = "management.users.read"
	CapabilityManagementUsersCreate    = "management.users.create"
	CapabilityManagementUsersUpdate    = "management.users.update"
	CapabilityManagementUsersDisable   = "management.users.disable"
	CapabilityManagementUsersEnable    = "management.users.enable"
	CapabilityManagementAuditLogsRead  = "management.audit_logs.read"
	CapabilityManagementSystemLogsRead = "management.system_logs.read"
	CapabilityAccountDeleteSelf        = "account.delete_self"
)

type ViewerAuthorization struct {
	RoleKeys     []string `json:"roleKeys"`
	Capabilities []string `json:"capabilities"`
}

type ViewerOptions struct {
	AllowUserCreate bool
	AllowSelfDelete bool
}

type UserAction string

const (
	UserActionCreate  UserAction = "create"
	UserActionUpdate  UserAction = "update"
	UserActionDisable UserAction = "disable"
	UserActionEnable  UserAction = "enable"
)

type UserTarget struct {
	UserID   int64
	RoleKeys []string
	Status   string
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

var roleCapabilities = map[string][]string{
	RoleOwner: {
		CapabilitySystemSettingsRead,
		CapabilitySystemSettingsUpdate,
		CapabilityManagementUsersRead,
		CapabilityManagementUsersCreate,
		CapabilityManagementUsersUpdate,
		CapabilityManagementUsersDisable,
		CapabilityManagementUsersEnable,
		CapabilityManagementAuditLogsRead,
		CapabilityManagementSystemLogsRead,
	},
	RoleAdmin: {
		CapabilityManagementUsersRead,
		CapabilityManagementUsersCreate,
		CapabilityManagementUsersUpdate,
		CapabilityManagementUsersDisable,
		CapabilityManagementUsersEnable,
	},
	RoleUser: {},
}

func CatalogRoleKeys() []string {
	return []string{RoleOwner, RoleAdmin, RoleUser}
}

func CatalogCapabilityKeys() []string {
	return []string{
		CapabilitySystemSettingsRead,
		CapabilitySystemSettingsUpdate,
		CapabilityManagementUsersRead,
		CapabilityManagementUsersCreate,
		CapabilityManagementUsersUpdate,
		CapabilityManagementUsersDisable,
		CapabilityManagementUsersEnable,
		CapabilityManagementAuditLogsRead,
		CapabilityManagementSystemLogsRead,
	}
}

func CatalogRolePermissions() map[string][]string {
	copyMap := make(map[string][]string, len(roleCapabilities))
	for roleKey, capabilities := range roleCapabilities {
		copyMap[roleKey] = append([]string(nil), capabilities...)
	}
	return copyMap
}

func EnsureCatalog(ctx context.Context, db *sql.DB) error {
	now := time.Now().UTC().Unix()

	return database.WithTx(ctx, db, func(tx *sql.Tx) error {
		for _, roleKey := range CatalogRoleKeys() {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO roles (key, created_at) VALUES (?, ?)`, roleKey, now); err != nil {
				return fmt.Errorf("authorization: seed role %q: %w", roleKey, err)
			}
		}

		for _, capabilityKey := range CatalogCapabilityKeys() {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO permissions (key, created_at) VALUES (?, ?)`, capabilityKey, now); err != nil {
				return fmt.Errorf("authorization: seed permission %q: %w", capabilityKey, err)
			}
		}

		for roleKey, capabilities := range roleCapabilities {
			for _, capabilityKey := range capabilities {
				if _, err := tx.ExecContext(
					ctx,
					`INSERT OR IGNORE INTO role_permissions (role_key, permission_key, created_at) VALUES (?, ?, ?)`,
					roleKey,
					capabilityKey,
					now,
				); err != nil {
					return fmt.Errorf("authorization: seed role permission %q -> %q: %w", roleKey, capabilityKey, err)
				}
			}
		}

		return nil
	})
}

func (s *Service) ViewerAuthorization(roleKeys []string, options ViewerOptions) ViewerAuthorization {
	capabilities := s.CapabilitiesForRoles(roleKeys)
	if !options.AllowUserCreate {
		capabilities = removeCapability(capabilities, CapabilityManagementUsersCreate)
	}
	if options.AllowSelfDelete && !slices.Contains(capabilities, CapabilityAccountDeleteSelf) {
		capabilities = append(capabilities, CapabilityAccountDeleteSelf)
	}
	sort.Strings(capabilities)

	nextRoleKeys := append([]string(nil), roleKeys...)
	sort.Strings(nextRoleKeys)
	return ViewerAuthorization{
		RoleKeys:     nextRoleKeys,
		Capabilities: capabilities,
	}
}

func (s *Service) CapabilitiesForRoles(roleKeys []string) []string {
	unique := make(map[string]struct{})
	for _, roleKey := range roleKeys {
		for _, capabilityKey := range roleCapabilities[roleKey] {
			unique[capabilityKey] = struct{}{}
		}
	}

	capabilities := make([]string, 0, len(unique))
	for capabilityKey := range unique {
		capabilities = append(capabilities, capabilityKey)
	}
	sort.Strings(capabilities)
	return capabilities
}

func (s *Service) HasCapability(capabilities []string, capability string) bool {
	return slices.Contains(capabilities, capability)
}

func (s *Service) AllowedManagedRoleKeys(actorRoleKeys []string, allowCreate bool) []string {
	if !allowCreate {
		return nil
	}

	switch {
	case slices.Contains(actorRoleKeys, RoleOwner):
		return []string{RoleUser, RoleAdmin}
	case slices.Contains(actorRoleKeys, RoleAdmin):
		return []string{RoleUser}
	default:
		return nil
	}
}

func (s *Service) CanManageUser(actor UserTarget, target UserTarget, action UserAction) bool {
	if actor.UserID == target.UserID {
		return false
	}
	if len(target.RoleKeys) == 0 {
		return false
	}

	switch {
	case slices.Contains(actor.RoleKeys, RoleOwner):
		if slices.Contains(target.RoleKeys, RoleOwner) {
			return false
		}
		return action == UserActionUpdate || action == UserActionDisable || action == UserActionEnable
	case slices.Contains(actor.RoleKeys, RoleAdmin):
		if !slices.Equal(target.RoleKeys, []string{RoleUser}) {
			return false
		}
		return action == UserActionUpdate || action == UserActionDisable || action == UserActionEnable
	default:
		return false
	}
}

func (s *Service) ManagedUserActions(actor UserTarget, target UserTarget) []string {
	actions := make([]string, 0, 2)
	if s.CanManageUser(actor, target, UserActionUpdate) {
		actions = append(actions, string(UserActionUpdate))
	}
	switch target.Status {
	case "active":
		if s.CanManageUser(actor, target, UserActionDisable) {
			actions = append(actions, string(UserActionDisable))
		}
	case "disabled":
		if s.CanManageUser(actor, target, UserActionEnable) {
			actions = append(actions, string(UserActionEnable))
		}
	}
	return actions
}

func removeCapability(capabilities []string, capability string) []string {
	filtered := capabilities[:0]
	for _, currentCapability := range capabilities {
		if currentCapability == capability {
			continue
		}
		filtered = append(filtered, currentCapability)
	}
	return filtered
}
