package transport

import (
	"encoding/json"
	"mcp-bench/internal/protocol"
	"mcp-bench/internal/server"
	"net/http"
)

func RunHTTP(addr string) error {
	http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req protocol.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(protocol.ErrResponse(0, -32700, "parse error"))
			return
		}

		resp, shouldRespond := server.Handle(req)
		w.Header().Set("Content-Type", "application/json")
		if shouldRespond {
			json.NewEncoder(w).Encode(resp)
		}
	})

	return http.ListenAndServe(addr, nil)
}
