package ignite

import (
	"net/http"
)

func NewServer(addr string, redis_addr string, redis_password string, redis_db int) *Hub {
	setRedisConfig(redis_addr, redis_password, redis_db)
	hub := newHub()
	handler := http.NewServeMux()
	handler.HandleFunc("/ws", hub.serveWs)

	go func() {
		err := http.ListenAndServe(addr, handler)
		if err != nil {
			panic(err)
		}
	}()
	getLogger().Info("Hub is running on ", addr)
	return hub
}
