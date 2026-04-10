package protocol

// RequestFrame is a client -> server request.
type RequestFrame struct {
	Type   string      `json:"type"` // "req"
	ID     string      `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// ResponseFrame is a server -> client response.
type ResponseFrame struct {
	Type    string      `json:"type"` // "res"
	ID      string      `json:"id"`
	OK      bool        `json:"ok"`
	Payload interface{} `json:"payload,omitempty"`
	Error   *ErrorShape `json:"error,omitempty"`
}

// EventFrame is a server -> client event.
type EventFrame struct {
	Type         string        `json:"type"` // "event"
	Event        string        `json:"event"`
	Payload      interface{}   `json:"payload,omitempty"`
	Seq          *int64        `json:"seq,omitempty"`
	StateVersion *StateVersion `json:"stateVersion,omitempty"`
}

// ErrorShape is the canonical error payload.
type ErrorShape struct {
	Code         string      `json:"code"`
	Message      string      `json:"message"`
	Details      interface{} `json:"details,omitempty"`
	Retryable    *bool       `json:"retryable,omitempty"`
	RetryAfterMs *int64      `json:"retryAfterMs,omitempty"`
}

// StateVersion tracks presence/health version.
type StateVersion struct {
	Presence int `json:"presence"`
	Health   int `json:"health"`
}

// ConnectParams is the client handshake payload.
type ConnectParams struct {
	MinProtocol int               `json:"minProtocol"`
	MaxProtocol int               `json:"maxProtocol"`
	Client      ConnectClientInfo `json:"client"`
	Caps        []string          `json:"caps,omitempty"`
	Commands    []string          `json:"commands,omitempty"`
	Permissions map[string]bool   `json:"permissions,omitempty"`
	Role        string            `json:"role,omitempty"`
	Scopes      []string          `json:"scopes,omitempty"`
	Auth        *ConnectAuth      `json:"auth,omitempty"`
	Locale      string            `json:"locale,omitempty"`
	UserAgent   string            `json:"userAgent,omitempty"`
}

// ConnectClientInfo identifies the client.
type ConnectClientInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName,omitempty"`
	Version     string `json:"version"`
	Platform    string `json:"platform"`
	Mode        string `json:"mode"`
	InstanceID  string `json:"instanceId,omitempty"`
}

// ConnectAuth is token/password auth.
type ConnectAuth struct {
	Token    string `json:"token,omitempty"`
	Password string `json:"password,omitempty"`
}

// HelloOk is the server handshake response.
type HelloOk struct {
	Type     string        `json:"type"` // "hello-ok"
	Protocol int           `json:"protocol"`
	Server   HelloServer   `json:"server"`
	Features HelloFeatures `json:"features"`
	Snapshot Snapshot      `json:"snapshot"`
	Policy   HelloPolicy   `json:"policy"`
}

// HelloServer identifies the server.
type HelloServer struct {
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
	Host    string `json:"host,omitempty"`
	ConnID  string `json:"connId"`
}

// HelloFeatures lists supported methods and events.
type HelloFeatures struct {
	Methods []string `json:"methods"`
	Events  []string `json:"events"`
}

// HelloPolicy defines connection limits.
type HelloPolicy struct {
	MaxPayload       int `json:"maxPayload"`
	MaxBufferedBytes int `json:"maxBufferedBytes"`
	TickIntervalMs   int `json:"tickIntervalMs"`
}

// Snapshot is the initial state sent in hello-ok.
type Snapshot struct {
	Presence     []PresenceEntry `json:"presence"`
	Health       interface{}     `json:"health"`
	StateVersion StateVersion    `json:"stateVersion"`
	UptimeMs     int64           `json:"uptimeMs"`
}

// PresenceEntry is a connected client entry.
type PresenceEntry struct {
	Host     string `json:"host,omitempty"`
	Version  string `json:"version,omitempty"`
	Platform string `json:"platform,omitempty"`
	Mode     string `json:"mode,omitempty"`
	Ts       int64  `json:"ts"`
}
