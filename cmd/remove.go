package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jahvon/flow/config/cache"
	"github.com/jahvon/flow/config/file"
	"github.com/jahvon/flow/internal/vault"
)

var removeCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm"},
	Short:   "Remove a flow object.",
}

var workspaceRemoveCmd = &cobra.Command{
	Use:     "workspace <name>",
	Aliases: []string{"ws"},
	Short:   "Remove an existing workspace from the list of known workspaces.",
	Long: "Remove an existing workspace. File contents will remain in the corresponding directory but the " +
		"workspace will be unlinked from the flow global configurations.\nNote: You cannot remove the current workspace.",
	Args:   cobra.ExactArgs(1),
	PreRun: setTermView,
	Run: func(cmd *cobra.Command, args []string) {
		logger := curCtx.Logger
		name := args[0]

		confirmed := processUserConfirmation("Are you sure you want to remove the workspace '" + name + "'?")
		if !confirmed {
			logger.Warnf("Aborting")
			return
		}

		userConfig := file.LoadUserConfig()
		if userConfig == nil {
			logger.Fatalf("failed to load user config")
		}
		if err := userConfig.Validate(); err != nil {
			logger.FatalErr(err)
		}

		if name == userConfig.CurrentWorkspace {
			logger.Fatalf("cannot remove the current workspace")
		}
		if _, found := userConfig.Workspaces[name]; !found {
			logger.Fatalf("workspace %s was not found", name)
		}

		delete(userConfig.Workspaces, name)
		if err := file.WriteUserConfig(userConfig); err != nil {
			logger.FatalErr(err)
		}

		logger.Warnf("Workspace '%s' removed", name)

		if err := cache.UpdateAll(); err != nil {
			logger.FatalErr(fmt.Errorf("failed to update cache - %w", err))
		}
	},
}

var vaultSecretRemoveCmd = &cobra.Command{
	Use:     "secret <name>",
	Aliases: []string{"scrt"},
	Short:   "Remove a secret from the vault.",
	Args:    cobra.ExactArgs(1),
	PreRun:  setTermView,
	Run: func(cmd *cobra.Command, args []string) {
		logger := curCtx.Logger
		reference := args[0]

		v := vault.NewVault()
		err := v.DeleteSecret(reference)
		if err != nil {
			logger.FatalErr(err)
		}
		logger.PlainTextSuccess(fmt.Sprintf("Secret %s removed from vault", reference))
	},
}

func init() {
	removeCmd.AddCommand(workspaceRemoveCmd)
	removeCmd.AddCommand(vaultSecretRemoveCmd)

	rootCmd.AddCommand(removeCmd)
}
