package ignite

import (
	"context"
	"fmt"
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := rdb.Ping(ctx)
		if cmd.Err() != nil {
			panic(cmd.Err())
		}
		fmt.Println("connected to redis ")

	})
	return rdb
}

func setRedisConfig(addr string, password string, db uint) {
	redisConfig = &redis.Options{
		Addr:        addr,
		Password:    password,
		DB:          int(db),
		DialTimeout: 3 * time.Second,
	}
}
