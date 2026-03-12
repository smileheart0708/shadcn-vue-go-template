package config

import "github.com/ysmood/goe"

var DataDir = goe.Get("DATA_DIR", ".data")
var DBName = goe.Get("DB_NAME", "data.db")
var Port = goe.Get("PORT", 8080)
