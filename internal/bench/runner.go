package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mcp-bench/internal/protocol"
	"net/http"
	"sync"
	"time"
)

const warmupN = 10

type Mode string

const (
	ModeCall    Mode = "call"
	ModeSession Mode = "session"
)

type Config struct {
	Tool        string
	N           int
	ServerURL   string
	Mode        Mode
	Concurrency int
}

func Run(cfg Config) (*Result, error) {
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}

	if cfg.Mode == ModeSession {
		client := &http.Client{}
		result := &Result{}
		start := time.Now()
		if err := handshake(client, cfg.ServerURL); err != nil {
			return nil, fmt.Errorf("handshake failed: %w", err)
		}
		for i := 0; i < cfg.N; i++ {
			if err := callTool(client, cfg.ServerURL, cfg.Tool, i); err != nil {
				result.AddError()
			}
		}
		result.Add(time.Since(start))
		return result, nil
	}

	// sequential
	if cfg.Concurrency == 1 {
		client := &http.Client{}
		if err := handshake(client, cfg.ServerURL); err != nil {
			return nil, fmt.Errorf("handshake failed: %w", err)
		}
		for i := 0; i < warmupN; i++ {
			callTool(client, cfg.ServerURL, cfg.Tool, -i)
		}
		result := &Result{}
		for i := 0; i < cfg.N; i++ {
			start := time.Now()
			err := callTool(client, cfg.ServerURL, cfg.Tool, i)
			elapsed := time.Since(start)
			if err != nil {
				result.AddError()
				continue
			}
			result.Add(elapsed)
		}
		return result, nil
	}

	// concurrent — каждая горутина свой http.Client и handshake
	perGoroutine := cfg.N / cfg.Concurrency
	result := &Result{}
	var wg sync.WaitGroup

	for g := 0; g < cfg.Concurrency; g++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			client := &http.Client{}
			if err := handshake(client, cfg.ServerURL); err != nil {
				return
			}
			for i := 0; i < warmupN; i++ {
				callTool(client, cfg.ServerURL, cfg.Tool, -i)
			}
			for i := 0; i < perGoroutine; i++ {
				start := time.Now()
				err := callTool(client, cfg.ServerURL, cfg.Tool, gID*perGoroutine+i)
				elapsed := time.Since(start)
				if err != nil {
					result.AddError()
					continue
				}
				result.Add(elapsed)
			}
		}(g)
	}

	wg.Wait()
	return result, nil
}

func handshake(client *http.Client, url string) error {
	params, _ := json.Marshal(protocol.InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      protocol.ClientInfo{Name: "mcp-bench", Version: "1.0.0"},
		Capabilities:    protocol.ClientCapabilities{},
	})
	if _, err := doRequest(client, url, protocol.Request{
		JSONRPC: "2.0", ID: 0, Method: "initialize", Params: params,
	}); err != nil {
		return err
	}
	notif, _ := json.Marshal(protocol.Request{JSONRPC: "2.0", Method: "notifications/initialized"})
	http.Post(url, "application/json", bytes.NewReader(notif))
	if _, err := doRequest(client, url, protocol.Request{
		JSONRPC: "2.0", ID: 1, Method: "tools/list",
	}); err != nil {
		return err
	}
	return nil
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
