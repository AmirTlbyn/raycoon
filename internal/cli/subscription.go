package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"raycoon/internal/subscription"
)

var (
	subManager *subscription.Manager
)

var subCmd = &cobra.Command{
	Use:     "sub",
	Aliases: []string{"subscription"},
	Short:   "Manage subscriptions",
	Long:    "Add, update, list, and manage subscription groups",
}

var subUpdateCmd = &cobra.Command{
	Use:               "update [group-name]",
	Short:             "Update subscription",
	Long:              "Update a subscription group or all groups if --all is specified",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Initialize subscription manager if needed
		if subManager == nil {
			subManager = subscription.NewManager(appInstance.Storage, appInstance.Parser)
		}

		updateAll, _ := cmd.Flags().GetBool("all")

		if updateAll {
			// Update all groups
			fmt.Println("Updating all subscription groups...")
			results, err := subManager.UpdateAllDue(ctx)
			if err != nil {
				return fmt.Errorf("failed to update subscriptions: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("No subscription groups due for update.")
				return nil
			}

			// Print results
			fmt.Println()
			for _, result := range results {
				fmt.Printf("Group: %s\n", result.GroupName)
				fmt.Printf("  Total URIs:    %d\n", result.TotalURIs)
				fmt.Printf("  Added:         %d\n", result.Added)
				fmt.Printf("  Failed:        %d\n", result.Failed)

				if len(result.Errors) > 0 {
					fmt.Printf("  Errors:\n")
					for _, err := range result.Errors {
						fmt.Printf("    - %v\n", err)
					}
				}
				fmt.Println()
			}

			fmt.Printf("🦝 Updated %d subscription groups\n", len(results))

		} else {
			// Update specific group
			if len(args) == 0 {
				return fmt.Errorf("please specify a group name or use --all")
			}

			groupName := args[0]
			fmt.Printf("Updating subscription for group '%s'...\n", groupName)

			result, err := subManager.UpdateGroupByName(ctx, groupName)
			if err != nil {
				return fmt.Errorf("failed to update subscription: %w", err)
			}

			fmt.Println()
			fmt.Printf("🦝 Subscription updated!\n\n")
			fmt.Printf("  Total URIs:    %d\n", result.TotalURIs)
			fmt.Printf("  Added:         %d configs\n", result.Added)
			fmt.Printf("  Failed:        %d\n", result.Failed)

			if len(result.Errors) > 0 {
				fmt.Printf("\n  Errors:\n")
				for _, err := range result.Errors {
					fmt.Printf("    - %v\n", err)
				}
			}
		}

		return nil
	},
}

var subStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show subscription status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Initialize subscription manager if needed
		if subManager == nil {
			subManager = subscription.NewManager(appInstance.Storage, appInstance.Parser)
		}

		statuses, err := subManager.GetUpdateStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get subscription status: %w", err)
		}

		if len(statuses) == 0 {
			fmt.Println("No subscription groups found.")
			return nil
		}

		// Print table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "GROUP\tCONFIGS\tAUTO-UPDATE\tINTERVAL\tLAST UPDATED\tNEXT UPDATE\tSTATUS")
		fmt.Fprintln(w, "-----\t-------\t-----------\t--------\t------------\t-----------\t------")

		for _, status := range statuses {
			autoUpdate := "✗"
			if status.AutoUpdate {
				autoUpdate = "✓"
			}

			interval := formatDuration(status.Interval)

			lastUpdated := "Never"
			if status.LastUpdated != nil {
				lastUpdated = formatTime(*status.LastUpdated)
			}

			nextUpdate := "N/A"
			if status.NextUpdate != nil {
				nextUpdate = formatTime(*status.NextUpdate)
			}

			statusStr := "OK"
			if status.IsDue {
				statusStr = "⚠ Due"
			}

			fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
				status.GroupName, status.ConfigCount, autoUpdate, interval,
				lastUpdated, nextUpdate, statusStr)
		}

		w.Flush()

		return nil
	},
}

// Helper functions

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	if hours >= 24 {
		days := hours / 24
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dh", hours)
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}
	if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}

	return t.Format("2006-01-02")
}

func init() {
	// Update flags
	subUpdateCmd.Flags().Bool("all", false, "update all subscriptions")
	subUpdateCmd.Flags().BoolP("force", "f", false, "force update even if not due")

	// Add subcommands
	subCmd.AddCommand(subUpdateCmd)
	subCmd.AddCommand(subStatusCmd)

	// Add to root
	rootCmd.AddCommand(subCmd)
}
