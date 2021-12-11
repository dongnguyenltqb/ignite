package ignite

import "encoding/json"

const (
	// Message Type
	msgJoinRoom  = "joinRoom"
	msgLeaveRoom = "leaveRoom"
	msgIdentity  = "identity"
)

type Message struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

type wsDirectMessage struct {
	c       *Client
	message []byte
}

type wsMessageForRoom struct {
	NodeId  string `json:"nodeId"`
	RoomId  string `json:"roomId"`
	Message []byte `json:"message"`
}

type wsBroadcastMessage struct {
	NodeId  string `json:"nodeId"`
	Message []byte `json:"message"`
}

type wsIdentityMessage struct {
	ClientId string `json:"clientId"`
}

type wsRoomActionMessage struct {
	Leave    bool     `json:"leave"`
	Join     bool     `json:"join"`
	Ids      []string `json:"ids"`
	MemberId string   `json:"memberId"`
}
