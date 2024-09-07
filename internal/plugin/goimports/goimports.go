package goimports

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/tools/imports"
)

type GoImportsPlugin struct{}

func (p *GoImportsPlugin) Name() string {
	return "GoImports"
}

func (p *GoImportsPlugin) PreWriteFile(ctx context.Context, filename string, content []byte) ([]byte, error) {
	// Get the directory of the file
	dir := filepath.Dir(filename)

	// Get the current working directory
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current working directory")
	}

	// Change to the directory of the file
	err = os.Chdir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to change to file directory")
	}

	// Ensure we change back to the original directory when we're done
	defer func() {
		err := os.Chdir(originalDir)
		if err != nil {
			// Log the error, but don't return it as the main operation has already completed
			// You might want to use a proper logger here
			slog.Warn("Warning: failed to change back to original directory", slog.String("error", err.Error()))
		}
	}()

	// Run goimports
	opts := &imports.Options{
		TabWidth:  4,
		TabIndent: true,
		Comments:  true,
		Fragment:  true,
	}

	processedContent, err := imports.Process(filepath.Base(filename), content, opts)
	if err != nil {
		return nil, errors.Wrap(err, "goimports processing failed")
	}

	return processedContent, nil
}
