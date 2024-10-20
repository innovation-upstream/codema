package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type (
	FieldDirective map[string]interface{}

	TagType string

	TagDefinition struct {
		Name string
		Type TagType
	}

	FieldDefinition struct {
		Name               string
		NameKebab          string
		NameCamel          string
		NameLowerCamel     string
		NameScreaming      string
		NameScreamingSnake string
		NameSnake          string
		Type               string
		Description        string
		Optional           bool
		Directives         map[string]interface{}
		Tags               []TagDefinition
	}

	EnumDefinition struct {
		Name        string   `yaml:"name"`
		Values      []string `yaml:"values"`
		Description string   `yaml:"description"`
	}

	ModelDefinition struct {
		Name               string `yaml:"name"`
		NameKebab          string
		NameCamel          string
		NameLowerCamel     string
		NameScreaming      string
		NameScreamingSnake string
		NameSnake          string
		Fields             []FieldDefinition `yaml:"fields"`
		Enums              []EnumDefinition  `yaml:"enums"`
		Description        string            `yaml:"description"`
		Directives         map[string]interface{}
	}

	FunctionDefinition struct {
		Name        string   `yaml:"name"`
		Parameters  []string `yaml:"parameters"`
		Description string   `yaml:"description"`
	}

	SnippetPaths struct {
		ContentPath    string
		ImportsPath    string
		HooksDirectory string
	}

	FunctionImplementation struct {
		Function       FunctionDefinition
		TargetSnippets map[string]SnippetPaths
	}

	MicroserviceDefinition struct {
		Label                   string
		PrimaryModel            ModelDefinition
		SecondaryModels         []ModelDefinition
		FunctionImplementations []FunctionImplementation
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
		Label   string `yaml:"label"`
		OutPath string `yaml:"outPath"`
		// Deprecated. Use VersionPath
		Version     string `yaml:"version"`
		VersionPath string
		SkipLabels  []string `yaml:"skipLabels"`
	}

	TargetOptions struct {
		FileMode os.FileMode `yaml:"fileMode"`
	}

	Target struct {
		Label        string      `yaml:"label"`
		TemplatePath string      `yaml:"templatePath"`
		TemplateDir  string      `yaml:"templateDir"`
		Apis         []TargetApi `yaml:"apis"`
		Each         bool        `yaml:"each"`
		// Deprecated. Use DefaultVersionPath
		DefaultVersion     string `yaml:"defaultVersion"`
		DefaultVersionPath string
		Plugins            []string      `yaml:"plugins"`
		Options            TargetOptions `yaml:"options"`
	}

	Config struct {
		Apis        []ApiDefinition `yaml:"apis"`
		TemplateDir string          `yaml:"templateDir"`
		Targets     []Target        `yaml:"targets"`
	}
)

const (
	TagTypeOwner       TagType = "OWNER"
	TagTypeParent      TagType = "PARENT"
	TagTypeUnspecified TagType = "UNSPECIFIED"
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

	for tx, t := range config.Targets {
		if t.DefaultVersionPath == "" {
			config.Targets[tx].DefaultVersionPath = t.DefaultVersion
		}

		for tax, ta := range t.Apis {
			if ta.VersionPath == "" {
				config.Targets[tx].Apis[tax].VersionPath = ta.Version
			}
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

func IsPrimitiveFieldType(fieldType string) bool {
	validTypes := map[string]bool{
		"Int": true, "Float": true, "String": true, "Boolean": true, "ID": true, "DateTime": true,
	}

	if validTypes[fieldType] {
		return true
	}

	if strings.HasPrefix(fieldType, "[") && strings.HasSuffix(fieldType, "]") {
		innerType := fieldType[1 : len(fieldType)-1]
		return IsPrimitiveFieldType(innerType)
	}

	return false
}

func (f FieldDefinition) GetDirectiveStringValue(name string) string {
	d := f.Directives[name]
	if d == nil {
		return ""
	}

	switch d.(type) {
	case string:
		return d.(string)
	default:
		slog.Warn("Unsupported directive value type", slog.String("value", fmt.Sprintf("%+v", d)))
	}

	return ""
}

func (f ModelDefinition) GetDirectiveStringValue(name string) string {
	d := f.Directives[name]
	if d == nil {
		return ""
	}

	switch d.(type) {
	case string:
		return d.(string)
	default:
		slog.Warn("Unsupported directive value type", slog.String("value", fmt.Sprintf("%+v", d)))
	}

	return ""
}

func (f ModelDefinition) GetDirectiveListValue(name string) []interface{} {
	d := f.Directives[name]
	if d == nil {
		return nil
	}

	switch d.(type) {
	case []interface{}:
		return d.([]interface{})
	default:
		slog.Warn("Unsupported directive value type", slog.String("value", fmt.Sprintf("%+v", d)))
	}

	return nil
}
