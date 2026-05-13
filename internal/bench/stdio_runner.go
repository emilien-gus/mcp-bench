package bench

import (
	"bufio"
	"encoding/json"
	"fmt"
	"mcp-bench/internal/protocol"
	"os/exec"
	"time"
)

type StdioClient struct {
	cmd     *exec.Cmd
	encoder *json.Encoder
	scanner *bufio.Scanner
	counter int
}

func NewStdioClient(serverBin string) (*StdioClient, error) {
	cmd := exec.Command(serverBin, "--transport", "stdio")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	return &StdioClient{
		cmd:     cmd,
		encoder: json.NewEncoder(stdin),
		scanner: scanner,
	}, nil
}

func (c *StdioClient) Send(req protocol.Request) (*protocol.Response, error) {
	if err := c.encoder.Encode(req); err != nil {
		return nil, err
	}
	if req.Method == "notifications/initialized" {
		return nil, nil
	}
	if !c.scanner.Scan() {
		return nil, fmt.Errorf("server closed stdout")
	}
	var resp protocol.Response
	if err := json.Unmarshal(c.scanner.Bytes(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *StdioClient) Handshake() error {
	// 1. initialize
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

	// 2. notifications/initialized
	c.Send(protocol.Request{
		JSONRPC: "2.0", Method: "notifications/initialized",
	})

	// 3. tools/list
	if _, err := c.Send(protocol.Request{
		JSONRPC: "2.0", ID: 1, Method: "tools/list",
	}); err != nil {
		return err
	}

	c.counter = 1
	return nil
}

func (c *StdioClient) CallTool(tool string) error {
	c.counter++
	params, _ := json.Marshal(protocol.ToolCallParams{
		Name:      tool,
		Arguments: json.RawMessage(`"hello"`),
	})
	resp, err := c.Send(protocol.Request{
		JSONRPC: "2.0", ID: c.counter, Method: "tools/call", Params: params,
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("rpc error: %s", resp.Error.Message)
	}
	return nil
}

func (c *StdioClient) Close() {
	c.cmd.Process.Kill()
	c.cmd.Wait()
}

func RunStdioBench(serverBin, tool string, n int, mode Mode) (*Result, error) {
	if mode == ModeSession {
		result := &Result{}
		start := time.Now()

		client, err := NewStdioClient(serverBin)
		if err != nil {
			return nil, err
		}
		defer client.Close()

		if err := client.Handshake(); err != nil {
			return nil, err
		}
		for i := 0; i < n; i++ {
			if err := client.CallTool(tool); err != nil {
				result.Errors++
			}
		}

		result.Add(time.Since(start))
		return result, nil
	}

	// режим call
	client, err := NewStdioClient(serverBin)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.Handshake(); err != nil {
		return nil, err
	}

	result := &Result{}
	for i := 0; i < n; i++ {
		start := time.Now()
		err := client.CallTool(tool)
		elapsed := time.Since(start)

		if err != nil {
			result.Errors++
			continue
		}
		result.Add(elapsed)
	}

	return result, nil
}
