package bench

import (
	"encoding/json"
	"fmt"
	"mcp-bench/internal/protocol"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	conn    *websocket.Conn
	counter int
	mu      sync.Mutex
}

func NewWSClient(url string) (*WSClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &WSClient{conn: conn}, nil
}

func (c *WSClient) Send(req protocol.Request) (*protocol.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return nil, err
	}
	if req.Method == "notifications/initialized" {
		return nil, nil
	}
	_, msg, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	var resp protocol.Response
	if err := json.Unmarshal(msg, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *WSClient) Handshake() error {
	params, _ := json.Marshal(protocol.InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      protocol.ClientInfo{Name: "mcp-bench", Version: "1.0.0"},
		Capabilities:    protocol.ClientCapabilities{},
	})
	if _, err := c.Send(protocol.Request{
		JSONRPC: "2.0", ID: 0, Method: "initialize", Params: params,
	}); err != nil {
		return err
	}
	c.Send(protocol.Request{JSONRPC: "2.0", Method: "notifications/initialized"})
	if _, err := c.Send(protocol.Request{
		JSONRPC: "2.0", ID: 1, Method: "tools/list",
	}); err != nil {
		return err
	}
	c.counter = 1
	return nil
}

func (c *WSClient) CallTool(tool string) error {
	c.mu.Lock()
	c.counter++
	id := c.counter
	c.mu.Unlock()

	params, _ := json.Marshal(protocol.ToolCallParams{
		Name:      tool,
		Arguments: json.RawMessage(`"hello"`),
	})
	resp, err := c.Send(protocol.Request{
		JSONRPC: "2.0", ID: id, Method: "tools/call", Params: params,
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("rpc error: %s", resp.Error.Message)
	}
	return nil
}

func (c *WSClient) Close() {
	c.conn.Close()
}

func RunWSBench(serverURL, tool string, n int, mode Mode, concurrency int) (*Result, error) {
	if concurrency < 1 {
		concurrency = 1
	}

	if mode == ModeSession {
		client, err := NewWSClient(serverURL)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		if err := client.Handshake(); err != nil {
			return nil, err
		}
		result := &Result{}
		start := time.Now()
		for i := 0; i < n; i++ {
			if err := client.CallTool(tool); err != nil {
				result.AddError()
			}
		}
		result.Add(time.Since(start))
		return result, nil
	}

	if concurrency == 1 {
		client, err := NewWSClient(serverURL)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		if err := client.Handshake(); err != nil {
			return nil, err
		}
		// warmup
		for i := 0; i < warmupN; i++ {
			client.CallTool(tool)
		}
		result := &Result{}
		for i := 0; i < n; i++ {
			start := time.Now()
			err := client.CallTool(tool)
			elapsed := time.Since(start)
			if err != nil {
				result.AddError()
				continue
			}
			result.Add(elapsed)
		}
		return result, nil
	}

	// concurrent — один сервер, несколько клиентов
	// каждая горутина своё WS-соединение
	perGoroutine := n / concurrency
	result := &Result{}
	var wg sync.WaitGroup

	for g := 0; g < concurrency; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := NewWSClient(serverURL)
			if err != nil {
				return
			}
			defer client.Close()
			if err := client.Handshake(); err != nil {
				return
			}
			for i := 0; i < warmupN; i++ {
				client.CallTool(tool)
			}
			for i := 0; i < perGoroutine; i++ {
				start := time.Now()
				err := client.CallTool(tool)
				elapsed := time.Since(start)
				if err != nil {
					result.AddError()
					continue
				}
				result.Add(elapsed)
			}
		}()
	}

	wg.Wait()
	return result, nil
}
