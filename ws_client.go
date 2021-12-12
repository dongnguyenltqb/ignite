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

type clientHandleFunc struct {
	event string
	id    string
	f     func(json.RawMessage)
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// Identity
	Id string

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
}

// remove every message and shutdown after 1 minutes
func (c *Client) clean() {
	c.rooms = nil
	close(c.send)
	close(c.rchan)
}

// roomPump pumps action for channel and process one by one
func (c *Client) roomPump() {
	for msg := range c.rchan {
		if msg.Join {
			for i := 0; i < len(msg.Ids); i++ {
				id := msg.Ids[i]
				n := len(c.rooms)
				joined := false
				for i := 0; i < n; i++ {
					if c.rooms[i] == id {
						joined = true
					}
				}
				if !joined {
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
func (c *Client) readPump() {
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
func (c *Client) writePump() {
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
func (c *Client) Join(roomId string) {
	rMsg := wsRoomActionMessage{
		Join: true,
		Ids:  []string{roomId},
	}
	c.rchan <- rMsg
}

func (c *Client) Leave(roomId string) {
	rMsg := wsRoomActionMessage{
		Leave: true,
		Ids:   []string{roomId},
	}
	c.rchan <- rMsg
}

func (c *Client) exist(roomId string) bool {
	n := len(c.rooms)
	for i := 0; i < n; i++ {
		if c.rooms[i] == roomId {
			return true
		}
	}
	return false
}

func (c *Client) sendRawMsg(message []byte) {
	c.hub.directMsg <- wsDirectMessage{
		c:       c,
		message: message,
	}
}

// Send direct message to client
func (c *Client) SendMessage(message Message) {
	b, _ := json.Marshal(message)
	c.hub.directMsg <- wsDirectMessage{
		c:       c,
		message: b,
	}
}

// Send message to specific room
func (c *Client) SendMsgToRoom(roomId string, message Message) {
	c.hub.SendMsgToRoom(roomId, message)
}

// Send message to all active connection
func (c *Client) BroadcastMsg(msg Message) {
	c.hub.BroadcastMsg(msg)
}

func (c *Client) SendIdentityMsg() {
	clientId := wsIdentityMessage{
		ClientId: c.Id,
	}
	b, _ := json.Marshal(clientId)
	msg := Message{
		Event:   msgIdentity,
		Payload: b,
	}
	b, _ = json.Marshal(msg)
	go c.sendRawMsg(b)
}

// register handle func
func (c *Client) On(event string, funcId string, f func(json.RawMessage)) {
	c.handleFuncs = append(c.handleFuncs, clientHandleFunc{
		event: event,
		id:    funcId,
		f:     f,
	})
}
func (c *Client) Off(event string, funcId string) {
	newHandleFuncs := []clientHandleFunc{}
	for _, handler := range c.handleFuncs {
		if handler.id != funcId && event != handler.event {
			newHandleFuncs = append(newHandleFuncs, handler)
		}
	}
	c.handleFuncs = newHandleFuncs
}
func (c *Client) OnClose(f func(string)) {
	c.onCloseHandelFuncs = append(c.onCloseHandelFuncs, f)
}

// process message from readPump
func (c *Client) processMsg(message []byte) {
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
