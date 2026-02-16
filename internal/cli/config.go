package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"raycoon/internal/storage"
	"raycoon/internal/storage/models"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage proxy configurations",
	Long:  "Add, list, show, edit, and delete proxy configurations",
}

var configAddCmd = &cobra.Command{
	Use:   "add <uri>",
	Short: "Add config from URI",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		uri := args[0]

		// Parse URI
		config, err := appInstance.Parser.Parse(uri)
		if err != nil {
			return fmt.Errorf("failed to parse URI: %w", err)
		}

		// Get flags
		groupName, _ := cmd.Flags().GetString("group")
		customName, _ := cmd.Flags().GetString("name")
		tags, _ := cmd.Flags().GetStringSlice("tags")
		notes, _ := cmd.Flags().GetString("notes")

		// Get or create group
		ctx := context.Background()
		var group *models.Group

		if groupName == "" || groupName == "global" {
			group, err = appInstance.Storage.GetGlobalGroup(ctx)
			if err != nil {
				return fmt.Errorf("failed to get global group: %w", err)
			}
		} else {
			group, err = appInstance.Storage.GetGroupByName(ctx, groupName)
			if err != nil {
				// Create group if it doesn't exist
				group = &models.Group{
					Name:        groupName,
					AutoUpdate:  false,
					UpdateInterval: 86400,
				}
				if err := appInstance.Storage.CreateGroup(ctx, group); err != nil {
					return fmt.Errorf("failed to create group: %w", err)
				}
				fmt.Printf("Created new group: %s\n", groupName)
			}
		}

		// Set group ID
		config.GroupID = group.ID
		config.FromSubscription = false // Manually added

		// Set custom fields
		if customName != "" {
			config.Name = customName
		}
		if len(tags) > 0 {
			config.Tags = tags
		}
		if notes != "" {
			config.Notes = notes
		}

		// Save to database
		if err := appInstance.Storage.CreateConfig(ctx, config); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("🦝 Config added successfully!\n\n")
		fmt.Printf("  ID:       %d\n", config.ID)
		fmt.Printf("  Name:     %s\n", config.Name)
		fmt.Printf("  Protocol: %s\n", config.Protocol)
		fmt.Printf("  Address:  %s:%d\n", config.Address, config.Port)
		fmt.Printf("  Group:    %s\n", group.Name)
		if len(config.Tags) > 0 {
			fmt.Printf("  Tags:     %v\n", config.Tags)
		}

		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configs",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Get flags
		groupName, _ := cmd.Flags().GetString("group")
		protocol, _ := cmd.Flags().GetString("protocol")
		enabledOnly, _ := cmd.Flags().GetBool("enabled")

		// Build filter
		filter := storage.ConfigFilter{}

		if groupName != "" {
			group, err := appInstance.Storage.GetGroupByName(ctx, groupName)
			if err != nil {
				return fmt.Errorf("group not found: %s", groupName)
			}
			filter.GroupID = &group.ID
		}

		if protocol != "" {
			filter.Protocol = &protocol
		}

		if enabledOnly {
			enabled := true
			filter.Enabled = &enabled
		}

		// Get configs
		configs, err := appInstance.Storage.GetAllConfigs(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to get configs: %w", err)
		}

		if len(configs) == 0 {
			fmt.Println("No configs found.")
			return nil
		}

		// Print table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tPROTOCOL\tADDRESS\tGROUP\tLATENCY\tENABLED")
		fmt.Fprintln(w, "--\t----\t--------\t-------\t-----\t-------\t-------")

		// Get all groups for display
		groups, err := appInstance.Storage.GetAllGroups(ctx)
		if err != nil {
			return err
		}
		groupMap := make(map[int64]string)
		for _, g := range groups {
			groupMap[g.ID] = g.Name
		}

		for _, config := range configs {
			enabled := "✗"
			if config.Enabled {
				enabled = "✓"
			}

			groupName := groupMap[config.GroupID]

			latStr := "-"
			if lat, err := appInstance.Storage.GetLatestLatency(ctx, config.ID); err == nil && lat != nil {
				if lat.Success && lat.LatencyMS != nil {
					latStr = fmt.Sprintf("%d ms", *lat.LatencyMS)
				} else {
					latStr = "fail"
				}
			}

			fmt.Fprintf(w, "%d\t%s\t%s\t%s:%d\t%s\t%s\t%s\n",
				config.ID, config.Name, config.Protocol,
				config.Address, config.Port, groupName, latStr, enabled)
		}

		w.Flush()

		fmt.Printf("\nTotal: %d configs\n", len(configs))

		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:               "show <id>",
	Short:             "Show config details",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeConfigNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Parse ID
		var id int64
		if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
			// Try to find by name
			config, err := appInstance.Storage.GetConfigByName(ctx, args[0])
			if err != nil {
				return fmt.Errorf("config not found: %s", args[0])
			}
			id = config.ID
		}

		// Get config
		config, err := appInstance.Storage.GetConfig(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get config: %w", err)
		}

		// Get group
		group, err := appInstance.Storage.GetGroup(ctx, config.GroupID)
		if err != nil {
			return fmt.Errorf("failed to get group: %w", err)
		}

		// Print details
		fmt.Printf("Config Details\n")
		fmt.Printf("══════════════\n\n")
		fmt.Printf("ID:           %d\n", config.ID)
		fmt.Printf("Name:         %s\n", config.Name)
		fmt.Printf("Protocol:     %s\n", config.Protocol)
		fmt.Printf("Address:      %s:%d\n", config.Address, config.Port)
		fmt.Printf("Network:      %s\n", config.Network)
		fmt.Printf("Group:        %s\n", group.Name)
		fmt.Printf("From Sub:     %v\n", config.FromSubscription)
		fmt.Printf("Enabled:      %v\n", config.Enabled)

		if len(config.Tags) > 0 {
			fmt.Printf("Tags:         %v\n", config.Tags)
		}

		if config.Notes != "" {
			fmt.Printf("Notes:        %s\n", config.Notes)
		}

		fmt.Printf("TLS Enabled:  %v\n", config.TLSEnabled)

		if config.UseCount > 0 {
			fmt.Printf("Use Count:    %d\n", config.UseCount)
		}

		if config.LastUsed != nil {
			fmt.Printf("Last Used:    %s\n", config.LastUsed.Format(time.RFC3339))
		}

		fmt.Printf("Created:      %s\n", config.CreatedAt.Format(time.RFC3339))
		fmt.Printf("Updated:      %s\n", config.UpdatedAt.Format(time.RFC3339))

		// Get latest latency
		latency, err := appInstance.Storage.GetLatestLatency(ctx, config.ID)
		if err == nil && latency != nil {
			fmt.Printf("\nLatest Latency Test:\n")
			if latency.Success && latency.LatencyMS != nil {
				fmt.Printf("  Latency:    %d ms\n", *latency.LatencyMS)
			} else {
				fmt.Printf("  Status:     Failed\n")
				if latency.ErrorMessage != "" {
					fmt.Printf("  Error:      %s\n", latency.ErrorMessage)
				}
			}
			fmt.Printf("  Tested:     %s\n", latency.TestedAt.Format(time.RFC3339))
		}

		// Show URI if available
		if config.URI != "" {
			fmt.Printf("\nURI:\n%s\n", config.URI)
		}

		return nil
	},
}

var configDeleteCmd = &cobra.Command{
	Use:               "delete <id>",
	Short:             "Delete config",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeConfigNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Parse ID
		var id int64
		if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
			config, err := appInstance.Storage.GetConfigByName(ctx, args[0])
			if err != nil {
				return fmt.Errorf("config not found: %s", args[0])
			}
			id = config.ID
		}

		// Get config for confirmation
		config, err := appInstance.Storage.GetConfig(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get config: %w", err)
		}

		// Confirm deletion
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Delete config '%s' (ID: %d)? [y/N]: ", config.Name, config.ID)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		// Delete
		if err := appInstance.Storage.DeleteConfig(ctx, id); err != nil {
			return fmt.Errorf("failed to delete config: %w", err)
		}

		fmt.Printf("🦝 Config deleted: %s\n", config.Name)

		return nil
	},
}

func init() {
	// Add flags
	configAddCmd.Flags().StringP("group", "g", "global", "group name")
	configAddCmd.Flags().StringP("name", "n", "", "custom name")
	configAddCmd.Flags().StringSlice("tags", []string{}, "tags (comma-separated)")
	configAddCmd.Flags().String("notes", "", "notes")

	configListCmd.Flags().StringP("group", "g", "", "filter by group")
	configListCmd.Flags().StringP("protocol", "p", "", "filter by protocol")
	configListCmd.Flags().Bool("enabled", false, "show only enabled")

	configDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	// Flag completions
	configAddCmd.RegisterFlagCompletionFunc("group", completeGroupNamesForFlag)
	configListCmd.RegisterFlagCompletionFunc("group", completeGroupNamesForFlag)

	// Add subcommands
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configDeleteCmd)
}
