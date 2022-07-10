package ignite

import (
	"fmt"
	"net/http"
)

// Create a Hub listen on a specific address.
// Require redis instance config to scale to multi nodes.

type ServerConfig struct {
	Namespace     string
	Address       string
	Path          string
	RedisHost     string
	RedisPort     uint
	RedisPassword string
	RedisDb       uint
}

func NewServer[K comparable, V any](config *ServerConfig) *Hub[K, V] {
	// prepare redis config
	setRedisConfig(config.RedisHost+":"+fmt.Sprintf("%v", config.RedisPort), config.RedisPassword, config.RedisDb)
	hub := newHub[K, V](config.Namespace)

	// spin a http server to handle request
	handler := http.NewServeMux()
	handler.HandleFunc(config.Path, hub.serveWs)
	go func() {
		err := http.ListenAndServe(config.Address, handler)
		if err != nil {
			panic(err)
		}
	}()
	return hub
}
