package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"raycoon/internal/core"
	"raycoon/internal/core/types"
	"raycoon/internal/subscription"
	"raycoon/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive terminal UI",
	Long:  `Launch the full-screen interactive terminal UI for managing configs, groups, and connections.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Determine active core type from settings.
		coreTypeStr := "xray"
		if ct, err := appInstance.Storage.GetSetting(ctx, "active_core"); err == nil && ct != "" {
			coreTypeStr = ct
		}
		var coreType types.CoreType
		switch coreTypeStr {
		case "singbox":
			coreType = types.CoreTypeSingbox
		default:
			coreType = types.CoreTypeXray
		}

		coreMgr, err := core.NewManager(coreType)
		if err != nil {
			return fmt.Errorf("failed to initialize core manager: %w", err)
		}

		subMgr := subscription.NewManager(appInstance.Storage, appInstance.Parser)

		deps := tui.Deps{
			Storage: appInstance.Storage,
			CoreMgr: coreMgr,
			Parser:  appInstance.Parser,
			SubMgr:  subMgr,
		}

		p := tui.NewProgram(deps)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
