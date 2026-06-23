package keyval

import (
	"charm.land/log/v2"
	"github.com/redis/go-redis/v9"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/config"
)

var RDB *redis.Client

func Setup() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     config.Config.GetString("redis-address"),
		Password: config.Config.GetString("redis-password"),
		DB:       0,
	})
}

func Close() {
	err := RDB.Close()
	if err != nil {
		log.Error("failed to close redis", "error", err)
	}
}
