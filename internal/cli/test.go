package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"raycoon/internal/latency"
	"raycoon/internal/storage"
	"raycoon/internal/storage/models"
)

var testCmd = &cobra.Command{
	Use:   "test [id-or-name]",
	Short: "Test proxy latency",
	Long: `Test latency of proxy configurations.

Test a single config by ID or name, or test multiple configs with --all or --group.
Default strategy is HTTP (full proxy validation). Use --strategy tcp for fast handshake test.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeConfigNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		strategyName, _ := cmd.Flags().GetString("strategy")
		workers, _ := cmd.Flags().GetInt64("workers")
		timeoutMS, _ := cmd.Flags().GetInt64("timeout")
		all, _ := cmd.Flags().GetBool("all")
		groupName, _ := cmd.Flags().GetString("group")

		// Load defaults from DB settings if not overridden
		if !cmd.Flags().Changed("workers") {
			if val, err := appInstance.Storage.GetSetting(ctx, "latency_test_workers"); err == nil {
				if parsed, parseErr := strconv.ParseInt(val, 10, 64); parseErr == nil {
					workers = parsed
				}
			}
		}
		if !cmd.Flags().Changed("timeout") {
			if val, err := appInstance.Storage.GetSetting(ctx, "latency_test_timeout"); err == nil {
				if parsed, parseErr := strconv.ParseInt(val, 10, 64); parseErr == nil {
					timeoutMS = parsed
				}
			}
		}

		strategy, err := latency.NewStrategy(strategyName)
		if err != nil {
			return err
		}

		tester := latency.NewTester(appInstance.Storage, latency.TesterConfig{
			Workers:  workers,
			Timeout:  time.Duration(timeoutMS) * time.Millisecond,
			Strategy: strategy,
		})

		if all || groupName != "" {
			return runBatchTest(ctx, tester, all, groupName)
		}

		if len(args) == 0 {
			return fmt.Errorf("please specify a config ID or name, or use --all / --group")
		}

		return runSingleTest(ctx, tester, args[0])
	},
}

var testHistoryCmd = &cobra.Command{
	Use:               "history <id-or-name>",
	Short:             "Show latency test history",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeConfigNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		limit, _ := cmd.Flags().GetInt("limit")

		config, err := resolveConfig(ctx, args[0])
		if err != nil {
			return err
		}

		history, err := appInstance.Storage.GetLatencyHistory(ctx, config.ID, limit)
		if err != nil {
			return err
		}
		if len(history) == 0 {
			fmt.Printf("No latency history for %s\n", config.Name)
			return nil
		}

		fmt.Printf("Latency History: %s (%s:%d)\n", config.Name, config.Address, config.Port)
		fmt.Println(strings.Repeat("═", 50))
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TIME\tSTRATEGY\tLATENCY\tSTATUS")
		fmt.Fprintln(w, "----\t--------\t-------\t------")

		for _, entry := range history {
			latStr := "N/A"
			statusStr := "FAIL"
			if entry.Success && entry.LatencyMS != nil {
				latStr = fmt.Sprintf("%d ms", *entry.LatencyMS)
				statusStr = "OK"
			}
			timeStr := entry.TestedAt.Format("2006-01-02 15:04:05")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				timeStr, entry.TestStrategy, latStr, statusStr)
		}
		w.Flush()

		return nil
	},
}

func resolveConfig(ctx context.Context, identifier string) (*models.Config, error) {
	if id, parseErr := strconv.ParseInt(identifier, 10, 64); parseErr == nil {
		config, err := appInstance.Storage.GetConfig(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("config not found: %s", identifier)
		}
		return config, nil
	}
	config, err := appInstance.Storage.GetConfigByName(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("config not found: %s", identifier)
	}
	return config, nil
}

func runSingleTest(ctx context.Context, tester *latency.Tester, identifier string) error {
	config, err := resolveConfig(ctx, identifier)
	if err != nil {
		return err
	}

	fmt.Printf("Testing %s (%s:%d)... ", config.Name, config.Address, config.Port)

	result := tester.TestSingle(ctx, config)

	if result.Latency.Success {
		fmt.Printf("%d ms\n", *result.Latency.LatencyMS)
	} else {
		fmt.Printf("FAILED (%s)\n", result.Latency.ErrorMessage)
	}

	return nil
}

func runBatchTest(ctx context.Context, tester *latency.Tester, all bool, groupName string) error {
	filter := storage.ConfigFilter{
		Enabled: func() *bool { b := true; return &b }(),
	}
	if groupName != "" {
		group, err := appInstance.Storage.GetGroupByName(ctx, groupName)
		if err != nil {
			return fmt.Errorf("group not found: %s", groupName)
		}
		filter.GroupID = &group.ID
	}

	configs, err := appInstance.Storage.GetAllConfigs(ctx, filter)
	if err != nil {
		return err
	}
	if len(configs) == 0 {
		fmt.Println("No enabled configs found.")
		return nil
	}

	fmt.Printf("Testing %d configs...\n\n", len(configs))

	progress := func(result *latency.TestResult, current, total int) {
		if result.Latency.Success {
			fmt.Printf("  [%d/%d] %-40s %d ms\n", current, total,
				truncateName(result.Config.Name, 40), *result.Latency.LatencyMS)
		} else {
			fmt.Printf("  [%d/%d] %-40s FAILED\n", current, total,
				truncateName(result.Config.Name, 40))
		}
	}

	batch := tester.TestBatch(ctx, configs, progress)

	// Print sorted results table
	fmt.Printf("\n\nResults (sorted by latency):\n")
	fmt.Println(strings.Repeat("─", 75))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tNAME\tADDRESS\tLATENCY\tSTATUS")
	fmt.Fprintln(w, "-\t----\t-------\t-------\t------")

	for i, result := range batch.Results {
		latStr := "N/A"
		statusStr := "FAIL"
		if result.Latency.Success {
			latStr = fmt.Sprintf("%d ms", *result.Latency.LatencyMS)
			statusStr = "OK"
		}
		fmt.Fprintf(w, "%d\t%s\t%s:%d\t%s\t%s\n",
			i+1, truncateName(result.Config.Name, 35),
			result.Config.Address, result.Config.Port,
			latStr, statusStr)
	}
	w.Flush()

	fmt.Printf("\nSummary: %d tested, %d succeeded, %d failed (%.1fs)\n",
		batch.Tested, batch.Succeeded, batch.Failed, batch.Duration.Seconds())

	return nil
}

func truncateName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	return name[:maxLen-3] + "..."
}

func init() {
	testCmd.Flags().StringP("strategy", "s", "http", "test strategy (http, tcp)")
	testCmd.Flags().Int64P("workers", "w", 10, "number of concurrent workers")
	testCmd.Flags().Int64P("timeout", "t", 5000, "per-test timeout in milliseconds")
	testCmd.Flags().Bool("all", false, "test all enabled configs")
	testCmd.Flags().StringP("group", "g", "", "test all configs in a group")

	// Test flag completions
	testCmd.RegisterFlagCompletionFunc("strategy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"http", "tcp"}, cobra.ShellCompDirectiveNoFileComp
	})
	testCmd.RegisterFlagCompletionFunc("group", completeGroupNamesForFlag)

	testHistoryCmd.Flags().IntP("limit", "n", 20, "number of history entries")

	testCmd.AddCommand(testHistoryCmd)
	rootCmd.AddCommand(testCmd)
}
