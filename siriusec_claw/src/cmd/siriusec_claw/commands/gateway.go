package commands

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/gateway/handlers"
	gwhttp "github.com/siriusec/siriusec_claw/pkg/gateway/http"
	"github.com/siriusec/siriusec_claw/pkg/logging"
	"github.com/siriusec/siriusec_claw/pkg/paths"
	"github.com/siriusec/siriusec_claw/pkg/version"
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Gateway server management",
	Long:  "Manage the SiriuSec Claw gateway server (HTTP + WebSocket).",
}

var gatewayRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the gateway server",
	RunE:  runGateway,
}

var gatewayPortFlag int
var gatewayDebugFlag bool

func init() {
	gatewayRunCmd.Flags().IntVarP(&gatewayPortFlag, "port", "p", 0, "Gateway listen port (default 18900)")
	gatewayRunCmd.Flags().BoolVarP(&gatewayDebugFlag, "debug", "d", false, "Enable debug logging")
	gatewayCmd.AddCommand(gatewayRunCmd)
	rootCmd.AddCommand(gatewayCmd)
}

func runGateway(cmd *cobra.Command, args []string) error {
	env := config.DefaultEnv

	cfg, err := config.Load(env)
	if err != nil {
		slog.Warn("failed to load config, using defaults", "error", err)
		cfg = &config.ClawConfig{}
	}

	// Apply env vars from config
	if cfg.Env != nil && cfg.Env.Vars != nil {
		for k, v := range cfg.Env.Vars {
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}

	// Resolve port
	var cfgPort *int
	if cfg.Gateway != nil {
		cfgPort = cfg.Gateway.Port
	}
	port := paths.ResolveGatewayPort(cfgPort, env)
	if gatewayPortFlag > 0 {
		port = gatewayPortFlag
	}

	// Resolve run mode and address
	var gatewayMode *string
	if cfg.Gateway != nil {
		gatewayMode = cfg.Gateway.Mode
	}
	runMode := paths.ResolveRunMode(env, gatewayMode)
	addr := paths.ResolveGatewayAddr(port, runMode)

	// Initialize logging
	stateDir := paths.ResolveStateDir(env)
	logLevel := logging.LevelInfo
	if gatewayDebugFlag {
		logLevel = logging.LevelDebug
	}
	logging.Init(stateDir, logLevel)

	if gatewayDebugFlag {
		slog.Debug("Debug logging enabled")
	}

	fmt.Printf("SiriuSec Claw %s\n", version.Version)
	fmt.Printf("Gateway starting on %s (mode: %s)\n", addr, runMode)

	// Initialize channels from config
	handlers.InitChannelsFromConfig(cfg)
	slog.Info("channels initialized from config")

	server := gwhttp.NewServer(addr, version.Version, cfg)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("gateway server error", "error", err)
			os.Exit(1)
		}
	}()

	fmt.Printf("Gateway listening on %s\n", addr)

	<-sigCh
	fmt.Println("\nShutting down gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}
