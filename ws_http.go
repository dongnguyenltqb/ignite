package ignite

import (
	"fmt"
	"net/http"
)

func NewServer(addr string, redis_addr string, redis_password string, redis_db int) *Hub {
	setRedisConfig(redis_addr, redis_password, redis_db)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		getHub().serveWs(w, r)
	})
	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			fmt.Print(err)
		}
	}()
	getLogger().Info("Hub is running on ", addr)
	return getHub()
}
