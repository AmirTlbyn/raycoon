package cli

// TUI is planned for a future release.
// The BubbleTea-based interactive TUI is implemented in internal/tui/
// but needs further polish before production use.
//
// TODO (Phase 5 - Future):
//   - Fix rendering issues with bubbles/table content bleeding between tabs
//   - Fix forceHeight approach for consistent line counts
//   - Settings choice selector (VPN mode, core, strategy)
//   - Stable connect/disconnect from TUI
//   - Live traffic stats via xray gRPC API
//
// To re-enable: uncomment the init() function below.

// func init() {
// 	rootCmd.AddCommand(tuiCmd)
// }
