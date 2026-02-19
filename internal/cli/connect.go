package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"raycoon/internal/core"
	"raycoon/internal/core/tun"
	"raycoon/internal/core/types"
	"raycoon/internal/latency"
	"raycoon/internal/storage"
	"raycoon/internal/storage/models"
)

var (
	coreManager *core.Manager
)

var connectCmd = &cobra.Command{
	Use:               "connect [config-id-or-name]",
	Short:             "Connect to a proxy config",
	Long: `Connect to a proxy configuration by ID or name.
If no argument is provided, you'll be prompted to select a config.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeConfigNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Get the config to connect to
		var config *models.Config
		var err error

		if len(args) > 0 {
			// Parse argument as ID or name
			if id, parseErr := strconv.ParseInt(args[0], 10, 64); parseErr == nil {
				config, err = appInstance.Storage.GetConfig(ctx, id)
			} else {
				config, err = appInstance.Storage.GetConfigByName(ctx, args[0])
			}

			if err != nil {
				return fmt.Errorf("config not found: %s", args[0])
			}
		} else {
			// Auto-select logic
			groupName, _ := cmd.Flags().GetString("group")
			autoSelect, _ := cmd.Flags().GetBool("auto")

			if autoSelect {
				// Select lowest latency config
				config, err = selectLowestLatencyConfig(ctx, groupName)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("please specify a config ID or name, or use --auto")
			}
		}

		// Check if already connected
		activeConn, err := appInstance.Storage.GetActiveConnection(ctx)
		if err == nil && activeConn != nil {
			currentConfig, _ := appInstance.Storage.GetConfig(ctx, activeConn.ConfigID)
			if currentConfig != nil && currentConfig.ID == config.ID {
				return fmt.Errorf("already connected to: %s", config.Name)
			}

			// Ask to disconnect
			fmt.Printf("Already connected to: %s\n", currentConfig.Name)
			fmt.Print("Disconnect and connect to new config? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				return nil
			}

			// Disconnect current
			if err := disconnectCurrent(ctx); err != nil {
				return fmt.Errorf("failed to disconnect current connection: %w", err)
			}
		}

		// Get connection settings
		vpnModeStr, _ := cmd.Flags().GetString("mode")
		socksPort, _ := cmd.Flags().GetInt("port")
		httpPort, _ := cmd.Flags().GetInt("http-port")
		coreTypeStr, _ := cmd.Flags().GetString("core")

		// Parse VPN mode
		var vpnMode types.VPNMode
		switch vpnModeStr {
		case "proxy":
			vpnMode = types.VPNModeProxy
		case "tunnel":
			vpnMode = types.VPNModeTunnel
		default:
			// Get from settings
			modeFromDB, err := appInstance.Storage.GetSetting(ctx, "vpn_mode")
			if err == nil {
				vpnModeStr = modeFromDB
				vpnMode = types.VPNMode(modeFromDB)
			} else {
				vpnMode = types.VPNModeProxy
			}
		}

		// Get ports from settings if not specified
		if socksPort == 0 {
			if portStr, err := appInstance.Storage.GetSetting(ctx, "proxy_port"); err == nil {
				socksPort, _ = strconv.Atoi(portStr)
			} else {
				socksPort = 1080
			}
		}

		if httpPort == 0 {
			if portStr, err := appInstance.Storage.GetSetting(ctx, "http_proxy_port"); err == nil {
				httpPort, _ = strconv.Atoi(portStr)
			} else {
				httpPort = 1081
			}
		}

		// Get core type from settings if not specified
		if coreTypeStr == "" {
			if ct, err := appInstance.Storage.GetSetting(ctx, "active_core"); err == nil {
				coreTypeStr = ct
			} else {
				coreTypeStr = "xray"
			}
		}

		// Parse core type
		var coreType types.CoreType
		switch coreTypeStr {
		case "xray":
			coreType = types.CoreTypeXray
		case "singbox":
			coreType = types.CoreTypeSingbox
		default:
			coreType = types.CoreTypeXray
		}

		// Initialize core manager if needed
		if coreManager == nil {
			coreManager, err = core.NewManager(coreType)
			if err != nil {
				return fmt.Errorf("failed to initialize core: %w", err)
			}
		}

		// Build core config
		coreConfig := core.BuildCoreConfig(config, vpnMode, socksPort, httpPort)

		// Test latency first if requested
		if testFirst, _ := cmd.Flags().GetBool("test"); testFirst {
			fmt.Printf("Testing latency to %s... ", config.Name)

			timeoutMS := int64(5000)
			if val, err := appInstance.Storage.GetSetting(ctx, "latency_test_timeout"); err == nil {
				if parsed, parseErr := strconv.ParseInt(val, 10, 64); parseErr == nil {
					timeoutMS = parsed
				}
			}

			strategy, _ := latency.NewStrategy("")
			tester := latency.NewTester(appInstance.Storage, latency.TesterConfig{
				Workers:  1,
				Timeout:  time.Duration(timeoutMS) * time.Millisecond,
				Strategy: strategy,
			})

			result := tester.TestSingle(ctx, config)
			if result.Latency.Success {
				fmt.Printf("%d ms\n\n", *result.Latency.LatencyMS)
			} else {
				fmt.Printf("failed (%s)\n\n", result.Latency.ErrorMessage)
			}
		}

		// Connect
		fmt.Printf("Connecting to %s (%s)...\n", config.Name, config.Protocol)
		fmt.Printf("  Address:   %s:%d\n", config.Address, config.Port)
		fmt.Printf("  Mode:      %s\n", vpnMode)
		if vpnMode == types.VPNModeProxy {
			fmt.Printf("  SOCKS:     127.0.0.1:%d\n", socksPort)
			fmt.Printf("  HTTP:      127.0.0.1:%d\n", httpPort)
		}
		fmt.Printf("  Core:      %s\n", coreType)
		fmt.Println()

		if err := coreManager.Start(ctx, coreConfig); err != nil {
			return fmt.Errorf("failed to start proxy: %w", err)
		}

		// Enable TUN device for tunnel mode.
		if vpnMode == types.VPNModeTunnel {
			fmt.Println("Enabling TUN device...")
			if err := tun.Enable(socksPort, []string{config.Address}); err != nil {
				// Core started but TUN failed — stop core and bail.
				coreManager.Stop(ctx)
				return fmt.Errorf("failed to enable TUN device: %w", err)
			}
		}

		// Save active connection
		activeConnection := &models.ActiveConnection{
			ConfigID: config.ID,
			CoreType: string(coreType),
			VPNMode:  string(vpnMode),
		}

		if err := appInstance.Storage.SetActiveConnection(ctx, activeConnection); err != nil {
			// Connection started but failed to save to DB
			fmt.Fprintf(os.Stderr, "Warning: failed to save connection state: %v\n", err)
		}

		// Update config stats
		now := time.Now()
		config.LastUsed = &now
		config.UseCount++
		if err := appInstance.Storage.UpdateConfig(ctx, config); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update config stats: %v\n", err)
		}

		fmt.Println("🦝 Connected successfully!")
		fmt.Println()
		fmt.Println("Proxy is now running. Use 'raycoon disconnect' to stop.")
		if vpnMode == types.VPNModeTunnel {
			fmt.Println()
			fmt.Println("TUN device active — all system traffic is tunneled.")
		} else {
			fmt.Println()
			fmt.Println("Configure your applications to use:")
			fmt.Printf("  SOCKS5: 127.0.0.1:%d\n", socksPort)
			fmt.Printf("  HTTP:   127.0.0.1:%d\n", httpPort)
		}

		return nil
	},
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect current connection",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		if err := disconnectCurrent(ctx); err != nil {
			return err
		}

		fmt.Println("🦝 Disconnected successfully!")

		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show connection status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Get active connection
		activeConn, err := appInstance.Storage.GetActiveConnection(ctx)
		if err != nil || activeConn == nil {
			fmt.Println("Status: Not connected")
			return nil
		}

		// Get config details
		config, err := appInstance.Storage.GetConfig(ctx, activeConn.ConfigID)
		if err != nil {
			return fmt.Errorf("failed to get config: %w", err)
		}

		// Get group
		group, err := appInstance.Storage.GetGroup(ctx, config.GroupID)
		if err != nil {
			group = &models.Group{Name: "unknown"}
		}

		// Check if xray process is actually running
		coreRunning := isXrayProcessRunning()

		fmt.Println("Connection Status")
		fmt.Println("═════════════════")
		fmt.Println()
		if coreRunning {
			fmt.Printf("Status:     ● Connected\n")
		} else {
			fmt.Printf("Status:     ○ Disconnected (stale)\n")
		}
		fmt.Printf("Config:     %s (ID: %d)\n", config.Name, config.ID)
		fmt.Printf("Protocol:   %s\n", config.Protocol)
		fmt.Printf("Address:    %s:%d\n", config.Address, config.Port)
		fmt.Printf("Group:      %s\n", group.Name)
		fmt.Printf("Core:       %s\n", activeConn.CoreType)
		fmt.Printf("Mode:       %s\n", activeConn.VPNMode)
		fmt.Printf("Started:    %s\n", activeConn.StartedAt.Format(time.RFC3339))
		fmt.Printf("Uptime:     %s\n", time.Since(activeConn.StartedAt).Round(time.Second))

		if !coreRunning {
			fmt.Println()
			fmt.Println("⚠ Core process is not running. Connection may have been interrupted.")
			fmt.Println("  Use 'raycoon disconnect' to clear stale state, then reconnect.")
		}

		return nil
	},
}

// Helper functions

func disconnectCurrent(ctx context.Context) error {
	// Get active connection
	activeConn, err := appInstance.Storage.GetActiveConnection(ctx)
	if err != nil || activeConn == nil {
		return fmt.Errorf("no active connection")
	}

	// Disable TUN device if tunnel mode was active.
	if activeConn.VPNMode == string(types.VPNModeTunnel) {
		fmt.Println("Disabling TUN device...")
		if err := tun.Disable(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to disable TUN device: %v\n", err)
		}
	}

	// Initialize core manager if needed (for cross-process disconnect)
	if coreManager == nil {
		coreType := types.CoreType(activeConn.CoreType)
		coreManager, err = core.NewManager(coreType)
		if err != nil {
			// If we can't create the manager, try pkill as fallback
			fmt.Println("Stopping proxy core...")
			killCmd := exec.Command("pkill", "-f", "xray run -c.*raycoon")
			killCmd.Run()
			time.Sleep(500 * time.Millisecond)

			if err := appInstance.Storage.ClearActiveConnection(ctx); err != nil {
				return fmt.Errorf("failed to clear connection state: %w", err)
			}
			return nil
		}
	}

	// Stop core
	if coreManager.IsRunning() {
		fmt.Println("Stopping proxy core...")
		if err := coreManager.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop core: %w", err)
		}
	}

	// Clear active connection
	if err := appInstance.Storage.ClearActiveConnection(ctx); err != nil {
		return fmt.Errorf("failed to clear connection state: %w", err)
	}

	return nil
}

func selectLowestLatencyConfig(ctx context.Context, groupName string) (*models.Config, error) {
	filter := storage.ConfigFilter{
		Enabled: func() *bool { b := true; return &b }(),
	}

	if groupName != "" {
		group, err := appInstance.Storage.GetGroupByName(ctx, groupName)
		if err != nil {
			return nil, fmt.Errorf("group not found: %s", groupName)
		}
		filter.GroupID = &group.ID
	}

	configs, err := appInstance.Storage.GetAllConfigs(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no enabled configs found")
	}

	// Find config with lowest latency
	var bestConfig *models.Config
	var bestLatency int = int(^uint(0) >> 1) // Max int

	for _, cfg := range configs {
		latency, _ := appInstance.Storage.GetLatestLatency(ctx, cfg.ID)
		if latency != nil && latency.Success && latency.LatencyMS != nil {
			if *latency.LatencyMS < bestLatency {
				bestLatency = *latency.LatencyMS
				bestConfig = cfg
			}
		}
	}

	if bestConfig == nil {
		// No latency data, return first config
		return configs[0], nil
	}

	return bestConfig, nil
}

// isXrayProcessRunning checks if an xray process managed by raycoon is running
func isXrayProcessRunning() bool {
	// Check if the raycoon config file-based xray process is running
	out, err := exec.Command("pgrep", "-f", "xray run -c.*raycoon").Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func init() {
	// Connect flags
	connectCmd.Flags().StringP("group", "g", "", "select from group")
	connectCmd.Flags().StringP("mode", "m", "", "VPN mode (tunnel/proxy)")
	connectCmd.Flags().IntP("port", "p", 0, "SOCKS proxy port")
	connectCmd.Flags().Int("http-port", 0, "HTTP proxy port")
	connectCmd.Flags().String("core", "", "core to use (xray/singbox)")
	connectCmd.Flags().Bool("test", false, "test latency before connecting")
	connectCmd.Flags().Bool("auto", false, "auto-select lowest latency config")

	// Connect flag completions
	connectCmd.RegisterFlagCompletionFunc("group", completeGroupNamesForFlag)
	connectCmd.RegisterFlagCompletionFunc("mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"tunnel", "proxy"}, cobra.ShellCompDirectiveNoFileComp
	})
	connectCmd.RegisterFlagCompletionFunc("core", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"xray", "singbox"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Disconnect flags
	disconnectCmd.Flags().BoolP("force", "f", false, "force disconnect")

	// Status flags
	statusCmd.Flags().Bool("stats", false, "show detailed statistics")

	// Add to root
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(disconnectCmd)
	rootCmd.AddCommand(statusCmd)
}
