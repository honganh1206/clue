package commands

import (
	"os"

	"github.com/spf13/cobra"
)

// Auto-complete  commands, flags, and args by sourcing the generated scripts to the shell environment
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:
  $ source <(adrift completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ adrift completion bash > /etc/bash_completion.d/adrift
  # macOS:
  $ adrift completion bash > /usr/local/etc/bash_completion.d/adrift

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ adrift completion zsh > "${fpath[1]}/_adrift"

Fish:
  $ adrift completion fish > ~/.config/fish/completions/adrift.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
			// TODO: When porting on Windows maybe
			// case "powershell":
			// 	cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}
