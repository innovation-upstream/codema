package target

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	goTmpl "text/template"

	"github.com/iancoleman/strcase"
	"github.com/innovation-upstream/codema/internal/config"
	"github.com/innovation-upstream/codema/internal/directive"
	"github.com/innovation-upstream/codema/internal/fs"
	"github.com/innovation-upstream/codema/internal/model"
	"github.com/innovation-upstream/codema/internal/plugin"
	"github.com/innovation-upstream/codema/internal/tag"
	"github.com/innovation-upstream/codema/internal/template"
	"github.com/pkg/errors"
)

type (
	TargetProcessorController struct {
		ApiRegistry    map[string]config.ApiDefinition
		ParentTarget   config.Target
		TemplatesDir   string
		PluginRegistry *plugin.PluginRegistry
		TagRegistry    tag.TagRegistry
		ModelRegistry  model.ModelRegistry
	}

	TargetProcessor struct {
		Api          config.ApiDefinition
		ParentTarget config.Target
		TemplatesDir string
	}
)

func (ctrl *TargetProcessorController) ProcessTargetApi(ta config.TargetApi) (int, error) {
	a, ok := ctrl.ApiRegistry[ta.Label]
	if !ok {
		msg := fmt.Sprintf("Could not find api: %s", ta.Label)
		return 0, errors.New(msg)
	}

	pathTmplStr, err := template.NewPathTemplateString(ta.OutPath)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	tp := TargetProcessor{
		Api:          a,
		ParentTarget: ctrl.ParentTarget,
		TemplatesDir: ctrl.TemplatesDir,
	}

	var numFiles int
	if ctrl.ParentTarget.Each {
	msLoop:
		for _, m := range a.Microservices {
			for _, sl := range ta.SkipLabels {
				if sl == m.Label {
					continue msLoop
				}
			}

			msOutFileSubPath, err := pathTmplStr.ExecuteMicroservicePathTemplate(template.MicroservicePathTemplateInput{
				Api:          a,
				Microservice: m,
				Label:        a.Label,
			})
			if err != nil {
				return 0, errors.WithStack(err)
			}

			msOutFilePath := config.ExpandModulePath(msOutFileSubPath)

			targetTmplRaw, err := tp.getRawTemplate(ta, ctrl.ParentTarget, msOutFilePath)
			if err != nil {
				return 0, errors.WithStack(err)
			}

			err = ctrl.renderEachFile(msOutFilePath, targetTmplRaw, a, m)
			if err != nil {
				return 0, errors.WithStack(err)
			}

			numFiles++
		}
	} else {
		apiOutFileSubPath, err := pathTmplStr.ExecuteApiPathTemplate(template.ApiPathTemplateInput{
			Api:   a,
			Label: a.Label,
		})
		if err != nil {
			return 0, errors.WithStack(err)
		}

		apiOutFilePath := config.ExpandModulePath(apiOutFileSubPath)

		targetTmplRaw, err := tp.getRawTemplate(ta, ctrl.ParentTarget, apiOutFilePath)
		if err != nil {
			return 0, errors.WithStack(err)
		}

		err = ctrl.renderSingleFile(apiOutFilePath, targetTmplRaw, a)
		if err != nil {
			return 0, errors.WithStack(err)
		}

		numFiles++
	}

	return numFiles, nil
}

func getTemplateVersion(defaultVersion, version string) string {
	if version == "" {
		return defaultVersion
	} else {
		return version
	}
}

func (tp *TargetProcessor) getRawTemplate(
	ta config.TargetApi,
	parentTarget config.Target,
	path string,
) (string, error) {
	templateVersion := getTemplateVersion(tp.ParentTarget.DefaultVersion, ta.Version)
	var tmplPath string
	if tp.ParentTarget.TemplateDir == "" {
		tmplPath = fs.GetLegacyTemplatePath(tp.TemplatesDir, tp.ParentTarget.TemplatePath)
	} else if templateVersion != "" {
		tmplPath = fs.GetTemplatePath(tp.TemplatesDir, tp.ParentTarget.TemplateDir, templateVersion)
	} else {
		desc := "You specified templateDir without specifing a template version!  You must specify either Target.DefaultVersion or a TargetApi.Version"
		msg := fmt.Sprintf(
			"Failed to render target: %s for api: %s. Message: %s",
			tp.ParentTarget.Label,
			ta.Label,
			desc,
		)
		err := errors.New(msg)
		panic(err)
	}

	isDir, err := fs.IsDir(path)
	if err != nil {
		panic(err)
	}

	if isDir {
		panic(fmt.Sprintf("ERROR: %s is a directory, aborting", path))
	}

	tmplRaw, err := os.ReadFile(tmplPath)
	if err != nil {
		panic(fmt.Sprintf("Error reading file: %+v", err))
	}

	templateContent := string(tmplRaw)

	return templateContent, nil
}

func (ctrl *TargetProcessorController) renderEachFile(
	path, templateRaw string,
	api config.ApiDefinition,
	ms config.MicroserviceDefinition,
) error {
	targetLabel := ctrl.ParentTarget.Label
	templatesDir := ctrl.TemplatesDir
	pluginReg := ctrl.PluginRegistry

	// Create the directory structure
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create directory structure")
	}

	os.Chmod(path, 0666)
	file, err := os.Create(path)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()

	// Inject function implementation snippets
	templateRaw, err = injectFunctionImplementationSnippets(templateRaw, ms, targetLabel, templatesDir)
	if err != nil {
		return errors.WithStack(err)
	}

	templateRaw = preprocessTemplate(templateRaw, ms, ctrl.TagRegistry)

	tmpl, err := goTmpl.New(path).Funcs(templateFuncs()).Parse(templateRaw)
	if err != nil {
		slog.Info(templateRaw)
		return errors.WithStack(err)
	}

	var sb strings.Builder
	err = tmpl.Execute(&sb, struct {
		Api          config.ApiDefinition
		Microservice config.MicroserviceDefinition
	}{
		Api:          api,
		Microservice: ms,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	result := strings.TrimSpace(sb.String())

	content := []byte(result)
	for _, p := range pluginReg.GetPlugins(targetLabel) {
		var err error
		content, err = p.PreWriteFile(context.Background(), path, content)
		if err != nil {
			return errors.Wrap(err, "plugin execution failed")
		}
	}

	_, err = file.Write(content)
	if err != nil {
		return errors.WithStack(err)
	}

	fileMode := ctrl.ParentTarget.Options.FileMode
	if fileMode == 0 {
		os.Chmod(path, 0444)
	} else {
		os.Chmod(path, fileMode)
	}

	return nil
}

func injectFunctionImplementationSnippets(
	templateRaw string,
	ms config.MicroserviceDefinition,
	targetLabel string,
	templatesDir string,
) (string, error) {
	for _, funcImpl := range ms.FunctionImplementations {
		snippetPaths, ok := funcImpl.TargetSnippets[targetLabel]
		if !ok {
			continue
		}

		// Handle hook property
		re := regexp.MustCompile(`{{/\* FUNCTION_IMPLEMENTATIONS\s+hook="(\w+)"\s+\*/}}`)
		templateRaw = re.ReplaceAllStringFunc(templateRaw, func(match string) string {
			hookName := re.FindStringSubmatch(match)[1]
			if snippetPaths.HooksDirectory != "" {
				hookPath := templatesDir + snippetPaths.HooksDirectory + "/" + hookName + ".template"
				hookContent, err := os.ReadFile(hookPath)
				if err == nil {
					return string(hookContent) + match
				}
			}
			return match
		})

		var snippetContent []byte
		if snippetPaths.ContentPath != "" {
			fullSnippetPath := templatesDir + snippetPaths.ContentPath
			var err error
			snippetContent, err = os.ReadFile(fullSnippetPath)
			if err != nil {
				return "", errors.Wrap(err, fmt.Sprintf("Error reading snippet file: %s", fullSnippetPath))
			}
		}

		var importsContent []byte
		if snippetPaths.ImportsPath != "" {
			fullImportsPath := templatesDir + snippetPaths.ImportsPath
			var err error
			importsContent, err = os.ReadFile(fullImportsPath)
			if err != nil {
				// If imports file doesn't exist, continue without it
				importsContent = []byte("")
			}
		}

		placeholderTag := "{{/* FUNCTION_IMPLEMENTATIONS */}}"
		templateRaw = strings.Replace(templateRaw, placeholderTag, string(snippetContent)+placeholderTag, 1)

		importsPlaceholderTag := "{{/* FUNCTION_IMPORTS */}}"
		templateRaw = strings.Replace(templateRaw, importsPlaceholderTag, string(importsContent)+importsPlaceholderTag, 1)
	}

	return templateRaw, nil
}

func (ctrl *TargetProcessorController) renderSingleFile(path, templateStr string, api config.ApiDefinition) error {
	targetLabel := ctrl.ParentTarget.Label
	pluginReg := ctrl.PluginRegistry

	// Create the directory structure
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create directory structure")
	}

	os.Chmod(path, 0666)
	file, err := os.Create(path)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()

	templateStr = preprocessTemplate(templateStr, config.MicroserviceDefinition{}, ctrl.TagRegistry)

	tmpl, err := goTmpl.New(path).Funcs(templateFuncs()).Parse(templateStr)
	if err != nil {
		return errors.WithStack(err)
	}

	var sb strings.Builder
	err = tmpl.Execute(&sb, api)
	if err != nil {
		return errors.WithStack(err)
	}

	result := strings.TrimSpace(sb.String())

	content := []byte(result)
	for _, p := range pluginReg.GetPlugins(targetLabel) {
		var err error
		content, err = p.PreWriteFile(context.Background(), path, content)
		if err != nil {
			return errors.Wrap(err, "plugin execution failed")
		}
	}

	_, err = file.Write(content)
	if err != nil {
		return errors.WithStack(err)
	}

	fileMode := ctrl.ParentTarget.Options.FileMode
	if fileMode == 0 {
		os.Chmod(path, 0444)
	} else {
		os.Chmod(path, fileMode)
	}

	return nil
}

func templateFuncs() goTmpl.FuncMap {
	return goTmpl.FuncMap{
		"protoType":                     mapToProtoType,
		"mapGoType":                     mapGoType,
		"mapGoTypeWithCustomTypePrefix": mapGoTypeWithCustomTypePrefix,
		"toGoModelFieldCase":            toGoModelFieldCase,
		"add":                           func(a, b int) int { return a + b },
		"camelCase":                     strcase.ToCamel,
		"camelCaseCapitalizeID":         camelCaseCapitalizeID,
		"lowerCamelCaseCapitalizeID":    lowerCamelCaseCapitalizeID,
		"lowerCamelCaseNoExceptions":    lowerCamelCaseNoExceptions,
		"camelCaseNoExceptions":         camelCaseNoExceptions,
		"snakecase":                     strcase.ToSnake,
		"lowerCamelCase":                strcase.ToLowerCamel,
		"mapGraphQLType":                mapGraphQLType,
		"mapGraphQLInputType":           mapGraphQLInputType,
		"getGraphqlTypeForField":        getGraphqlTypeForField,
		"getGraphqlNameForField":        getGraphqlNameForField,
		"mapTypescriptType":             mapTypescriptType,
		"fieldHasTag":                   fieldHasTag,
		"isPrimitiveFieldType":          config.IsPrimitiveFieldType,
		"getModelDirective":             getModelDirective,
		"getModelDirectiveList":         getModelDirectiveList,
		"getFieldDirective":             getFieldDirective,
	}
}

func mapToProtoType(codemaType string) string {
	switch codemaType {
	case "ID", "String":
		return "string"
	case "Int":
		return "int64"
	case "Float":
		return "double"
	case "Boolean":
		return "bool"
	case "DateTime":
		return "google.protobuf.Timestamp"
	default:
		if strings.HasPrefix(codemaType, "[") && strings.HasSuffix(codemaType, "]") {
			return "repeated " + mapToProtoType(codemaType[1:len(codemaType)-1])
		}
		return codemaType // For custom types, use as-is
	}
}

func mapGoType(codemaType string) string {
	switch codemaType {
	case "ID", "String":
		return "string"
	case "Int":
		return "int64"
	case "Float":
		return "float64"
	case "Boolean":
		return "bool"
	case "DateTime":
		return "time.Time"
	default:
		if strings.HasPrefix(codemaType, "[") && strings.HasSuffix(codemaType, "]") {
			return "[]" + mapGoType(codemaType[1:len(codemaType)-1])
		}
		return codemaType // For custom types, use as-is
	}
}

func mapGoTypeWithCustomTypePrefix(codemaType string, customTypePrefix string) string {
	switch codemaType {
	case "ID", "String":
		return "string"
	case "Int":
		return "int64"
	case "Float":
		return "float64"
	case "Boolean":
		return "bool"
	case "DateTime":
		return "time.Time"
	default:
		if strings.HasPrefix(codemaType, "[") && strings.HasSuffix(codemaType, "]") {
			return "[]" + mapGoType(codemaType[1:len(codemaType)-1])
		}
		return customTypePrefix + codemaType
	}
}

func preprocessTemplate(
	templateStr string,
	ms config.MicroserviceDefinition,
	tagReg tag.TagRegistry,
) string {
	// Replace @PM# or @PrimaryModel# or # followed by a tag name
	re := regexp.MustCompile(`\{\{\W?([^}]*)?#(\w+)[^}]*}}`)
	templateStr = re.ReplaceAllStringFunc(templateStr, func(match string) string {
		groups := re.FindStringSubmatch(match)
		before := groups[1]
		tagName := groups[2]

		for _, field := range ms.PrimaryModel.Fields {
			for _, fieldTag := range field.Tags {
				if fieldTag.Name == tagName {
					if before == "" {
						return field.Name
					} else {
						return strings.Replace(match, "#"+tagName, field.Name, -1)
					}
				}
			}
		}

		slog.Warn("got no field for tag", slog.String("tag", tagName), slog.String("model", ms.PrimaryModel.Name))

		return match // If no matching tag is found, return the original match
	})

	templateStr = resolveTagReferences(templateStr, tagReg.GetTagByName)

	// Replace @PM or @PrimaryModel with {{ .Microservice.PrimaryModel }}
	re = regexp.MustCompile(`@PM|@PrimaryModel`)
	templateStr = re.ReplaceAllString(templateStr, ".Microservice.PrimaryModel")

	return templateStr
}

func resolveTagReferences(template string, resolveTag func(string) config.TagDefinition) string {
	re := regexp.MustCompile(`@Tags\.[^\W.]+`)
	matches := re.FindAllString(template, -1)

	for _, match := range matches {
		tagName := strings.TrimPrefix(match, "@Tags.")
		tag := resolveTag(tagName)
		if tag.Name == "" {
			msg := fmt.Sprintf("not found: %s", tagName)
			slog.Warn(msg)
			template = strings.ReplaceAll(template, match, "\"TAG_NOT_FOUND\"")
			return template
		}

		template = strings.ReplaceAll(template, match, "\""+tag.Name+"\"")
	}

	return template
}

func mapGraphQLType(t string) string {
	switch t {
	case "ID", "String":
		return "String"
	case "Int":
		return "Int"
	case "Float":
		return "Float"
	case "Boolean":
		return "Boolean"
	case "DateTime":
		return "Int"
	default:
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			return "[" + mapGraphQLType(t[1:len(t)-1]) + "]"
		}
		return t
	}
}

func mapGraphQLInputType(t string) string {
	switch t {
	case "ID", "String":
		return "String"
	case "Int":
		return "Int"
	case "Float":
		return "Float"
	case "Boolean":
		return "Boolean"
	case "DateTime":
		return "Int"
	default:
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			return "[" + mapGraphQLInputType(t[1:len(t)-1]) + "]"
		}
		return t + "Input"
	}
}

func getGraphqlTypeForField(f config.FieldDefinition) string {
	mask := f.GetDirectiveStringValue(directive.WellKnownDirectiveGraphQLTypeNameMask)
	if mask != "" {
		return mask
	}

	return mapGraphQLType(f.Type)
}

func getGraphqlNameForField(f config.FieldDefinition) string {
	mask := f.GetDirectiveStringValue(directive.WellKnownDirectiveGraphQLFieldNameMask)
	if mask != "" {
		return mask
	}

	return strcase.ToLowerCamel(f.Name)
}

func mapTypescriptType(codemaType string) string {
	switch codemaType {
	case "ID", "String":
		return "string"
	case "Int":
		return "number"
	case "Float":
		return "number"
	case "Boolean":
		return "boolean"
	case "DateTime":
		return "number"
	default:
		return codemaType // For custom types and enums, use as-is
	}
}

func toGoModelFieldCase(fieldName string) string {
	// Convert field name to TitleCase
	titleCaseField := strcase.ToCamel(fieldName)

	// Check if the field name ends with "Id" and change it to "ID"
	if strings.HasSuffix(titleCaseField, "Id") {
		titleCaseField = strings.TrimSuffix(titleCaseField, "Id") + "ID"
	}

	return titleCaseField
}

func camelCaseCapitalizeID(fieldName string) string {
	camelCaseField := strcase.ToSnake(fieldName)

	if strings.HasSuffix(camelCaseField, "_id") {
		// Replace the last occurrence of "id" or "Id" with "ID"
		camelCaseField = strcase.ToCamel(strings.TrimSuffix(camelCaseField, "_id")) + "ID"
	}
	return camelCaseField
}

func lowerCamelCaseCapitalizeID(fieldName string) string {
	camelCaseField := strcase.ToSnake(fieldName)

	if strings.HasSuffix(camelCaseField, "_id") {
		// Replace the last occurrence of "id" or "Id" with "ID"
		camelCaseField = strcase.ToLowerCamel(strings.TrimSuffix(camelCaseField, "_id")) + "ID"
	}
	return camelCaseField
}

func lowerCamelCaseNoExceptions(fieldName string) string {
	camelCaseField := strcase.ToSnake(fieldName)

	if strings.HasSuffix(camelCaseField, "_id") {
		camelCaseField = strcase.ToLowerCamel(strings.TrimSuffix(camelCaseField, "_id")) + "Id"
	}
	return camelCaseField
}

func camelCaseNoExceptions(fieldName string) string {
	camelCaseField := strcase.ToSnake(fieldName)

	if strings.HasSuffix(camelCaseField, "_id") {
		camelCaseField = strcase.ToCamel(strings.TrimSuffix(camelCaseField, "_id")) + "Id"
	}
	return camelCaseField
}

func fieldHasTag(field config.FieldDefinition, tagName string) bool {
	var hasTag bool
	for _, t := range field.Tags {
		if t.Name == tagName {
			hasTag = true
			break
		}
	}

	return hasTag
}

func getModelDirective(f config.ModelDefinition, s string, defaultVal string) string {
	val := f.GetDirectiveStringValue(s)
	if val != "" {
		return val
	}

	return defaultVal
}

func getModelDirectiveList(f config.ModelDefinition, s string) []interface{} {
	arrVal := f.GetDirectiveListValue(s)
	if arrVal != nil && len(arrVal) > 0 {
		return arrVal
	}

	return make([]interface{}, 0)
}

func getFieldDirective(f config.FieldDefinition, s string, defaultVal string) string {
	val := f.GetDirectiveStringValue(s)
	if val != "" {
		return val
	}

	return defaultVal
}
