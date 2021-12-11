# ignite

websocket server module

pub/sub redis config env, use a single redis instance.

```
ignite_redis_db=10
ignite_redis_addr=localhost:6379
ignite_redis_password=
```

use module like the code below.

```go
package main

import (
	"encoding/json"
	"fmt"
	"ignite"
)

func main() {
	never_die := make(chan bool)
	hub := ignite.NewServer("localhost:8787")
	hub.OnNewClient = func(client *ignite.Client) {
		// Send identity message
		client.SendIdentityMsg()
		// Register handle function with function id
		client.On("buy", "1", func(raw json.RawMessage) {
			fmt.Println("BUY=>", string(raw))
		})
		client.On("sell", "2", func(raw json.RawMessage) {
			fmt.Println("SELL =>", string(raw))
		})
		client.On("stop_buy", "3", func(raw json.RawMessage) {
			// Unregister handle function with function id
			client.Off("buy", "1")
		})
		// Handle closed connection
		client.OnClose(func(reason string) {
			fmt.Println("Client ", client.Id, " closed:", reason)
		})

	}
	<-never_die
}
```
