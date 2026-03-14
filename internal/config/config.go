package config

import (
	"fmt"
	"time"

	"github.com/ysmood/goe"
)

const (
	DefaultJWTSecret = "change-me-in-production"
)

var DataDir = goe.Get("DATA_DIR", ".data")
var DBName = goe.Get("DB_NAME", "data.db")
var Port = goe.Get("PORT", 8080)
var FrontendDistDir = goe.Get("WEB_DIST_DIR", "web/dist")
var APIRequestLogEnabled = goe.Get("API_REQUEST_LOG_ENABLED", false)
var JWTSecret = goe.Get("JWT_SECRET", DefaultJWTSecret)
var JWTIssuer = goe.Get("JWT_ISSUER", "shadcn-vue-go-template")
var JWTTTL = mustParseDuration("JWT_TTL", "24h")

func mustParseDuration(name string, fallback string) time.Duration {
	value := goe.Get(name, fallback)

	parsed, err := time.ParseDuration(value)
	if err != nil {
		panic(fmt.Sprintf("config %s must be a valid duration: %v", name, err))
	}

	return parsed
}
