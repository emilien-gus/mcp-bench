package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mcp-bench/internal/protocol"
	"net/http"
	"time"
)

type Mode string

const (
	ModeCall    Mode = "call"    // мерим только tools/call
	ModeSession Mode = "session" // мерим полный сеанс initialize → tools/call × N
)

type Config struct {
	Tool      string
	N         int
	ServerURL string
	Mode      Mode
}

func Run(cfg Config) (*Result, error) {
	client := &http.Client{}

	if cfg.Mode == ModeSession {
		// в режиме сессии мерим всё целиком
		result := &Result{}
		start := time.Now()

		if err := handshake(client, cfg.ServerURL); err != nil {
			return nil, fmt.Errorf("handshake failed: %w", err)
		}
		for i := 0; i < cfg.N; i++ {
			if err := callTool(client, cfg.ServerURL, cfg.Tool, i); err != nil {
				result.Errors++
			}
		}

		result.Add(time.Since(start))
		return result, nil
	}

	// режим call — handshake отдельно, мерим только tools/call
	if err := handshake(client, cfg.ServerURL); err != nil {
		return nil, fmt.Errorf("handshake failed: %w", err)
	}

	result := &Result{}
	for i := 0; i < cfg.N; i++ {
		start := time.Now()
		err := callTool(client, cfg.ServerURL, cfg.Tool, i)
		elapsed := time.Since(start)

		if err != nil {
			result.Errors++
			continue
		}
		result.Add(elapsed)
	}

	return result, nil
}

func handshake(client *http.Client, url string) error {
	// 1. initialize
	params, _ := json.Marshal(protocol.InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      protocol.ClientInfo{Name: "mcp-bench", Version: "1.0.0"},
		Capabilities:    protocol.ClientCapabilities{},
	})
	_, err := doRequest(client, url, protocol.Request{
		JSONRPC: "2.0", ID: 0, Method: "initialize", Params: params,
	})
	if err != nil {
		return err
	}

	// 2. notifications/initialized
	notif, _ := json.Marshal(protocol.Request{
		JSONRPC: "2.0", Method: "notifications/initialized",
	})
	http.Post(url, "application/json", bytes.NewReader(notif))

	// 3. tools/list — теперь обязательная часть handshake
	_, err = doRequest(client, url, protocol.Request{
		JSONRPC: "2.0", ID: 1, Method: "tools/list",
	})
	return err
}

func callTool(client *http.Client, url, tool string, id int) error {
	params, _ := json.Marshal(protocol.ToolCallParams{
		Name:      tool,
		Arguments: json.RawMessage(`"hello"`),
	})
	resp, err := doRequest(client, url, protocol.Request{
		JSONRPC: "2.0", ID: id + 2, Method: "tools/call", Params: params,
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("rpc error: %s", resp.Error.Message)
	}
	return nil
}

func doRequest(client *http.Client, url string, req protocol.Request) (*protocol.Response, error) {
	body, _ := json.Marshal(req)
	httpResp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	var resp protocol.Response
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
