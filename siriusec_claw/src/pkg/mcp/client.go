package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/logging"
)

var mcpLog = logging.Sub("mcp")

// Client represents an MCP server client.
type Client struct {
	name       string
	transport  string
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.Reader
	httpClient *http.Client
	url        string
	mu         sync.Mutex
	tools      []Tool
	connected  bool
	command    string
	args       []string
	env        map[string]string
}

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolCallResult represents the result of a tool call.
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError"`
}

// ContentBlock represents content in a tool result.
type ContentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`
}

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewClient creates a new MCP client for the given server spec.
func NewClient(spec ServerSpec) *Client {
	return &Client{
		name:       spec.Name,
		transport:  spec.Transport,
		url:        spec.URL,
		command:    spec.Command,
		args:       spec.Args,
		env:        spec.Env,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Connect establishes connection to the MCP server.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.transport {
	case "stdio":
		return c.connectStdio(ctx)
	case "sse", "http":
		return c.connectHTTP(ctx)
	default:
		return fmt.Errorf("unsupported transport: %s", c.transport)
	}
}

// connectStdio starts a stdio-based MCP server.
func (c *Client) connectStdio(ctx context.Context) error {
	if c.command == "" {
		return fmt.Errorf("no command configured for MCP server: %s", c.name)
	}

	// Start the process
	c.cmd = exec.CommandContext(ctx, c.command, c.args...)
	if c.env != nil {
		c.cmd.Env = append(c.cmd.Env, os.Environ()...)
		for k, v := range c.env {
			c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	c.stdin = stdin

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	c.stdout = stdout

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Initialize the connection
	if err := c.initialize(ctx); err != nil {
		c.cmd.Process.Kill()
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	c.connected = true
	mcpLog.Info("connected to MCP server: %s (stdio)", c.name)
	return nil
}

// connectHTTP establishes connection to an HTTP/SSE-based MCP server.
func (c *Client) connectHTTP(ctx context.Context) error {
	// For HTTP transport, we just need to initialize
	if err := c.initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	c.connected = true
	mcpLog.Info("connected to MCP server: %s (%s)", c.name, c.transport)
	return nil
}

// initialize sends the initialize request to the MCP server.
func (c *Client) initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    "siriusec-claw",
			"version": "1.0.0",
		},
	}

	var result struct {
		ProtocolVersion string                 `json:"protocolVersion"`
		Capabilities    map[string]interface{} `json:"capabilities"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}

	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return err
	}

	mcpLog.Debug("initialized MCP server %s: protocol=%s server=%s/%s",
		c.name, result.ProtocolVersion, result.ServerInfo.Name, result.ServerInfo.Version)

	// Send initialized notification
	c.notify(ctx, "notifications/initialized", nil)

	// List available tools
	if err := c.listTools(ctx); err != nil {
		mcpLog.Warn("failed to list tools from MCP server %s: %v", c.name, err)
	}

	return nil
}

// listTools retrieves the list of tools from the MCP server.
func (c *Client) listTools(ctx context.Context) error {
	var result struct {
		Tools []Tool `json:"tools"`
	}

	if err := c.call(ctx, "tools/list", nil, &result); err != nil {
		return err
	}

	c.tools = result.Tools
	mcpLog.Info("MCP server %s has %d tools", c.name, len(c.tools))
	return nil
}

// Close closes the connection to the MCP server.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = false

	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	mcpLog.Info("closed MCP server: %s", c.name)
	return nil
}

// CallTool calls a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolCallResult, error) {
	c.mu.Lock()
	connected := c.connected
	c.mu.Unlock()

	if !connected {
		return nil, fmt.Errorf("MCP server %s not connected", c.name)
	}

	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	var result ToolCallResult
	if err := c.call(ctx, "tools/call", params, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListTools returns the list of available tools.
func (c *Client) ListTools() []Tool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tools
}

// IsConnected returns whether the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// GetName returns the server name.
func (c *Client) GetName() string {
	return c.name
}

// call makes a JSON-RPC call to the MCP server.
func (c *Client) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Method:  method,
		Params:  params,
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	if result != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("failed to parse result: %w", err)
		}
	}

	return nil
}

// notify sends a JSON-RPC notification to the MCP server.
func (c *Client) notify(ctx context.Context, method string, params interface{}) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	_, err := c.doRequest(ctx, req)
	return err
}

// doRequest sends a request and returns the response.
func (c *Client) doRequest(ctx context.Context, req JSONRPCRequest) (*JSONRPCResponse, error) {
	switch c.transport {
	case "stdio":
		return c.doStdioRequest(req)
	case "sse", "http":
		return c.doHTTPRequest(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported transport: %s", c.transport)
	}
}

// doStdioRequest sends a request via stdio.
func (c *Client) doStdioRequest(req JSONRPCRequest) (*JSONRPCResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Write request
	if _, err := fmt.Fprintf(c.stdin, "%s\n", data); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(c.stdout)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// doHTTPRequest sends a request via HTTP.
func (c *Client) doHTTPRequest(ctx context.Context, req JSONRPCRequest) (*JSONRPCResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.url
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var jsonResp JSONRPCResponse
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &jsonResp, nil
}

// Manager manages multiple MCP server connections.
type Manager struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

// NewManager creates a new MCP manager.
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
	}
}

// AddClient adds a client to the manager.
func (m *Manager) AddClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[client.name] = client
}

// GetClient returns a client by name.
func (m *Manager) GetClient(name string) *Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[name]
}

// ListClients returns all clients.
func (m *Manager) ListClients() []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	clients := make([]*Client, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	return clients
}

// ConnectAll connects all configured MCP servers.
func (m *Manager) ConnectAll(ctx context.Context, cfg *config.McpConfig) error {
	specs := BuildServerSpecs(cfg)
	for _, spec := range specs {
		client := NewClient(spec)
		if err := client.Connect(ctx); err != nil {
			mcpLog.Error("failed to connect MCP server %s: %v", spec.Name, err)
			continue
		}
		m.AddClient(client)
	}
	return nil
}

// CloseAll closes all MCP server connections.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, client := range m.clients {
		client.Close()
	}
	m.clients = make(map[string]*Client)
}

// ListAllTools returns all tools from all connected servers.
func (m *Manager) ListAllTools() map[string][]Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string][]Tool)
	for name, client := range m.clients {
		if client.IsConnected() {
			result[name] = client.ListTools()
		}
	}
	return result
}

// CallTool calls a tool on a specific server.
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (*ToolCallResult, error) {
	client := m.GetClient(serverName)
	if client == nil {
		return nil, fmt.Errorf("MCP server not found: %s", serverName)
	}
	return client.CallTool(ctx, toolName, args)
}
