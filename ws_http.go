package ignite

import (
	"net/http"
)

// Create a Hub listen on a specific address
// Require redis instance config to scale to multi nodes
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
