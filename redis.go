package ignite

import (
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var onceSetupRedis sync.Once
var rdb *redis.Client

func getRedis() *redis.Client {
	onceSetupRedis.Do(func() {
		db, _ := strconv.Atoi(os.Getenv("ignite_redis_db"))
		rdb = redis.NewClient(&redis.Options{
			Addr:        os.Getenv("ignite_redis_addr"),
			Password:    os.Getenv("ignite_redis_password"),
			DB:          db,
			DialTimeout: 3 * time.Second,
		})
	})
	return rdb
}
