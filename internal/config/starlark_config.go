package config

import (
	"os"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type starlarkConfigLoader struct{}

func NewStarlarkConfigLoader() ConfigLoader {
	return &starlarkConfigLoader{}
}

func (l *starlarkConfigLoader) GetConfig() (*Config, error) {
	// Load the Starlark file
	data, err := os.ReadFile("codema.star")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Initialize a Starlark thread and environment
	thread := &starlark.Thread{Name: "main"}
	globals, err := starlark.ExecFileOptions(syntax.LegacyFileOptions(), thread, "config.star", string(data), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Extract the configuration from the Starlark globals
	configVal, ok := globals["config"]
	if !ok {
		return nil, errors.New("config variable not defined in Starlark script")
	}

	// Convert Starlark value to Go structure
	var config Config
	if err := fillConfig(&config, configVal); err != nil {
		return nil, errors.WithStack(err)
	}

	return &config, nil
}

func fillConfig(c *Config, val starlark.Value) error {
	dict, ok := val.(*starlark.Dict)
	if !ok {
		return errors.New("expected a dictionary at the root of the Starlark config")
	}

	// Parsing moduleDir
	moduleDirVal, found, err := dict.Get(starlark.String("moduleDir"))
	if err != nil {
		return err
	}
	if found {
		moduleDirRaw, ok := moduleDirVal.(starlark.String)
		if !ok {
			return errors.New("moduleDir must be a string")
		}
		c.ModuleDir = moduleDirRaw.GoString()
	}

	// Parsing templateDir
	templateDirVal, found, err := dict.Get(starlark.String("templateDir"))
	if err != nil {
		return err
	}
	if found {
		tmlDirRaw, ok := templateDirVal.(starlark.String)
		if !ok {
			return errors.New("templateDir must be a string")
		}
		c.TemplateDir = tmlDirRaw.GoString()
	}

	// Parsing Apis
	apisVal, found, err := dict.Get(starlark.String("apis"))
	if err != nil {
		return err
	}
	if found {
		apisList, ok := apisVal.(*starlark.List)
		if !ok {
			return errors.New("apis must be a list")
		}
		for i := 0; i < apisList.Len(); i++ {
			apiItem := apisList.Index(i)
			apiDict, ok := apiItem.(*starlark.Dict)
			if !ok {
				return errors.New("each item in apis must be a dictionary")
			}
			var api ApiDefinition
			if err := parseApiDefinition(&api, apiDict); err != nil {
				return err
			}
			c.Apis = append(c.Apis, api)
		}
	}

	// Parsing Targets
	targetsVal, found, err := dict.Get(starlark.String("targets"))
	if err != nil {
		return err
	}
	if found {
		targetsList, ok := targetsVal.(*starlark.List)
		if !ok {
			return errors.New("targets must be a list")
		}
		for i := 0; i < targetsList.Len(); i++ {
			targetItem := targetsList.Index(i)
			targetDict, ok := targetItem.(*starlark.Dict)
			if !ok {
				return errors.New("each item in targets must be a dictionary")
			}
			var target Target
			if err := parseTarget(&target, targetDict); err != nil {
				return err
			}
			c.Targets = append(c.Targets, target)
		}
	}

	return nil
}

func parseApiDefinition(api *ApiDefinition, dict *starlark.Dict) error {
	// Helper function to get and check string fields
	getStringField := func(dict *starlark.Dict, key string) (string, error) {
		value, found, err := dict.Get(starlark.String(key))
		if err != nil {
			return "", err
		}
		if found {
			strValue, ok := value.(starlark.String)
			if !ok {
				return "", errors.Errorf("%s must be a string", key)
			}
			return string(strValue), nil
		}
		return "", nil // Return an empty string if not found
	}

	var err error
	if api.Package, err = getStringField(dict, "package"); err != nil {
		return err
	}

	if api.Label, err = getStringField(dict, "label"); err != nil {
		return err
	}

	al := api.Label
	apiLabelLowerCamel := strcase.ToLowerCamel(al)
	apiLabelKebab := strcase.ToKebab(al)
	apiLabelCamel := strcase.ToCamel(al)
	apiLabelScreaming := strings.ToUpper(al)
	apiLabelScreamingSnake := strcase.ToScreamingSnake(al)
	apiLabelSnake := strcase.ToSnake(al)

	api.LabelCamel = apiLabelCamel
	api.LabelLowerCamel = apiLabelLowerCamel
	api.LabelKebab = apiLabelKebab
	api.LabelScreaming = apiLabelScreaming
	api.LabelScreamingSnake = apiLabelScreamingSnake
	api.LabelSnake = apiLabelSnake

	// Parsing microservices array
	microservicesVal, found, err := dict.Get(starlark.String("microservices"))
	if err != nil {
		return err
	}
	if found {
		microservicesList, ok := microservicesVal.(*starlark.List)
		if !ok {
			return errors.New("microservices must be a list")
		}
		for i := 0; i < microservicesList.Len(); i++ {
			microItem := microservicesList.Index(i)
			microDict, ok := microItem.(*starlark.Dict)
			if !ok {
				return errors.New("each microservice must be a dictionary")
			}
			var micro MicroserviceDefinition
			if micro.Label, err = getStringField(microDict, "label"); err != nil {
				return err
			}

			// Additional labels derived from 'label'
			l := micro.Label
			labelLowerCamel := strcase.ToLowerCamel(l)
			labelKebab := strcase.ToKebab(l)
			labelCamel := strcase.ToCamel(l)
			labelScreaming := strings.ToUpper(l)
			labelScreamingSnake := strcase.ToScreamingSnake(l)
			labelSnake := strcase.ToSnake(l)

			micro.LabelKebab = labelKebab
			micro.LabelCamel = labelCamel
			micro.LabelLowerCamel = labelLowerCamel
			micro.LabelScreaming = labelScreaming
			micro.LabelScreamingSnake = labelScreamingSnake
			micro.LabelSnake = labelSnake

			api.Microservices = append(api.Microservices, micro)
		}
	}

	return nil
}

func parseTarget(target *Target, dict *starlark.Dict) error {
	// Helper function to get and check string fields
	getStringField := func(dict *starlark.Dict, key string) (string, error) {
		value, found, err := dict.Get(starlark.String(key))
		if err != nil {
			return "", err
		}
		if found {
			strValue, ok := value.(starlark.String)
			if !ok {
				return "", errors.Errorf("%s must be a string", key)
			}
			return string(strValue), nil
		}
		return "", nil // Return an empty string if not found
	}

	var err error
	if target.Label, err = getStringField(dict, "label"); err != nil {
		return err
	}
	if target.TemplateDir, err = getStringField(dict, "templateDir"); err != nil {
		return err
	}
	if target.TemplatePath, err = getStringField(dict, "templatePath"); err != nil {
		return err
	}
	if target.DefaultVersion, err = getStringField(dict, "defaultVersion"); err != nil {
		return err
	}
	eachVal, found, err := dict.Get(starlark.String("each"))
	if err != nil {
		return err
	}
	if found {
		eachBool, ok := eachVal.(starlark.Bool)
		if !ok {
			return errors.New("each must be a boolean")
		}
		target.Each = bool(eachBool)
	}

	// Parse apis
	apisVal, found, err := dict.Get(starlark.String("apis"))
	if err != nil {
		return err
	}
	if found {
		apisList, ok := apisVal.(*starlark.List)
		if !ok {
			return errors.New("apis must be a list")
		}
		for i := 0; i < apisList.Len(); i++ {
			apiItem := apisList.Index(i)
			apiDict, ok := apiItem.(*starlark.Dict)
			if !ok {
				return errors.New("each api must be a dictionary")
			}
			var api TargetApi
			if api.Label, err = getStringField(apiDict, "label"); err != nil {
				return err
			}
			if api.OutPath, err = getStringField(apiDict, "outPath"); err != nil {
				return err
			}
			if api.Version, err = getStringField(apiDict, "version"); err != nil {
				return err
			}
			// Handle skipLabels and args for each API
			skipLabelsVal, found, err := apiDict.Get(starlark.String("skipLabels"))
			if err != nil {
				return err
			}
			if found {
				skipLabelsList, ok := skipLabelsVal.(*starlark.List)
				if !ok {
					return errors.New("skipLabels must be a list")
				}
				api.SkipLabels = make([]string, skipLabelsList.Len())
				for j := 0; j < skipLabelsList.Len(); j++ {
					labelVal := skipLabelsList.Index(j)
					labelStr, ok := labelVal.(starlark.String)
					if !ok {
						return errors.New("each skipLabel must be a string")
					}
					api.SkipLabels[j] = string(labelStr)
				}
			}
			// Arguments would similarly be handled here (omitted for brevity)

			target.Apis = append(target.Apis, api)
		}
	}

	return nil
}
