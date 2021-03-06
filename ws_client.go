package ignite

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 1024

	// Max room size
	maxRoomSize = 100
)

// Client handle function for each event
// Each function include function id, to easy to remove
type clientHandleFunc struct {
	event string
	id    string
	f     func(json.RawMessage)
}

// Client is a middleman between the websocket connection and the hub.
type Client[K comparable, V any] struct {
	hub *Hub[K, V]

	// Identity
	ID string

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Room channel event
	rchan chan wsRoomActionMessage

	// The list of room which client is joining
	rooms []string

	// Logger
	logger *logrus.Logger

	// On close handler
	onCloseHandelFuncs []func(string)

	// Handle function
	handleFuncs []clientHandleFunc

	// Client metadata
	metadata map[K]V
}

// set a value to client metadata
func (c *Client[K, V]) Set(key K, value V) {
	c.metadata[key] = value
}

// retrieve value by given key from client metadata
func (c *Client[K, V]) Get(key K) V {
	return c.metadata[key]
}

// marshall message to json.RawMessage
func (c *Client[K, V]) RawMessage(payload interface{}) []byte {
	bytes, _ := json.Marshal(payload)
	return bytes
}

// remove every message and shutdown after 1 minutes
func (c *Client[K, V]) clean() {
	c.rooms = nil
	close(c.send)
	close(c.rchan)
}

// roomPump pumps action for channel and process one by one
func (c *Client[K, V]) roomPump() {
	for msg := range c.rchan {
		if msg.Join {
			for i := 0; i < len(msg.Ids); i++ {
				id := msg.Ids[i]
				if !containtString(c.rooms, id) {
					c.rooms = append(c.rooms, id)
				}
			}
		}
		if msg.Leave {
			for i := 0; i < len(msg.Ids); i++ {
				id := msg.Ids[i]
				n := len(c.rooms)
				r := make([]string, 0)
				for i := 0; i < n; i++ {
					if c.rooms[i] != id {
						r = append(r, c.rooms[i])
					}
				}
				c.rooms = r
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client[K, V]) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error(err)
				if len(c.onCloseHandelFuncs) > 0 {
					for _, handler := range c.onCloseHandelFuncs {
						handler(err.Error())
					}
				}
			}
			break
		}
		go c.processMsg(message)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client[K, V]) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Join a room
func (c *Client[K, V]) Join(roomId string) {
	rMsg := wsRoomActionMessage{
		Join: true,
		Ids:  []string{roomId},
	}
	c.rchan <- rMsg
}

// Leave a room
func (c *Client[K, V]) Leave(roomId string) {
	rMsg := wsRoomActionMessage{
		Leave: true,
		Ids:   []string{roomId},
	}
	c.rchan <- rMsg
}

// Check if client join this room or not
func (c *Client[K, V]) exist(roomId string) bool {
	return containtString(c.rooms, roomId)
}

// Send a raw message to client
func (c *Client[K, V]) sendRawMsg(message []byte) {
	c.hub.directMsg <- wsDirectMessage[K, V]{
		c:       c,
		message: message,
	}
}

// Send direct message to client
func (c *Client[K, V]) SendMsg(message Message) {
	b, _ := json.Marshal(message)
	c.hub.directMsg <- wsDirectMessage[K, V]{
		c:       c,
		message: b,
	}
}

// Send message to specific room
func (c *Client[K, V]) SendMsgToRoom(roomId string, message Message) {
	c.hub.SendMsgToRoom(roomId, message)
}

// Send message to specific room except some client
func (c *Client[K, V]) SendMsgExcept(roomId string, exclude_ids []string, message Message) {
	c.hub.SendMsgExcept(roomId, exclude_ids, message)
}

// Send message to all active connection
func (c *Client[K, V]) BroadcastMsg(msg Message) {
	c.hub.BroadcastMsg(msg)
}

// Send id to client
func (c *Client[K, V]) SendId() {
	clientId := wsIdentityMessage{
		ClientId: c.ID,
	}
	b, _ := json.Marshal(clientId)
	msg := Message{
		Event:   msgIdentity,
		Payload: b,
	}
	b, _ = json.Marshal(msg)
	go c.sendRawMsg(b)
}

// Register handle func for an event
func (c *Client[K, V]) On(event string, funcId string, f func(json.RawMessage)) {
	c.handleFuncs = append(c.handleFuncs, clientHandleFunc{
		event: event,
		id:    funcId,
		f:     f,
	})
}

// Unregister handle func for an event
func (c *Client[K, V]) Off(event string, funcId string) {
	newHandleFuncs := []clientHandleFunc{}
	for _, handler := range c.handleFuncs {
		if handler.id != funcId && event != handler.event {
			newHandleFuncs = append(newHandleFuncs, handler)
		}
	}
	c.handleFuncs = newHandleFuncs
}

// Callback function when a client closed connection
func (c *Client[K, V]) OnClose(f func(string)) {
	c.onCloseHandelFuncs = append(c.onCloseHandelFuncs, f)
}

// process message from readPump
func (c *Client[K, V]) processMsg(message []byte) {
	msg := Message{}
	if err := json.Unmarshal(message, &msg); err != nil {
		c.logger.Error(err)
		return
	}
	// Handle chat message, broadcast room
	for _, handler := range c.handleFuncs {
		if handler.event == msg.Event {
			handler.f(msg.Payload)
		}
	}
}
