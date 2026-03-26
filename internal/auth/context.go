package auth

import (
	"context"
	"strings"
)

type contextKey struct{}

var principalContextKey contextKey

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(Principal)
	return principal, ok
}

func BearerTokenFromHeader(header string) (string, error) {
	if header == "" {
		return "", ErrMissingBearerToken
	}

	prefix, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(prefix, "Bearer") {
		return "", ErrMissingBearerToken
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrMissingBearerToken
	}

	return token, nil
}
