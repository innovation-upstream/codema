package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/innovation-upstream/codema/internal/config"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull [patternLabel]",
	Short: "Pull pattern updates",
	Long:  `Pull a specific pattern or all patterns if no label is provided.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		patternLabel := args[0]
		err := pullPattern(patternLabel)
		if err != nil {
			fmt.Printf("Error pulling pattern: %v\n", err)
			os.Exit(1)
		}
	},
}

func pullPattern(patternLabel string) error {
	client := NewCodemaClient()
	body, err := client.PullPattern(patternLabel)
	if err != nil {
		return err
	}

	// Create a reader from the response body
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("error creating zip reader: %w", err)
	}

	// Create the cache directory if it doesn't exist
	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "codema", "pattern", patternLabel)
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating cache directory: %w", err)
	}

	// Read and parse the manifest
	manifest, err := readManifest(zipReader)
	if err != nil {
		return fmt.Errorf("error reading manifest: %w", err)
	}

	// Extract the files according to the manifest
	for _, file := range zipReader.File {
		outPath := filepath.Join(cacheDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(outPath, file.Mode())
			continue
		}

		err := extractFile(file, outPath)
		if err != nil {
			return fmt.Errorf("error extracting file %s: %w", file.Name, err)
		}
	}

	// Process the manifest and organize files
	for key, entry := range manifest {
		switch entry.Type {
		case "FunctionImplementation":
			err := processFunctionImplementation(cacheDir, key, entry)
			if err != nil {
				return fmt.Errorf("error processing function implementation %s: %w", key, err)
			}
		}
	}

	fmt.Printf("Pattern '%s' pulled successfully and stored in %s\n", patternLabel, cacheDir)
	return nil
}

func readManifest(zipReader *zip.Reader) (Manifest, error) {
	for _, file := range zipReader.File {
		if file.Name == "codema.manifest" {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			var manifest Manifest
			err = json.NewDecoder(rc).Decode(&manifest)
			if err != nil {
				return nil, err
			}

			return manifest, nil
		}
	}

	return nil, fmt.Errorf("manifest file not found in archive")
}

func extractFile(file *zip.File, outPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

func processFunctionImplementation(cacheDir, key string, entry ManifestEntry) error {
	// Read the implementation JSON
	implPath := filepath.Join(cacheDir, entry.ImplementationPath)
	implData, err := os.ReadFile(implPath)
	if err != nil {
		return fmt.Errorf("error reading implementation file: %w", err)
	}

	var funcImpl config.FunctionImplementation
	err = json.Unmarshal(implData, &funcImpl)
	if err != nil {
		return fmt.Errorf("error unmarshaling implementation data: %w", err)
	}

	// Create a new map for updated TargetSnippets
	updatedTargetSnippets := make(map[string]config.SnippetPaths)

	// Update the TargetSnippets with the correct paths
	for target, snippets := range funcImpl.TargetSnippets {
		updatedSnippets := snippets // Create a copy of the snippets

		if entry.ContentPath != "" {
			updatedSnippets.ContentPath = filepath.Join(cacheDir, entry.ContentPath)
		}
		if entry.ImportsPath != "" {
			updatedSnippets.ImportsPath = filepath.Join(cacheDir, entry.ImportsPath)
		}
		if entry.HooksDirectory != "" {
			updatedSnippets.HooksDirectory = filepath.Join(cacheDir, entry.HooksDirectory)
		}

		updatedTargetSnippets[target] = updatedSnippets
	}

	// Replace the old TargetSnippets with the updated one
	funcImpl.TargetSnippets = updatedTargetSnippets

	// Write the updated implementation JSON back to disk
	updatedImplData, err := json.MarshalIndent(funcImpl, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling updated implementation data: %w", err)
	}

	err = os.WriteFile(implPath, updatedImplData, 0644)
	if err != nil {
		return fmt.Errorf("error writing updated implementation file: %w", err)
	}

	return nil
}
