package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/innovation-upstream/codema/internal/config"
	"github.com/innovation-upstream/codema/internal/plugin/goimports"
	"github.com/spf13/cobra"

	"github.com/innovation-upstream/codema/internal/model"
	"github.com/innovation-upstream/codema/internal/plugin"
	"github.com/innovation-upstream/codema/internal/tag"
	"github.com/innovation-upstream/codema/internal/target"
)

var (
	targetsRaw      string
	configFormatRaw string
)

type (
	TargetFlags []string
)

func (t TargetFlags) Includes(s string) bool {
	if t == nil || len(t) == 0 {
		return false
	}

	head := t[0]
	tail := t[1:]

	if string(head) == s {
		return true
	}

	return tail.Includes(s)
}

func (t TargetFlags) TrimSpace() TargetFlags {
	if t == nil || len(t) == 0 {
		return t
	}

	head := t[0]
	tail := t[1:]
	chunk := tail.TrimSpace()

	return append(chunk, strings.TrimSpace(head))
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate code based on API definitions",
	Long:  `Generate code for all or specific targets based on your API definitions.`,
	Run: func(cmd *cobra.Command, args []string) {
		isAllTargets := targetsRaw == "*"
		targetsToRender := TargetFlags(strings.Split(targetsRaw, ","))
		targetsToRender = targetsToRender.TrimSpace()

		renderedTargets := TargetFlags{}
		logRenderTargets := strings.Join([]string(targetsToRender), ", ")
		if isAllTargets {
			logRenderTargets = "ALL"
		}

		isYAML := configFormatRaw == "yaml"
		var cfgLoader config.ConfigLoader

		if isYAML {
			cfgLoader = config.NewYAMLConfigLoader()
		} else {
			cfgLoader = config.NewStarlarkConfigLoader()
		}

		cfg, err := cfgLoader.GetConfig()
		if err != nil {
			panic(err)
		}

		templatesDir := config.ExpandTemplatePath(cfg.TemplateDir)

		apis := make(map[string]config.ApiDefinition)

		tagReg := tag.NewTagRegistery(nil)
		modelReg := model.NewModelRegistery(nil)

		for _, a := range cfg.Apis {
			apis[a.Label] = a

			for _, ms := range a.Microservices {
				modelReg.RegisterModel(ms.PrimaryModel)

				for _, field := range ms.PrimaryModel.Fields {
					for _, tag := range field.Tags {
						tagReg.RegisterTag(tag)
					}
				}

				for _, model := range ms.SecondaryModels {
					modelReg.RegisterModel(model)

					for _, field := range model.Fields {
						for _, tag := range field.Tags {
							tagReg.RegisterTag(tag)
						}
					}
				}
			}
		}

		pluginRegistry := plugin.NewPluginRegistry()

		for _, t := range cfg.Targets {
			err := loadPluginsForTarget(pluginRegistry, t)
			if err != nil {
				slog.Error("Failed to load plugins for target " + t.Label + " " + err.Error())
				os.Exit(1)
			}
		}

		slog.Info("Will render target(s):", slog.String("targets", logRenderTargets))
		var totalFileCount int
		for _, t := range cfg.Targets {
			if !isAllTargets {
				enabledByFlag := targetsToRender.Includes(t.Label)
				if !enabledByFlag {
					continue
				}

				renderedTargets = append(renderedTargets, t.Label)
			}

			ctrl := target.TargetProcessorController{
				ApiRegistry:    apis,
				ParentTarget:   t,
				TemplatesDir:   templatesDir,
				PluginRegistry: pluginRegistry,
				TagRegistry:    tagReg,
				ModelRegistry:  modelReg,
			}

			var targetFileCount int
			for _, ta := range t.Apis {
				slog.Info("Rendering target for api", slog.String("target", t.Label), slog.String("api", ta.Label))
				fileCount, err := ctrl.ProcessTargetApi(ta)
				if err != nil {
					fmt.Printf("%+v", err)
					os.Exit(1)
				}

				targetFileCount += fileCount
				totalFileCount += fileCount
				slog.Info("Rendered target for api", slog.String("target", t.Label), slog.String("api", ta.Label), slog.Int("file_count", fileCount))
			}

			slog.Info("Rendered target", slog.String("target", t.Label), slog.Int("api_count", len(t.Apis)), slog.Int("file_count", targetFileCount))
		}

		if !isAllTargets && len(renderedTargets) != len(targetsToRender) {
			for _, tr := range targetsToRender {
				if !renderedTargets.Includes(tr) {
					fmt.Printf("WARN Skipped target: %s because it was not defined\n", tr)
				}
			}
		}

		slog.Info("Rendered files", slog.Int("file_count", totalFileCount))
	},
}

func init() {
	generateCmd.Flags().StringVarP(&targetsRaw, "targets", "t", "*", "Targets to render")
	generateCmd.Flags().StringVarP(&configFormatRaw, "config", "c", "yaml", "Config format. One of: yaml, starlark")
}

func loadPluginsForTarget(registry *plugin.PluginRegistry, t config.Target) error {
	for _, pluginName := range t.Plugins {
		var p plugin.Plugin
		switch pluginName {
		case "GoImports":
			p = &goimports.GoImportsPlugin{}
		// Add cases for other plugins here
		default:
			return fmt.Errorf("unknown plugin: %s", pluginName)
		}
		registry.Register(t.Label, p)
	}
	return nil
}
