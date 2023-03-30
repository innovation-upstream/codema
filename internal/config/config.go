package config

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type (
	MicroserviceDefinition struct {
		Label               string `yaml:"label"`
		LabelKebab          string
		LabelCamel          string
		LabelLowerCamel     string
		LabelScreaming      string
		LabelScreamingSnake string
	}

	ApiDefinition struct {
		Microservices       []MicroserviceDefinition `yaml:"microservices"`
		Package             string                   `yaml:"package"`
		Label               string                   `yaml:"label"`
		LabelKebab          string
		LabelCamel          string
		LabelLowerCamel     string
		LabelScreaming      string
		LabelScreamingSnake string
	}

	TargetApiArgs map[string]map[string]map[string]string

	TargetApi struct {
		Label      string        `yaml:"label"`
		OutPath    string        `yaml:"outPath"`
		Version    string        `yaml:"version"`
		SkipLabels []string      `yaml:"skipLabels"`
		Args       TargetApiArgs `yaml:"args"`
	}

	Target struct {
		Label string `yaml:"label"`
		// TemplatePath is deprecated, use TemplateDir instead
		TemplatePath string      `yaml:"templatePath"`
		TemplateDir  string      `yaml:"templateDir"`
		Apis         []TargetApi `yaml:"apis"`
		Each         bool        `yaml:"each"`
		// DefaultVerion is the default version tag to use for apis that render this target
		DefaultVerion string `yaml:"defaultVersion"`
	}

	Config struct {
		Apis        []ApiDefinition `yaml:"apis"`
		ModuleDir   string          `yaml:"moduleDir"`
		TemplateDir string          `yaml:"templateDir"`
		Targets     []Target        `yaml:"targets"`
	}
)

func GetConfig() (*Config, error) {
	data, err := ioutil.ReadFile("codema.yaml")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for ax, a := range config.Apis {
		al := a.Label
		apiLabelLowerCamel := strcase.ToLowerCamel(al)
		apiLabelKebab := strcase.ToKebab(al)
		apiLabelCamel := strcase.ToCamel(al)
		apiLabelScreaming := strings.ToUpper(al)
		apiLabelScreamingSnake := strcase.ToScreamingSnake(al)

		config.Apis[ax].LabelCamel = apiLabelCamel
		config.Apis[ax].LabelLowerCamel = apiLabelLowerCamel
		config.Apis[ax].LabelKebab = apiLabelKebab
		config.Apis[ax].LabelScreaming = apiLabelScreaming
		config.Apis[ax].LabelScreamingSnake = apiLabelScreamingSnake

		for ix, m := range a.Microservices {
			l := m.Label
			labelLowerCamel := strcase.ToLowerCamel(l)
			labelKebab := strcase.ToKebab(l)
			labelCamel := strcase.ToCamel(l)
			labelScreaming := strings.ToUpper(l)
			labelScreamingSnake := strcase.ToScreamingSnake(l)

			config.Apis[ax].Microservices[ix].LabelLowerCamel = labelLowerCamel
			config.Apis[ax].Microservices[ix].LabelCamel = labelCamel
			config.Apis[ax].Microservices[ix].LabelKebab = labelKebab
			config.Apis[ax].Microservices[ix].LabelScreaming = labelScreaming
			config.Apis[ax].Microservices[ix].LabelScreamingSnake = labelScreamingSnake
		}
	}

	return &config, nil
}

func ExpandModulePath(modulePathRaw string) string {
	modulePath := os.ExpandEnv(
		strings.ReplaceAll(modulePathRaw, "~", "$HOME"),
	)

	return modulePath
}

func ExpandTemplatePath(templatePathRaw string) string {
	templatePath := os.ExpandEnv(
		strings.ReplaceAll(templatePathRaw, "~", "$HOME"),
	)

	return templatePath
}
