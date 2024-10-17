package target

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/innovation-upstream/codema/internal/config"
	"github.com/innovation-upstream/codema/internal/fs"
	"github.com/innovation-upstream/codema/internal/model"
	"github.com/innovation-upstream/codema/internal/plugin"
	"github.com/innovation-upstream/codema/internal/tag"
	targetrenderer "github.com/innovation-upstream/codema/internal/target-renderer"
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

	targetTmplRaw, tmplPath, err := tp.getRawTemplate(ta)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	var renderer targetrenderer.TargetRenderer
	switch true {
	case strings.HasSuffix(tmplPath, ".plush"):
		renderer = &targetrenderer.PlushTemplateTargetRenderer{}
		break
	case strings.HasSuffix(tmplPath, ".template") || strings.HasSuffix(tmplPath, ".gotemplate"):
		renderer = &targetrenderer.GoTemplateTargetRenderer{}
		break
	default:
		renderer = &targetrenderer.GoTemplateTargetRenderer{}
		break
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

			err = ctrl.renderEachFile(msOutFilePath, targetTmplRaw, a, m, renderer)
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

		err = ctrl.renderSingleFile(apiOutFilePath, targetTmplRaw, a, renderer)
		if err != nil {
			return 0, errors.WithStack(err)
		}

		numFiles++
	}

	return numFiles, nil
}

func getTemplateVersionPath(defaultVersion, version string) string {
	if version == "" {
		return defaultVersion
	} else {
		return version
	}
}

func (tp *TargetProcessor) getRawTemplate(
	ta config.TargetApi,
) (string, string, error) {
	templateVersionPath := getTemplateVersionPath(tp.ParentTarget.DefaultVersionPath, ta.VersionPath)
	var tmplPath string
	if tp.ParentTarget.TemplateDir == "" {
		tmplPath = fs.GetLegacyTemplatePath(tp.TemplatesDir, tp.ParentTarget.TemplatePath)
	} else if templateVersionPath != "" {
		tmplPath = fs.GetTemplatePath(tp.TemplatesDir, tp.ParentTarget.TemplateDir, templateVersionPath)
	} else {
		desc := "You specified templateDir without specifing a template version!  You must specify either Target.DefaultVersion or a TargetApi.Version"
		msg := fmt.Sprintf(
			"Failed to render target: %s for api: %s. Message: %s",
			tp.ParentTarget.Label,
			ta.Label,
			desc,
		)
		err := errors.New(msg)
		return "", "", err
	}

	tmplRaw, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", "", errors.New(fmt.Sprintf("Error reading file: %+v", err))
	}

	templateContent := string(tmplRaw)

	return templateContent, tmplPath, nil
}

func (ctrl *TargetProcessorController) renderEachFile(
	path, templateRaw string,
	api config.ApiDefinition,
	ms config.MicroserviceDefinition,
	renderer targetrenderer.TargetRenderer,
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

	data := struct {
		Api          config.ApiDefinition
		Microservice config.MicroserviceDefinition
	}{
		Api:          api,
		Microservice: ms,
	}

	result, err := renderer.Render(templateRaw, data)
	if err != nil {
		return errors.WithStack(err)
	}

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

func replacePlaceholder(templateRaw, placeholderTag, content string, repeat bool) string {
	// Define a regex pattern that makes {{ and }} optional around the placeholder
	placeholderPattern := regexp.MustCompile(`{{\s*` + regexp.QuoteMeta(placeholderTag) + `\s*}}|` + regexp.QuoteMeta(placeholderTag))

	return placeholderPattern.ReplaceAllStringFunc(templateRaw, func(match string) string {
		if repeat {
			return content + match
		}
		return content
	})
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
		re := regexp.MustCompile(`({{)?/\* FUNCTION_IMPLEMENTATIONS\s+hook="(\w+)"\s+\*/(}})?`)
		templateRaw = re.ReplaceAllStringFunc(templateRaw, func(match string) string {
			hookName := re.FindStringSubmatch(match)[1]
			if snippetPaths.HooksDirectory != "" {
				hookPath := templatesDir + snippetPaths.HooksDirectory + "/" + hookName
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

		templateRaw = replacePlaceholder(templateRaw, "/* FUNCTION_IMPLEMENTATIONS */", string(snippetContent), true)
		templateRaw = replacePlaceholder(templateRaw, "/* FUNCTION_IMPORTS */", string(importsContent), true)
	}

	templateRaw = replacePlaceholder(templateRaw, "/* FUNCTION_IMPLEMENTATIONS */", "", false)
	templateRaw = replacePlaceholder(templateRaw, "/* FUNCTION_IMPORTS */", "", false)

	return templateRaw, nil
}

func (ctrl *TargetProcessorController) renderSingleFile(
	path,
	templateStr string,
	api config.ApiDefinition,
	renderer targetrenderer.TargetRenderer,
) error {
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

	result, err := renderer.Render(templateStr, api)
	if err != nil {
		return errors.WithStack(err)
	}

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
