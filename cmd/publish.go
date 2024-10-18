package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/innovation-upstream/codema/internal/config"
	"github.com/spf13/cobra"
)

type PatternConfig struct {
	Label string `json:"label"`
}

type ManifestEntry struct {
	Type               string   `json:"type"`
	ContentPath        string   `json:"contentPath,omitempty"`
	ImportsPath        string   `json:"importsPath,omitempty"`
	HooksDirectory     string   `json:"hooksDirectory,omitempty"`
	HookFiles          []string `json:"hookFiles,omitempty"`
	ImplementationPath string   `json:"implementationPath,omitempty"`
}

type Manifest map[string]ManifestEntry

var publishCmd = &cobra.Command{
	Use:   "publish [patternLabel]@[version]",
	Short: "Publish a Codema pattern",
	Long:  `Publish a Codema pattern with the specified label and version.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version := args[0]
		err := publishPattern(version)
		if err != nil {
			fmt.Printf("Error publishing pattern: %v\n", err)
			os.Exit(1)
		}
	},
}

func publishPattern(version string) error {
	// Read the pattern label from codema-pattern.json
	patternLabel, err := readPatternLabel()
	if err != nil {
		return err
	}

	// Load configuration
	cfgLoader := config.NewStarlarkConfigLoader()
	cfg, err := cfgLoader.GetConfig()
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	// Create a buffer to store our archive
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Create manifest
	manifest := make(Manifest)

	// Pack function implementations and related files
	for _, api := range cfg.Apis {
		for _, ms := range api.Microservices {
			for _, funcImpl := range ms.FunctionImplementations {
				funcName := funcImpl.Function.Name
				implementationJSON, err := json.Marshal(funcImpl)
				if err != nil {
					return fmt.Errorf("error marshaling function implementation: %w", err)
				}

				// Add implementation JSON to zip
				implPath := fmt.Sprintf("%s/%s_implementation.json", ms.Label, funcName)
				err = addFileToZip(zipWriter, implPath, implementationJSON)
				if err != nil {
					return fmt.Errorf("error adding implementation JSON to zip: %w", err)
				}

				for target, snippets := range funcImpl.TargetSnippets {
					manifestEntry := ManifestEntry{
						Type:               "FunctionImplementation",
						ImplementationPath: implPath,
					}

					// Add content file
					if snippets.ContentPath != "" {
						contentPath := fmt.Sprintf("%s/%s/%s_content.template", ms.Label, target, funcName)
						err = addFileFromDiskToZip(zipWriter, cfg.TemplateDir+snippets.ContentPath, contentPath)
						if err != nil {
							return fmt.Errorf("error adding content file to zip: %w", err)
						}
						manifestEntry.ContentPath = contentPath
					}

					// Add imports file
					if snippets.ImportsPath != "" {
						importsPath := fmt.Sprintf("%s/%s/%s_imports.template", ms.Label, target, funcName)
						err = addFileFromDiskToZip(zipWriter, cfg.TemplateDir+snippets.ImportsPath, importsPath)
						if err != nil {
							return fmt.Errorf("error adding imports file to zip: %w", err)
						}
						manifestEntry.ImportsPath = importsPath
					}

					// Add hook files
					if snippets.HooksDirectory != "" {
						hooksDir := cfg.TemplateDir + snippets.HooksDirectory
						hookFiles, err := filepath.Glob(filepath.Join(hooksDir, "*.template"))
						if err != nil {
							return fmt.Errorf("error finding hook files: %w", err)
						}
						manifestEntry.HookFiles = make([]string, len(hookFiles))
						for i, hookFile := range hookFiles {
							hookFileName := filepath.Base(hookFile)
							zipPath := fmt.Sprintf("%s/%s/hooks/%s", ms.Label, target, hookFileName)
							err = addFileFromDiskToZip(zipWriter, hookFile, zipPath)
							if err != nil {
								return fmt.Errorf("error adding hook file to zip: %w", err)
							}
							manifestEntry.HookFiles[i] = zipPath
						}
						manifestEntry.HooksDirectory = fmt.Sprintf("%s/%s/hooks", ms.Label, target)
					}

					manifest[fmt.Sprintf("%s/%s/%s", ms.Label, target, funcName)] = manifestEntry
				}
			}
		}
	}

	// Add manifest to zip
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("error marshaling manifest: %w", err)
	}
	err = addFileToZip(zipWriter, "codema.manifest", manifestJSON)
	if err != nil {
		return fmt.Errorf("error adding manifest to zip: %w", err)
	}

	// Close the zip writer
	err = zipWriter.Close()
	if err != nil {
		return fmt.Errorf("error closing zip writer: %w", err)
	}

	// Create a new Codema client and publish the pattern
	client := NewCodemaClient()
	err = client.PublishPattern(patternLabel, version, buf.Bytes())
	if err != nil {
		return fmt.Errorf("error publishing pattern: %w", err)
	}

	fmt.Printf("Successfully published pattern '%s' version '%s'\n", patternLabel, version)
	return nil
}

func readPatternLabel() (string, error) {
	// Check if codema-pattern.json exists
	if _, err := os.Stat("codema-pattern.json"); os.IsNotExist(err) {
		return "", fmt.Errorf("codema-pattern.json not found. Please run 'codema init' to create it")
	}

	// Read and parse codema-pattern.json
	file, err := os.ReadFile("codema-pattern.json")
	if err != nil {
		return "", fmt.Errorf("error reading codema-pattern.json: %w", err)
	}

	var patternConfig PatternConfig
	err = json.Unmarshal(file, &patternConfig)
	if err != nil {
		return "", fmt.Errorf("error parsing codema-pattern.json: %w", err)
	}

	if patternConfig.Label == "" {
		return "", fmt.Errorf("pattern label is empty in codema-pattern.json")
	}

	return patternConfig.Label, nil
}

func addFileToZip(zipWriter *zip.Writer, filename string, content []byte) error {
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}
	_, err = writer.Write(content)
	return err
}

func addFileFromDiskToZip(zipWriter *zip.Writer, diskPath, zipPath string) error {
	fileContent, err := os.ReadFile(diskPath)
	if err != nil {
		return err
	}
	return addFileToZip(zipWriter, zipPath, fileContent)
}
