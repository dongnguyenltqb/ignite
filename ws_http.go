package ignite

import (
	"net/http"
)

// Create a Hub listen on a specific address.
// Require redis instance config to scale to multi nodes.
func NewServer(addr string, listen_path string, namespace string, redis_addr string, redis_password string, redis_db int) *Hub {
	setRedisConfig(redis_addr, redis_password, redis_db)
	hub := newHub(namespace)
	handler := http.NewServeMux()
	handler.HandleFunc(listen_path, hub.serveWs)
	go func() {
		err := http.ListenAndServe(addr, handler)
		if err != nil {
			panic(err)
		}
	}()
	return hub
}
