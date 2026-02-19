package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"raycoon/internal/app"
)

var (
	appInstance *app.App
	version     = "dev"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "raycoon",
	Short: "🦝 Raycoon - A modern V2Ray/proxy CLI client",
	Long: `🦝 Raycoon - A modern V2Ray/proxy CLI client

  Manage V2Ray/Xray proxy connections from your terminal.

  Quick start:
    raycoon group create myproxy --subscription "https://..."
    raycoon sub update myproxy
    raycoon test --all
    raycoon connect --auto

  Core features:
    • Xray-core with VLESS, VMess, Trojan, Shadowsocks, Reality
    • Subscription management with auto-update
    • TCP & HTTP latency testing with parallel workers
    • Proxy mode (SOCKS5/HTTP) and TUN mode (system-wide tunneling)`,
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize app
		var err error
		appInstance, err = app.New()
		if err != nil {
			return fmt.Errorf("failed to initialize application: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Cleanup
		if appInstance != nil {
			return appInstance.Close()
		}
		return nil
	},
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("db", "", "database path")

	// Add subcommands
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(groupCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🦝 Raycoon %s\n", version)
	},
}
