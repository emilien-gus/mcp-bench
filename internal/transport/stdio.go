package transport

import (
	"bufio"
	"encoding/json"
	"mcp-bench/internal/protocol"
	"mcp-bench/internal/server"
	"os"
)

func RunStdio() error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // буфер 1МБ для heavy
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		var req protocol.Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			encoder.Encode(protocol.ErrResponse(0, -32700, "parse error"))
			continue
		}
		resp, shouldRespond := server.Handle(req)
		if shouldRespond {
			encoder.Encode(resp)
		}
	}
	return scanner.Err()
}
