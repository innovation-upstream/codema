package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/innovation-upstream/codema/internal/config"
	"github.com/innovation-upstream/codema/internal/target"
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

func main() {
	var targetsRaw, configFormatRaw string
	flag.StringVar(&targetsRaw, "t", "*", "Targets to render")
	flag.StringVar(&configFormatRaw, "c", "yaml", "Config format. One of: yaml, starlark")
	flag.Parse()

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

	modulePath := config.ExpandModulePath(cfg.ModuleDir)
	templatesDir := config.ExpandTemplatePath(cfg.TemplateDir)

	apis := make(map[string]config.ApiDefinition)

	for _, a := range cfg.Apis {
		apis[a.Label] = a
	}

	fmt.Printf("Will render target(s): %s\n", logRenderTargets)
	for _, t := range cfg.Targets {
		if !isAllTargets {
			enabledByFlag := targetsToRender.Includes(t.Label)
			if !enabledByFlag {
				continue
			}

			renderedTargets = append(renderedTargets, t.Label)
		}
		ctrl := target.TargetProcessorController{
			ApiRegistry:  apis,
			ModulePath:   modulePath,
			ParentTarget: t,
			TemplatesDir: templatesDir,
		}

		for _, ta := range t.Apis {
			err := ctrl.ProcessTargetApi(ta)
			if err != nil {
				fmt.Printf("%+v", err)
				os.Exit(1)
			}
		}

		fmt.Printf("Rendered target: %s\n", t.Label)
	}

	if !isAllTargets && len(renderedTargets) != len(targetsToRender) {
		for _, tr := range targetsToRender {
			if !renderedTargets.Includes(tr) {
				fmt.Printf("WARN Skipped target: %s because it was not defined\n", tr)
			}
		}
	}
}

func getTemplateVersion(defaultVersion, version string) string {
	if version == "" {
		return defaultVersion
	} else {
		return version
	}
}
