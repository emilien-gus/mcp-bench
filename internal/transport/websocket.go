package transport

import (
	"encoding/json"
	"mcp-bench/internal/protocol"
	"mcp-bench/internal/server"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  32 * 1024 * 1024,
	WriteBufferSize: 32 * 1024 * 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func RunWebSocket(addr string) error {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var req protocol.Request
			if err := json.Unmarshal(msg, &req); err != nil {
				resp := protocol.ErrResponse(0, -32700, "parse error")
				data, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, data)
				continue
			}

			resp, shouldRespond := server.Handle(req)
			if shouldRespond {
				data, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, data)
			}
		}
	})

	return http.ListenAndServe(addr, nil)
}
