package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [patternLabel]",
	Short: "Initialize a new Codema pattern",
	Long:  `Initialize a new Codema pattern by creating a codema-pattern.json file with the specified label.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		patternLabel := args[0]
		err := initializePattern(patternLabel)
		if err != nil {
			fmt.Printf("Error initializing pattern: %v\n", err)
			os.Exit(1)
		}
	},
}

func initializePattern(patternLabel string) error {
	patternConfig := struct {
		Label string `json:"label"`
	}{
		Label: patternLabel,
	}

	jsonData, err := json.MarshalIndent(patternConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	err = os.WriteFile("codema-pattern.json", jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Printf("Initialized Codema pattern '%s'. Created codema-pattern.json file.\n", patternLabel)
	return nil
}
