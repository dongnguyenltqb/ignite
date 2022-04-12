package ignite

import "encoding/json"

// Message Type
const (
	msgJoinRoom  = "joinRoom"
	msgLeaveRoom = "leaveRoom"
	msgIdentity  = "identity"
)

// Client/Server message format
type Message struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func ToPayload(value interface{}) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		getLogger().Error(err)
		return nil
	}
	return raw
}

type wsDirectMessage struct {
	c       *Client
	message []byte
}

type wsMessageForRoom struct {
	NodeId     string   `json:"nodeId"`
	RoomId     string   `json:"roomId"`
	Message    []byte   `json:"message"`
	ExcludeIds []string `json:"excludeIds"`
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
