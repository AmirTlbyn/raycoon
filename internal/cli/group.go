package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"raycoon/internal/storage"
	"raycoon/internal/storage/models"
	"raycoon/internal/subscription"
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage config groups",
	Long:  "Create, list, and manage configuration groups",
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		groups, err := appInstance.Storage.GetAllGroups(ctx)
		if err != nil {
			return fmt.Errorf("failed to get groups: %w", err)
		}

		if len(groups) == 0 {
			fmt.Println("No groups found.")
			return nil
		}

		// Print table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSUBSCRIPTION\tAUTO-UPDATE\tDESCRIPTION")
		fmt.Fprintln(w, "--\t----\t------------\t-----------\t-----------")

		for _, group := range groups {
			hasSub := "No"
			if group.SubscriptionURL != nil && *group.SubscriptionURL != "" {
				hasSub = "Yes"
			}

			autoUpdate := "✗"
			if group.AutoUpdate {
				autoUpdate = "✓"
			}

			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
				group.ID, group.Name, hasSub, autoUpdate, group.Description)
		}

		w.Flush()

		fmt.Printf("\nTotal: %d groups\n", len(groups))

		return nil
	},
}

var groupCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name := args[0]

		desc, _ := cmd.Flags().GetString("desc")
		subURL, _ := cmd.Flags().GetString("subscription")
		autoUpdate, _ := cmd.Flags().GetBool("auto-update")
		interval, _ := cmd.Flags().GetInt("interval")

		group := &models.Group{
			Name:           name,
			Description:    desc,
			AutoUpdate:     autoUpdate,
			UpdateInterval: interval,
		}

		if subURL != "" {
			group.SubscriptionURL = &subURL
		}

		if err := appInstance.Storage.CreateGroup(ctx, group); err != nil {
			return fmt.Errorf("failed to create group: %w", err)
		}

		fmt.Printf("🦝 Group created!\n\n")
		fmt.Printf("  ID:          %d\n", group.ID)
		fmt.Printf("  Name:        %s\n", group.Name)
		if desc != "" {
			fmt.Printf("  Description: %s\n", desc)
		}
		if subURL != "" {
			fmt.Printf("  Subscription: %s\n", subURL)
			fmt.Printf("  Auto-update:  %v\n", autoUpdate)
			fmt.Printf("  Interval:     %ds\n", interval)

			// Ask if user wants to update now
			fmt.Print("\nUpdate subscription now? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if response == "y" || response == "Y" {
				// Initialize subscription manager
				if subManager == nil {
					subManager = subscription.NewManager(appInstance.Storage, appInstance.Parser)
				}

				fmt.Println("\nUpdating subscription...")
				result, err := subManager.UpdateGroup(ctx, group.ID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to update subscription: %v\n", err)
				} else {
					fmt.Printf("🦝 Added %d configs from subscription\n", result.Added)
					if result.Failed > 0 {
						fmt.Printf("  Failed: %d configs\n", result.Failed)
					}
				}
			}
		}

		return nil
	},
}

var groupDeleteCmd = &cobra.Command{
	Use:               "delete <name>",
	Short:             "Delete a group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name := args[0]

		// Get group
		group, err := appInstance.Storage.GetGroupByName(ctx, name)
		if err != nil {
			return fmt.Errorf("group not found: %s", name)
		}

		// Check if global
		if group.IsGlobal {
			return fmt.Errorf("cannot delete global group")
		}

		// Confirm deletion
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Delete group '%s' and all its configs? [y/N]: ", name)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		// Delete
		if err := appInstance.Storage.DeleteGroup(ctx, group.ID); err != nil {
			return fmt.Errorf("failed to delete group: %w", err)
		}

		fmt.Printf("🦝 Group deleted: %s\n", name)

		return nil
	},
}

var groupConfigsCmd = &cobra.Command{
	Use:               "configs <group-name>",
	Short:             "List configs in a group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeGroupNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name := args[0]

		// Get group
		group, err := appInstance.Storage.GetGroupByName(ctx, name)
		if err != nil {
			return fmt.Errorf("group not found: %s", name)
		}

		// Get configs in this group
		filter := storage.ConfigFilter{
			GroupID: &group.ID,
		}
		configs, err := appInstance.Storage.GetAllConfigs(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to get configs: %w", err)
		}

		if len(configs) == 0 {
			fmt.Printf("No configs in group '%s'.\n", name)
			return nil
		}

		fmt.Printf("Configs in group: %s\n", name)
		fmt.Println(strings.Repeat("═", 60))
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tPROTOCOL\tADDRESS\tLATENCY\tENABLED")
		fmt.Fprintln(w, "--\t----\t--------\t-------\t-------\t-------")

		for _, config := range configs {
			enabled := "✗"
			if config.Enabled {
				enabled = "✓"
			}

			latStr := "-"
			if lat, latErr := appInstance.Storage.GetLatestLatency(ctx, config.ID); latErr == nil && lat != nil {
				if lat.Success && lat.LatencyMS != nil {
					latStr = fmt.Sprintf("%d ms", *lat.LatencyMS)
				} else {
					latStr = "fail"
				}
			}

			fmt.Fprintf(w, "%d\t%s\t%s\t%s:%d\t%s\t%s\n",
				config.ID, config.Name, config.Protocol,
				config.Address, config.Port, latStr, enabled)
		}

		w.Flush()
		fmt.Printf("\nTotal: %d configs\n", len(configs))

		return nil
	},
}

func init() {
	groupCreateCmd.Flags().String("desc", "", "group description")
	groupCreateCmd.Flags().String("subscription", "", "subscription URL")
	groupCreateCmd.Flags().Bool("auto-update", true, "enable auto-update")
	groupCreateCmd.Flags().Int("interval", 86400, "update interval in seconds")

	groupDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	groupCmd.AddCommand(groupListCmd)
	groupCmd.AddCommand(groupCreateCmd)
	groupCmd.AddCommand(groupDeleteCmd)
	groupCmd.AddCommand(groupConfigsCmd)
}
