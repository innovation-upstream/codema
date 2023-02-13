package main

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type (
	OutFile struct {
		Path         string   `yaml:"path"`
		TemplatePath string   `yaml:"templatePath"`
		Each         bool     `yaml:"each"`
		SkipLabels   []string `yaml:"skipLabels"`
	}

	Microservice struct {
		Label               string `yaml:"label"`
		LabelKebab          string
		LabelCamel          string
		LabelLowerCamel     string
		LabelScreaming      string
		LabelScreamingSnake string
	}

	Api struct {
		Files               []OutFile      `yaml:"files"`
		Microservices       []Microservice `yaml:"microservices"`
		Package             string         `yaml:"package"`
		Label               string         `yaml:"label"`
		LabelKebab          string
		LabelCamel          string
		LabelLowerCamel     string
		LabelScreaming      string
		LabelScreamingSnake string
	}

	Config struct {
		Apis        []Api  `yaml:"apis"`
		ModuleDir   string `yaml:"moduleDir"`
		TemplateDir string `yaml:"templateDir"`
	}
)

func getConfig() (*Config, error) {
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

func expandModulePath(modulePathRaw string) string {
	modulePath := os.ExpandEnv(
		strings.ReplaceAll(modulePathRaw, "~", "$HOME"),
	)

	return modulePath
}

func expandTemplatePath(templatePathRaw string) string {
	templatePath := os.ExpandEnv(
		strings.ReplaceAll(templatePathRaw, "~", "$HOME"),
	)

	return templatePath
}
