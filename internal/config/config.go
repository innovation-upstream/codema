package config

import (
	"os"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type (
	FieldDirective map[string]interface{}

	FieldDefinition struct {
		Name        string                 `yaml:"name"`
		Type        string                 `yaml:"type"`
		Description string                 `yaml:"description"`
		Optional    bool                   `yaml:"optional"`
		Directives  map[string]interface{} `yaml:"directives"`
	}

	ModelDefinition struct {
		Name        string            `yaml:"name"`
		Fields      []FieldDefinition `yaml:"fields"`
		Description string            `yaml:"description"`
	}

	FunctionDefinition struct {
		Name        string   `yaml:"name"`
		Parameters  []string `yaml:"parameters"`
		Description string   `yaml:"description"`
	}

	FunctionImplementation struct {
		Function       FunctionDefinition `yaml:"function"`
		TargetSnippets map[string]string  `yaml:"target_snippets"`
	}

	MicroserviceDefinition struct {
		Label                   string                   `yaml:"label"`
		Models                  []ModelDefinition        `yaml:"models"`
		FunctionImplementations []FunctionImplementation `yaml:"function_implementations"`
		LabelKebab              string
		LabelCamel              string
		LabelLowerCamel         string
		LabelScreaming          string
		LabelScreamingSnake     string
		LabelSnake              string
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
		LabelSnake          string
	}

	TargetApi struct {
		Label      string   `yaml:"label"`
		OutPath    string   `yaml:"outPath"`
		Version    string   `yaml:"version"`
		SkipLabels []string `yaml:"skipLabels"`
	}

	Target struct {
		Label          string      `yaml:"label"`
		TemplatePath   string      `yaml:"templatePath"`
		TemplateDir    string      `yaml:"templateDir"`
		Apis           []TargetApi `yaml:"apis"`
		Each           bool        `yaml:"each"`
		DefaultVersion string      `yaml:"defaultVersion"`
	}

	Config struct {
		Apis        []ApiDefinition `yaml:"apis"`
		ModuleDir   string          `yaml:"moduleDir"`
		TemplateDir string          `yaml:"templateDir"`
		Targets     []Target        `yaml:"targets"`
	}
)

type (
	ConfigLoader interface {
		GetConfig() (*Config, error)
	}

	yamlConfigLoader struct{}
)

func NewYAMLConfigLoader() ConfigLoader {
	return &yamlConfigLoader{}
}

func (l *yamlConfigLoader) GetConfig() (*Config, error) {
	data, err := os.ReadFile("codema.yaml")
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
		apiLabelSnake := strcase.ToSnake(al)

		config.Apis[ax].LabelCamel = apiLabelCamel
		config.Apis[ax].LabelLowerCamel = apiLabelLowerCamel
		config.Apis[ax].LabelKebab = apiLabelKebab
		config.Apis[ax].LabelScreaming = apiLabelScreaming
		config.Apis[ax].LabelScreamingSnake = apiLabelScreamingSnake
		config.Apis[ax].LabelSnake = apiLabelSnake

		for ix, m := range a.Microservices {
			l := m.Label
			labelLowerCamel := strcase.ToLowerCamel(l)
			labelKebab := strcase.ToKebab(l)
			labelCamel := strcase.ToCamel(l)
			labelScreaming := strings.ToUpper(l)
			labelScreamingSnake := strcase.ToScreamingSnake(l)
			labelSnake := strcase.ToSnake(l)

			config.Apis[ax].Microservices[ix].LabelLowerCamel = labelLowerCamel
			config.Apis[ax].Microservices[ix].LabelCamel = labelCamel
			config.Apis[ax].Microservices[ix].LabelKebab = labelKebab
			config.Apis[ax].Microservices[ix].LabelScreaming = labelScreaming
			config.Apis[ax].Microservices[ix].LabelScreamingSnake = labelScreamingSnake
			config.Apis[ax].Microservices[ix].LabelSnake = labelSnake
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
