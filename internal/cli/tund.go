package cli

import (
	"github.com/spf13/cobra"
	"raycoon/internal/core/tun"
)

// tundCmd is a hidden internal command started by "raycoon connect -m tun".
// It owns the TUN device and tun2socks engine for the lifetime of the
// connection. Users should never invoke this directly.
var tundCmd = &cobra.Command{
	Use:    "tund",
	Short:  "Internal TUN daemon (not for direct use)",
	Hidden: true,

	// Skip the root PersistentPreRunE so we don't initialise the DB / app.
	// The daemon only needs the TUN package — no storage access required.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },

	RunE: func(cmd *cobra.Command, args []string) error {
		socksPort, _ := cmd.Flags().GetInt("socks-port")
		bypasses, _ := cmd.Flags().GetStringArray("bypass")
		return tun.RunDaemon(socksPort, bypasses)
	},
}

func init() {
	tundCmd.Flags().Int("socks-port", 1080, "SOCKS5 proxy port")
	tundCmd.Flags().StringArray("bypass", nil, "Addresses to bypass TUN routing (repeatable)")
	rootCmd.AddCommand(tundCmd)
}
