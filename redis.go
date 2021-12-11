package ignite

import (
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var onceSetupRedis sync.Once
var rdb *redis.Client
var redisConfig *redis.Options

func getRedis() *redis.Client {
	onceSetupRedis.Do(func() {
		rdb = redis.NewClient(redisConfig)
	})
	return rdb
}

func setRedisConfig(addr string, password string, db int) {
	redisConfig = &redis.Options{
		Addr:        addr,
		Password:    password,
		DB:          db,
		DialTimeout: 3 * time.Second,
	}
}
