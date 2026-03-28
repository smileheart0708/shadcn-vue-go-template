package users

const (
	AuthLogEventLoginFailed                 = "login_failed"
	AuthLogEventLoginSuccess                = "login_success"
	AuthLogEventRegisterSuccess             = "register_success"
	AuthLogEventRefreshFailed               = "refresh_failed"
	AuthLogEventRefreshSuccess              = "refresh_success"
	AuthLogEventTokenReuseDetected          = "token_reuse_detected"
	AuthLogEventLogout                      = "logout"
	AuthLogEventPasswordChanged             = "password_changed"
	AuthLogEventPasswordChangedForcedLogout = "password_changed_forced_logout"
)

func IsValidAuthLogEventType(eventType string) bool {
	switch eventType {
	case
		AuthLogEventLoginFailed,
		AuthLogEventLoginSuccess,
		AuthLogEventRegisterSuccess,
		AuthLogEventRefreshFailed,
		AuthLogEventRefreshSuccess,
		AuthLogEventTokenReuseDetected,
		AuthLogEventLogout,
		AuthLogEventPasswordChanged,
		AuthLogEventPasswordChangedForcedLogout:
		return true
	default:
		return false
	}
}
