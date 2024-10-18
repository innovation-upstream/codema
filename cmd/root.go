package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "codema",
	Short: "Codema is a code generation tool",
	Long:  `Codema is a flexible code generation tool that helps you generate code based on your API definitions.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Warning: Running codema without a subcommand is deprecated. Use 'codema generate' instead.")
		generateCmd.Run(cmd, args)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(publishCmd)
}
