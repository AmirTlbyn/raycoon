package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for raycoon.

To load completions:

Bash:
  $ source <(raycoon completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ raycoon completion bash > /etc/bash_completion.d/raycoon
  # macOS:
  $ raycoon completion bash > $(brew --prefix)/etc/bash_completion.d/raycoon

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ raycoon completion zsh > "${fpath[1]}/_raycoon"
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ raycoon completion fish | source
  # To load completions for each session, execute once:
  $ raycoon completion fish > ~/.config/fish/completions/raycoon.fish

PowerShell:
  PS> raycoon completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> raycoon completion powershell > raycoon.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
