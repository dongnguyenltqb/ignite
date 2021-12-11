package ignite

import (
	"fmt"
	"net/http"
)

func NewServer(addr string) *Hub {
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
