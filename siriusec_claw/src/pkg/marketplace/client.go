package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultMarketplaceURL is the default OpenOcta marketplace URL.
const DefaultMarketplaceURL = "https://openocta.ai"

// Client is a marketplace API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string // Optional auth token
}

// NewClient creates a new marketplace client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultMarketplaceURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken sets the authentication token for the client.
func (c *Client) SetToken(token string) {
	c.token = token
}

// === Skills ===

// SkillItem represents a skill in the marketplace.
type SkillItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Emoji       string   `json:"emoji,omitempty"`
	Description string   `json:"description"`
	Author      string   `json:"author,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Version     string   `json:"version,omitempty"`
	Downloads   int      `json:"downloads,omitempty"`
	Rating      float64  `json:"rating,omitempty"`
	Installed   bool     `json:"installed,omitempty"`
}

// SkillDetail represents detailed skill information including content.
type SkillDetail struct {
	SkillItem
	Content   string                 `json:"content,omitempty"`   // SKILL.md content
	Config    map[string]interface{} `json:"config,omitempty"`    // Default config
	Variables []ConfigVariable       `json:"variables,omitempty"` // Config variables
}

// ConfigVariable represents a configuration variable for a skill.
type ConfigVariable struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // text, password, number, boolean, select
	Required    bool     `json:"required"`
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
	Description string   `json:"description,omitempty"`
}

// SkillsResponse is the response from listing skills.
type SkillsResponse struct {
	Skills []SkillItem `json:"skills"`
	Total  int         `json:"total"`
}

// ListSkills fetches the list of skills from the marketplace.
func (c *Client) ListSkills(ctx context.Context, query string) (*SkillsResponse, error) {
	u, _ := url.Parse(c.baseURL + "/api/skills")
	if query != "" {
		q := u.Query()
		q.Set("q", query)
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var result SkillsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSkill fetches a single skill's details.
func (c *Client) GetSkill(ctx context.Context, id string) (*SkillDetail, error) {
	u := c.baseURL + "/api/skills/" + id

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("skill not found: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var result SkillDetail
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// === MCPs ===

// MCPItem represents an MCP server in the marketplace.
type MCPItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Emoji       string   `json:"emoji,omitempty"`
	Description string   `json:"description"`
	Author      string   `json:"author,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Version     string   `json:"version,omitempty"`
	Downloads   int      `json:"downloads,omitempty"`
	Rating      float64  `json:"rating,omitempty"`
	Transport   string   `json:"transport,omitempty"` // stdio, sse, http
	Installed   bool     `json:"installed,omitempty"`
}

// MCPDetail represents detailed MCP server information.
type MCPDetail struct {
	MCPItem
	Command    string                 `json:"command,omitempty"`
	Args       []string               `json:"args,omitempty"`
	URL        string                 `json:"url,omitempty"`
	Env        map[string]string      `json:"env,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
	Variables  []ConfigVariable       `json:"variables,omitempty"`
	ToolPrefix string                 `json:"toolPrefix,omitempty"`
}

// MCPsResponse is the response from listing MCPs.
type MCPsResponse struct {
	MCPServers []MCPItem `json:"mcpServers"`
	Total      int       `json:"total"`
}

// ListMCPs fetches the list of MCP servers from the marketplace.
func (c *Client) ListMCPs(ctx context.Context, query string) (*MCPsResponse, error) {
	u, _ := url.Parse(c.baseURL + "/api/mcps")
	if query != "" {
		q := u.Query()
		q.Set("q", query)
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var result MCPsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetMCP fetches a single MCP server's details.
func (c *Client) GetMCP(ctx context.Context, id string) (*MCPDetail, error) {
	u := c.baseURL + "/api/mcps/" + id

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("MCP server not found: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var result MCPDetail
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// === Employees ===

// EmployeeItem represents an employee/agent in the marketplace.
type EmployeeItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Emoji       string   `json:"emoji,omitempty"`
	Description string   `json:"description"`
	Author      string   `json:"author,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Version     string   `json:"version,omitempty"`
	Downloads   int      `json:"downloads,omitempty"`
	Rating      float64  `json:"rating,omitempty"`
	Installed   bool     `json:"installed,omitempty"`
}

// EmployeeDetail represents detailed employee information.
type EmployeeDetail struct {
	EmployeeItem
	Prompt     string                 `json:"prompt,omitempty"`
	SkillIDs   []string               `json:"skillIds,omitempty"`
	McpServers map[string]interface{} `json:"mcpServers,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
	Variables  []ConfigVariable       `json:"variables,omitempty"`
}

// EmployeesResponse is the response from listing employees.
type EmployeesResponse struct {
	Employees []EmployeeItem `json:"employees"`
	Total     int            `json:"total"`
}

// ListEmployees fetches the list of employees from the marketplace.
func (c *Client) ListEmployees(ctx context.Context, query string) (*EmployeesResponse, error) {
	u, _ := url.Parse(c.baseURL + "/api/employees")
	if query != "" {
		q := u.Query()
		q.Set("q", query)
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var result EmployeesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetEmployee fetches a single employee's details.
func (c *Client) GetEmployee(ctx context.Context, id string) (*EmployeeDetail, error) {
	u := c.baseURL + "/api/employees/" + id

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("employee not found: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var result EmployeeDetail
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// === Helper methods ===

func (c *Client) setAuth(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/json")
}

// FetchContent fetches raw content from a URL (for downloading SKILL.md, etc.)
func (c *Client) FetchContent(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch content: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
