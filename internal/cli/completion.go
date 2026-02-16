package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"raycoon/internal/app"
	"raycoon/internal/storage"
)

// ensureApp lazily initializes appInstance for shell completion.
// Cobra may invoke ValidArgsFunction without running PersistentPreRunE.
func ensureApp() error {
	if appInstance != nil {
		return nil
	}
	var err error
	appInstance, err = app.New()
	return err
}

// completeConfigNames provides shell completion for config names/IDs.
func completeConfigNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	if err := ensureApp(); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	ctx := context.Background()
	configs, err := appInstance.Storage.GetAllConfigs(ctx, storage.ConfigFilter{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, cfg := range configs {
		if strings.HasPrefix(strings.ToLower(cfg.Name), strings.ToLower(toComplete)) {
			completions = append(completions, cfg.Name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeGroupNames provides shell completion for group names.
func completeGroupNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	if err := ensureApp(); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	ctx := context.Background()
	groups, err := appInstance.Storage.GetAllGroups(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, g := range groups {
		if strings.HasPrefix(strings.ToLower(g.Name), strings.ToLower(toComplete)) {
			completions = append(completions, g.Name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeGroupNamesForFlag provides group name completion for --group flags.
func completeGroupNamesForFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if err := ensureApp(); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	ctx := context.Background()
	groups, err := appInstance.Storage.GetAllGroups(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, g := range groups {
		if strings.HasPrefix(strings.ToLower(g.Name), strings.ToLower(toComplete)) {
			completions = append(completions, g.Name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
