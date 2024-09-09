package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type starlarkConfigLoader struct {
	baseDir string
	cache   map[string]starlark.StringDict
}

func NewStarlarkConfigLoader() ConfigLoader {
	return &starlarkConfigLoader{
		baseDir: ".",
		cache:   make(map[string]starlark.StringDict),
	}
}

func (l *starlarkConfigLoader) GetConfig() (*Config, error) {
	// Load the main Starlark file
	globals, err := l.loadFile("codema.star")
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

func (l *starlarkConfigLoader) loadFile(filename string) (starlark.StringDict, error) {
	// Check if the file has already been loaded
	if globals, ok := l.cache[filename]; ok {
		return globals, nil
	}

	// Read the Starlark file
	data, err := os.ReadFile(filepath.Join(l.baseDir, filename))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Initialize a Starlark thread with a custom load function
	thread := &starlark.Thread{
		Name: filename,
		Load: l.load,
	}

	// Execute the Starlark file
	globals, err := starlark.ExecFileOptions(syntax.LegacyFileOptions(), thread, filename, data, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Cache the result
	l.cache[filename] = globals

	return globals, nil
}

func (l *starlarkConfigLoader) load(_ *starlark.Thread, module string) (starlark.StringDict, error) {
	return l.loadFile(module)
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

func getStringField(dict *starlark.Dict, key string) (string, error) {
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

func parseApiDefinition(api *ApiDefinition, dict *starlark.Dict) error {
	// Helper function to get and check string fields

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
			if err := parseMicroserviceDefinition(&micro, microDict); err != nil {
				return err
			}
			api.Microservices = append(api.Microservices, micro)
		}
	}

	return nil
}

func parseMicroserviceDefinition(micro *MicroserviceDefinition, dict *starlark.Dict) error {
	var err error
	if micro.Label, err = getStringField(dict, "label"); err != nil {
		return err
	}

	// Parse primary model
	primaryModelVal, found, err := dict.Get(starlark.String("primary_model"))
	if err != nil {
		return err
	}
	if found {
		primaryModelDict, ok := primaryModelVal.(*starlark.Dict)
		if !ok {
			return errors.New("primary_model must be a dictionary")
		}
		if err := parseModelDefinition(&micro.PrimaryModel, primaryModelDict); err != nil {
			return err
		}
	}

	// Parse secondary models
	secondaryModelsVal, found, err := dict.Get(starlark.String("secondary_models"))
	if err != nil {
		return err
	}
	if found {
		secondaryModelsList, ok := secondaryModelsVal.(*starlark.List)
		if !ok {
			return errors.New("secondary_models must be a list")
		}
		for i := 0; i < secondaryModelsList.Len(); i++ {
			secondaryModelItem := secondaryModelsList.Index(i)
			secondaryModelDict, ok := secondaryModelItem.(*starlark.Dict)
			if !ok {
				return errors.New("each secondary model must be a dictionary")
			}
			var secondaryModel SecondaryModel
			if err := parseSecondaryModel(&secondaryModel, secondaryModelDict); err != nil {
				return err
			}
			micro.SecondaryModels = append(micro.SecondaryModels, secondaryModel)
		}
	}

	// Parse function implementations
	funcImplVal, found, err := dict.Get(starlark.String("function_implementations"))
	if err != nil {
		return err
	}
	if found {
		funcImplList, ok := funcImplVal.(*starlark.List)
		if !ok {
			return errors.New("function_implementations must be a list")
		}
		for i := 0; i < funcImplList.Len(); i++ {
			funcImplItem := funcImplList.Index(i)
			funcImplDict, ok := funcImplItem.(*starlark.Dict)
			if !ok {
				return errors.New("each function implementation must be a dictionary")
			}
			var funcImpl FunctionImplementation
			if err := parseFunctionImplementation(&funcImpl, funcImplDict); err != nil {
				return err
			}
			micro.FunctionImplementations = append(micro.FunctionImplementations, funcImpl)
		}
	}

	// Additional labels
	l := micro.Label
	micro.LabelKebab = strcase.ToKebab(l)
	micro.LabelCamel = strcase.ToCamel(l)
	micro.LabelLowerCamel = strcase.ToLowerCamel(l)
	micro.LabelScreaming = strings.ToUpper(l)
	micro.LabelScreamingSnake = strcase.ToScreamingSnake(l)
	micro.LabelSnake = strcase.ToSnake(l)

	return nil
}

func parseSecondaryModel(secondaryModel *SecondaryModel, dict *starlark.Dict) error {
	modelVal, found, err := dict.Get(starlark.String("model"))
	if err != nil {
		return err
	}
	if !found {
		return errors.New("model is required in secondary model")
	}
	modelDict, ok := modelVal.(*starlark.Dict)
	if !ok {
		return errors.New("model must be a dictionary")
	}
	if err := parseModelDefinition(&secondaryModel.Model, modelDict); err != nil {
		return err
	}

	typeVal, found, err := dict.Get(starlark.String("type"))
	if err != nil {
		return err
	}
	if found {
		typeStr, ok := typeVal.(starlark.String)
		if !ok {
			return errors.New("type must be a string")
		}
		secondaryModel.Type = SecondaryModelType(typeStr)
	} else {
		secondaryModel.Type = SecondaryModelTypeUnspecified
	}

	return nil
}

func parseModelDefinition(model *ModelDefinition, dict *starlark.Dict) error {
	var err error
	if model.Name, err = getStringField(dict, "name"); err != nil {
		return err
	}

	model.NameKebab = strcase.ToKebab(model.Name)
	model.NameCamel = strcase.ToCamel(model.Name)
	model.NameLowerCamel = strcase.ToLowerCamel(model.Name)
	model.NameScreaming = strcase.ToScreamingSnake(model.Name)
	model.NameScreamingSnake = model.NameScreaming
	model.NameSnake = strcase.ToSnake(model.Name)

	if model.Description, err = getStringField(dict, "description"); err != nil {
		return err
	}

	// Parse enums
	enumsVal, found, err := dict.Get(starlark.String("enums"))
	if err != nil {
		return err
	}
	if found {
		enumsList, ok := enumsVal.(*starlark.List)
		if !ok {
			return errors.New("enums must be a list")
		}
		for i := 0; i < enumsList.Len(); i++ {
			enumItem := enumsList.Index(i)
			enumDict, ok := enumItem.(*starlark.Dict)
			if !ok {
				return errors.New("each enum must be a dictionary")
			}
			var enum EnumDefinition
			if err := parseEnumDefinition(&enum, enumDict); err != nil {
				return err
			}
			model.Enums = append(model.Enums, enum)
		}
	}

	// Parse fields
	fieldsVal, found, err := dict.Get(starlark.String("fields"))
	if err != nil {
		return err
	}
	if found {
		fieldsList, ok := fieldsVal.(*starlark.List)
		if !ok {
			return errors.New("fields must be a list")
		}
		for i := 0; i < fieldsList.Len(); i++ {
			fieldItem := fieldsList.Index(i)
			fieldDict, ok := fieldItem.(*starlark.Dict)
			if !ok {
				return errors.New("each field must be a dictionary")
			}
			var field FieldDefinition
			if err := parseFieldDefinition(&field, fieldDict, model.Enums); err != nil {
				return err
			}
			model.Fields = append(model.Fields, field)
		}
	}

	return nil
}

func parseEnumDefinition(enum *EnumDefinition, dict *starlark.Dict) error {
	var err error
	if enum.Name, err = getStringField(dict, "name"); err != nil {
		return err
	}
	if enum.Description, err = getStringField(dict, "description"); err != nil {
		return err
	}

	// Parse values
	valuesVal, found, err := dict.Get(starlark.String("values"))
	if err != nil {
		return err
	}
	if found {
		valuesList, ok := valuesVal.(*starlark.List)
		if !ok {
			return errors.New("enum values must be a list")
		}
		for i := 0; i < valuesList.Len(); i++ {
			valueItem := valuesList.Index(i)
			valueStr, ok := valueItem.(starlark.String)
			if !ok {
				return errors.New("each enum value must be a string")
			}
			enum.Values = append(enum.Values, string(valueStr))
		}
	}

	return nil
}

func parseFieldDefinition(field *FieldDefinition, dict *starlark.Dict, enums []EnumDefinition) error {
	var err error
	if field.Name, err = getStringField(dict, "name"); err != nil {
		return err
	}

	field.NameKebab = strcase.ToKebab(field.Name)
	field.NameCamel = strcase.ToCamel(field.Name)
	field.NameLowerCamel = strcase.ToLowerCamel(field.Name)
	field.NameScreaming = strcase.ToScreamingSnake(field.Name)
	field.NameScreamingSnake = field.NameScreaming
	field.NameSnake = strcase.ToSnake(field.Name)

	if field.Type, err = getStringField(dict, "type"); err != nil {
		return err
	}
	if err := validateFieldType(field.Type, enums); err != nil {
		return errors.Wrapf(err, "invalid field type for %s", field.Name)
	}
	if field.Description, err = getStringField(dict, "description"); err != nil {
		return err
	}

	optionalVal, found, err := dict.Get(starlark.String("optional"))
	if err != nil {
		return err
	}
	if found {
		optionalBool, ok := optionalVal.(starlark.Bool)
		if !ok {
			return errors.New("optional must be a boolean")
		}
		field.Optional = bool(optionalBool)
	}

	// Parse directives
	directivesVal, found, err := dict.Get(starlark.String("directives"))
	if err != nil {
		return err
	}
	if found {
		directivesDict, ok := directivesVal.(*starlark.Dict)
		if !ok {
			return errors.New("directives must be a dictionary")
		}
		field.Directives = make(map[string]interface{})
		for _, item := range directivesDict.Items() {
			key, value := item[0].(starlark.String), item[1]
			field.Directives[string(key)] = starlarkValueToGo(value)
		}
	}

	tagsVal, found, err := dict.Get(starlark.String("tags"))
	if err != nil {
		return err
	}
	if found {
		tagsList, ok := tagsVal.(*starlark.List)
		if !ok {
			return errors.New("tags must be a list")
		}
		for i := 0; i < tagsList.Len(); i++ {
			tagItem := tagsList.Index(i)
			tagDict, ok := tagItem.(*starlark.Dict)
			if !ok {
				return errors.New("each tag must be a dictionary")
			}
			var tag TagDefinition
			if err := parseTagDefinition(&tag, tagDict); err != nil {
				return err
			}
			field.Tags = append(field.Tags, tag)
		}
	}

	return nil
}

func parseTagDefinition(tag *TagDefinition, dict *starlark.Dict) error {
	var err error
	if tag.Name, err = getStringField(dict, "name"); err != nil {
		return err
	}

	typeVal, found, err := dict.Get(starlark.String("type"))
	if err != nil {
		return err
	}
	if found {
		typeStr, ok := typeVal.(starlark.String)
		if !ok {
			return errors.New("tag type must be a string")
		}
		tag.Type = TagType(typeStr)
	} else {
		tag.Type = TagTypeUnspecified
	}

	return nil
}

func parseFunctionDefinition(function *FunctionDefinition, dict *starlark.Dict) error {
	var err error
	if function.Name, err = getStringField(dict, "name"); err != nil {
		return err
	}
	if function.Description, err = getStringField(dict, "description"); err != nil {
		return err
	}

	// Parse parameters
	parametersVal, found, err := dict.Get(starlark.String("parameters"))
	if err != nil {
		return err
	}
	if found {
		parametersList, ok := parametersVal.(*starlark.List)
		if !ok {
			return errors.New("parameters must be a list")
		}
		for i := 0; i < parametersList.Len(); i++ {
			paramItem := parametersList.Index(i)
			paramStr, ok := paramItem.(starlark.String)
			if !ok {
				return errors.New("each parameter must be a string")
			}
			function.Parameters = append(function.Parameters, string(paramStr))
		}
	}

	return nil
}

func parseFunctionImplementation(funcImpl *FunctionImplementation, dict *starlark.Dict) error {
	functionVal, found, err := dict.Get(starlark.String("function"))
	if err != nil {
		return err
	}
	if !found {
		return errors.New("function is required in function implementation")
	}
	functionDict, ok := functionVal.(*starlark.Dict)
	if !ok {
		return errors.New("function must be a dictionary")
	}
	if err := parseFunctionDefinition(&funcImpl.Function, functionDict); err != nil {
		return err
	}

	targetSnippetsVal, found, err := dict.Get(starlark.String("target_snippets"))
	if err != nil {
		return err
	}
	if found {
		targetSnippetsDict, ok := targetSnippetsVal.(*starlark.Dict)
		if !ok {
			return errors.New("target_snippets must be a dictionary")
		}
		funcImpl.TargetSnippets = make(map[string]SnippetPaths)
		for _, item := range targetSnippetsDict.Items() {
			key, value := item[0].(starlark.String), item[1].(*starlark.Dict)
			contentPath, _, _ := value.Get(starlark.String("content_path"))
			importsPath, _, _ := value.Get(starlark.String("imports_path"))
			funcImpl.TargetSnippets[string(key)] = SnippetPaths{
				ContentPath: contentPath.(starlark.String).GoString(),
				ImportsPath: importsPath.(starlark.String).GoString(),
			}
		}
	}

	return nil
}

func starlarkValueToGo(v starlark.Value) interface{} {
	switch v := v.(type) {
	case starlark.Bool:
		return bool(v)
	case starlark.Int:
		i, _ := v.Int64()
		return i
	case starlark.Float:
		return float64(v)
	case starlark.String:
		return string(v)
	case *starlark.List:
		result := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = starlarkValueToGo(v.Index(i))
		}
		return result
	case *starlark.Dict:
		result := make(map[string]interface{})
		for _, item := range v.Items() {
			key, value := item[0].(starlark.String), item[1]
			result[string(key)] = starlarkValueToGo(value)
		}
		return result
	default:
		return nil
	}
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

	pluginsVal, found, err := dict.Get(starlark.String("plugins"))
	if err != nil {
		return errors.WithStack(err)
	}
	if found {
		pluginsList, ok := pluginsVal.(*starlark.List)
		if !ok {
			return errors.New("plugins must be a list")
		}
		target.Plugins = make([]string, pluginsList.Len())
		for i := 0; i < pluginsList.Len(); i++ {
			pluginVal := pluginsList.Index(i)
			pluginStr, ok := pluginVal.(starlark.String)
			if !ok {
				return errors.New("each plugin must be a string")
			}
			target.Plugins[i] = string(pluginStr)
		}
	}

	// Parse options
	optionsVal, found, err := dict.Get(starlark.String("options"))
	if err != nil {
		return errors.WithStack(err)
	}
	if found {
		optionsDict, ok := optionsVal.(*starlark.Dict)
		if !ok {
			return errors.New("options must be a dictionary")
		}
		fileModeVal, found, err := optionsDict.Get(starlark.String("fileMode"))
		if err != nil {
			return errors.WithStack(err)
		}
		if found {
			fileModeInt, ok := fileModeVal.(starlark.Int)
			if !ok {
				return errors.New("fileMode must be an integer")
			}
			fileModeRaw := "0" + fileModeInt.String()

			// Parse the string as a base-8 (octal) integer
			fileMode, err := strconv.ParseUint(fileModeRaw, 8, 32)
			if err != nil {
				return errors.WithStack(err)
			}

			target.Options.FileMode = os.FileMode(fileMode)
		}
	}

	target.setDefaultOptions()

	return nil
}

func (t *Target) setDefaultOptions() {
}
