package ignite

import (
	"fmt"
	"net/http"
)

func NewServer(port int) *Hub {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		getHub().serveWs(w, r)
	})
	go func() {
		addr := ":" + fmt.Sprint(port)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			fmt.Print(err)
		}
	}()
	getLogger().Info("Hub is running on port ", port)
	return getHub()
}
