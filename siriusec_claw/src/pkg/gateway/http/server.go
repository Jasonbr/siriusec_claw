// Package http provides the Gateway HTTP server skeleton.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	clawembed "github.com/siriusec/siriusec_claw/embed"
	"github.com/siriusec/siriusec_claw/pkg/agent/skills"
	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/gateway/handlers"
	"github.com/siriusec/siriusec_claw/pkg/gateway/protocol"
	"github.com/siriusec/siriusec_claw/pkg/gateway/webhooks"
	"github.com/siriusec/siriusec_claw/pkg/gateway/ws"
	"github.com/siriusec/siriusec_claw/pkg/logging"
	"github.com/siriusec/siriusec_claw/pkg/paths"
)

var serverLog = logging.Sub("http")

// Server is the Gateway HTTP server.
type Server struct {
	addr           string
	version        string
	server         *http.Server
	mux            *http.ServeMux
	hub            *ws.Hub
	ctx            *handlers.Context
	webhookHandler *webhooks.Handler
	distOnce       sync.Once
	distDir        string
	distErr        error
	distFS         fs.FS
}

// NewServer creates a new HTTP server with WebSocket hub and handlers.
func NewServer(addr string, version string, cfg *config.ClawConfig) *Server {
	mux := http.NewServeMux()
	env := config.DefaultEnv

	hub := ws.NewHub(version)

	if cfg == nil {
		cfg = &config.ClawConfig{}
	}

	ctx := &handlers.Context{
		Version: version,
		Config:  cfg,
		LoadConfigSnapshot: func() (*handlers.ConfigSnapshot, error) {
			return handlers.LoadConfigSnapshot(env)
		},
		AgentRunSeq: make(map[string]int64),
		Broadcast: func(event string, payload interface{}, opts *handlers.BroadcastOptions) {
			hub.Broadcast(event, payload, &ws.BroadcastOptions{
				DropIfSlow: opts != nil && opts.DropIfSlow,
			})
		},
		BroadcastToConnIds: func(event string, payload interface{}, connIds map[string]bool, opts *handlers.BroadcastOptions) {
			hub.BroadcastToConnIds(event, payload, connIds, &ws.BroadcastOptions{
				DropIfSlow: opts != nil && opts.DropIfSlow,
			})
		},
	}

	hub.SetContext(ctx)
	reg := handlers.NewRegistry(ctx)
	hub.SetHandlers(&reg)

	// Allow agent tools to synchronously invoke gateway methods
	ctx.InvokeMethod = func(method string, params map[string]interface{}) (bool, interface{}, *protocol.ErrorShape) {
		var resultOk bool
		var resultPayload interface{}
		var resultErr *protocol.ErrorShape
		done := make(chan struct{})
		opts := handlers.HandlerOpts{
			Req:     protocol.RequestFrame{Method: method, Params: params},
			Params:  params,
			Context: ctx,
			Respond: func(o bool, p interface{}, e *protocol.ErrorShape, _ map[string]interface{}) {
				resultOk, resultPayload, resultErr = o, p, e
				close(done)
			},
		}
		reg.Dispatch(opts)
		<-done
		return resultOk, resultPayload, resultErr
	}

	go hub.Run()

	// Initialize webhook handler
	channelManager := handlers.GetChannelManager()
	whHandler := webhooks.NewHandler(channelManager, nil, nil)

	s := &Server{
		addr:           addr,
		version:        version,
		mux:            mux,
		hub:            hub,
		ctx:            ctx,
		webhookHandler: whHandler,
	}
	s.registerRoutes()
	return s
}

func isAPIPath(p string) bool {
	return strings.HasPrefix(p, "/_ready") ||
		strings.HasPrefix(p, "/api/") ||
		p == "/ws" || strings.HasPrefix(p, "/ws/") || strings.HasPrefix(p, "/ws?") ||
		strings.HasPrefix(p, "/health") ||
		strings.HasPrefix(p, "/hooks") ||
		strings.HasPrefix(p, "/debug/") ||
		strings.HasPrefix(p, "/skills/")
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Root path WebSocket upgrade
		if r.URL.Path == "/" && r.Method == "GET" && strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			s.handleWSUpgrade(w, r)
			return
		}
		// Non-API GET/HEAD paths go to frontend
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) && !isAPIPath(r.URL.Path) {
			s.handleDist(w, r)
			return
		}
		s.mux.ServeHTTP(w, r)
	})
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /_ready", s.handleReady)
	s.mux.Handle("GET /", http.HandlerFunc(s.handleDist))
	s.mux.HandleFunc("GET /health", s.requireGatewayToken(s.handleHealth))
	s.mux.HandleFunc("GET /api/health", s.requireGatewayToken(s.handleHealth))
	s.mux.HandleFunc("GET /api/config", s.requireGatewayToken(s.handleConfigGet))
	s.mux.HandleFunc("GET /api/config/env", s.requireGatewayToken(s.handleConfigEnv))
	s.mux.HandleFunc("POST /api/config/patch", s.requireGatewayToken(s.handleConfigPatch))
	s.mux.HandleFunc("PATCH /api/config/patch", s.requireGatewayToken(s.handleConfigPatch))
	s.mux.HandleFunc("POST /api/skills/upload", s.requireGatewayToken(s.handleSkillsUpload))
	s.mux.HandleFunc("GET /ws", s.handleWSUpgrade)
	s.mux.HandleFunc("POST /hooks/", s.handleHooks)
	s.mux.HandleFunc("POST /hooks", s.handleHooks)
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true,"version":"` + s.version + `"}`))
}

func (s *Server) handleConfigGet(w http.ResponseWriter, _ *http.Request) {
	snap, err := s.ctx.LoadConfigSnapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snap)
}

func (s *Server) handleConfigEnv(w http.ResponseWriter, _ *http.Request) {
	env := config.DefaultEnv
	stateDir := paths.ResolveStateDir(env)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stateDir":   stateDir,
		"configPath": paths.ResolveConfigPath(env, stateDir),
	})
}

func (s *Server) handleConfigPatch(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	patch, ok := body["patch"].(map[string]interface{})
	if !ok {
		http.Error(w, "patch field required", http.StatusBadRequest)
		return
	}

	env := config.DefaultEnv
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveConfigPath(env, stateDir)

	currentData, _ := os.ReadFile(configPath)
	var current map[string]interface{}
	if len(currentData) > 0 {
		_ = json.Unmarshal(currentData, &current)
	}
	if current == nil {
		current = make(map[string]interface{})
	}
	for k, v := range patch {
		current[k] = v
	}
	data, _ := json.MarshalIndent(current, "", "  ")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleWSUpgrade(w http.ResponseWriter, r *http.Request) {
	s.hub.ServeWS(w, r)
}

func (s *Server) handleHooks(w http.ResponseWriter, r *http.Request) {
	if s.webhookHandler != nil {
		s.webhookHandler.ServeHTTP(w, r)
		return
	}
	// Fallback if webhook handler is not initialized
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true,"message":"hooks received"}`))
}

// handleDist serves the static frontend.
func (s *Server) handleDist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}
	s.distOnce.Do(func() {
		if efs, err := clawembed.FrontendFS(); err == nil {
			if _, err := fs.Stat(efs, "index.html"); err == nil {
				s.distFS = efs
				return
			}
		}
		s.distDir, s.distErr = resolveDistDirFile()
	})
	if s.distErr != nil {
		http.Error(w, s.distErr.Error(), http.StatusInternalServerError)
		return
	}

	serveIndex := func() {
		var f fs.File
		var err error
		if s.distFS != nil {
			f, err = s.distFS.Open("index.html")
		} else {
			var of *os.File
			of, err = os.Open(filepath.Join(s.distDir, "index.html"))
			if of != nil {
				f = of
			}
		}
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var rs io.ReadSeeker
		if rsf, ok := f.(io.ReadSeeker); ok {
			rs = rsf
		} else {
			data, _ := io.ReadAll(f)
			rs = bytes.NewReader(data)
		}
		http.ServeContent(w, r, "index.html", info.ModTime(), rs)
	}

	cleanPath := path.Clean("/" + strings.TrimSpace(r.URL.Path))
	if cleanPath == "/" || cleanPath == "" {
		serveIndex()
		return
	}

	name := strings.TrimPrefix(cleanPath, "/")
	name = filepath.ToSlash(filepath.Clean(name))
	if name == "" || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}

	var f fs.File
	var err error
	if s.distFS != nil {
		f, err = s.distFS.Open(name)
	} else {
		var of *os.File
		of, err = os.Open(filepath.Join(s.distDir, name))
		if of != nil {
			f = of
		}
	}
	if err != nil {
		accept := strings.ToLower(r.Header.Get("Accept"))
		if strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*") {
			serveIndex()
			return
		}
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	var rs io.ReadSeeker
	if rsf, ok := f.(io.ReadSeeker); ok {
		rs = rsf
	} else {
		data, _ := io.ReadAll(f)
		rs = bytes.NewReader(data)
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), rs)
}

func resolveDistDirFile() (string, error) {
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "dist", "control-ui"),
		filepath.Join(cwd, "embed", "frontend"),
		filepath.Join(cwd, "src", "embed", "frontend"),
	}
	if env := strings.TrimSpace(os.Getenv("SIRIUSEC_CLAW_FRONTEND_DIR")); env != "" {
		candidates = append([]string{filepath.Clean(env)}, candidates...)
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
			return filepath.Clean(dir), nil
		}
	}
	return "", nil // No frontend available, embedded FS will be used
}

// handleSkillsUpload handles multipart file upload for skill installation.
func (s *Server) handleSkillsUpload(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file name
	filename := strings.ToLower(strings.TrimSpace(header.Filename))
	if !strings.HasSuffix(filename, ".md") && !strings.HasSuffix(filename, ".zip") {
		http.Error(w, `{"error":"file must be .md or .zip"}`, http.StatusBadRequest)
		return
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, `{"error":"failed to read file"}`, http.StatusInternalServerError)
		return
	}

	// Get optional name parameter
	name := strings.TrimSpace(r.FormValue("name"))

	env := os.Getenv
	var result *skills.InstallResult

	if strings.HasSuffix(filename, ".zip") {
		// For zip files, extract and install
		http.Error(w, `{"error":"zip upload not yet implemented, use github source"}`, http.StatusNotImplemented)
		return
	}

	// Install from markdown content
	result, err = skills.InstallFromUpload(name, content, env)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"name":     result.Name,
		"skillDir": result.SkillDir,
		"source":   result.Source,
		"metadata": result.Metadata,
	})
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      s.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
