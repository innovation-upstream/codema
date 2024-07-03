package target

import (
	"fmt"
	"go/format"
	"os"
	"strings"

	goTmpl "text/template"

	"github.com/innovation-upstream/codema/internal/config"
	"github.com/innovation-upstream/codema/internal/fs"
	"github.com/innovation-upstream/codema/internal/template"
	"github.com/pkg/errors"
)

type (
	TargetProcessorController struct {
		ApiRegistry  map[string]config.ApiDefinition
		ModulePath   string
		ParentTarget config.Target
		TemplatesDir string
	}

	TargetProcessor struct {
		Api          config.ApiDefinition
		ParentTarget config.Target
		TemplatesDir string
	}
)

func (ctrl *TargetProcessorController) ProcessTargetApi(ta config.TargetApi) error {
	a, ok := ctrl.ApiRegistry[ta.Label]
	if !ok {
		msg := fmt.Sprintf("Could not find api: %s", ta.Label)
		return errors.New(msg)
	}

	pathTmplStr, err := template.NewPathTemplateString(ta.OutPath)
	if err != nil {
		return errors.WithStack(err)
	}

	tp := TargetProcessor{
		Api:          a,
		ParentTarget: ctrl.ParentTarget,
		TemplatesDir: ctrl.TemplatesDir,
	}

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
				return errors.WithStack(err)
			}

			msOutFilePath := ctrl.ModulePath + msOutFileSubPath

			targetTmplRaw, err := tp.getRawTemplate(ta, ctrl.ParentTarget, msOutFilePath)
			if err != nil {
				return errors.WithStack(err)
			}

			args := ta.Args[m.Label]

			err = renderEachFile(msOutFilePath, targetTmplRaw, a, m, args)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	} else {
		apiOutFileSubPath, err := pathTmplStr.ExecuteApiPathTemplate(template.ApiPathTemplateInput{
			Api:   a,
			Label: a.Label,
		})
		if err != nil {
			return errors.WithStack(err)
		}

		apiOutFilePath := ctrl.ModulePath + apiOutFileSubPath

		targetTmplRaw, err := tp.getRawTemplate(ta, ctrl.ParentTarget, apiOutFilePath)
		if err != nil {
			return errors.WithStack(err)
		}

		err = renderSingleFile(apiOutFilePath, targetTmplRaw, a)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
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

	return string(tmplRaw), nil
}

func renderEachFile(
	path, templateRaw string,
	api config.ApiDefinition,
	ms config.MicroserviceDefinition,
	args map[string]map[string]string,
) error {
	os.Chmod(path, 0666)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := goTmpl.New(path).Parse(templateRaw)
	if err != nil {
		return err
	}

	var sb strings.Builder
	err = tmpl.Execute(&sb, struct {
		Api          config.ApiDefinition
		Microservice config.MicroserviceDefinition
		Args         map[string]map[string]string
	}{
		Api:          api,
		Microservice: ms,
		Args:         args,
	})
	if err != nil {
		return err
	}

	tmpResult := sb.String()
	fmtResult, err := format.Source([]byte(tmpResult))
	if err != nil {
		return err
	}

	result := strings.TrimSpace(string(fmtResult))

	_, err = file.WriteString(result)
	if err != nil {
		return err
	}

	os.Chmod(path, 0444)

	return nil
}

func renderSingleFile(path, templateStr string, api config.ApiDefinition) error {
	os.Chmod(path, 0666)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := goTmpl.New(path).Parse(templateStr)
	if err != nil {
		return errors.WithStack(err)
	}

	var sb strings.Builder
	err = tmpl.Execute(&sb, api)
	if err != nil {
		return errors.WithStack(err)
	}

	tmpResult := sb.String()
	fmtResult, err := format.Source([]byte(tmpResult))
	if err != nil {
		return errors.WithStack(err)
	}

	result := strings.TrimSpace(string(fmtResult))

	_, err = file.WriteString(result)
	if err != nil {
		return errors.WithStack(err)
	}

	os.Chmod(path, 0444)

	return nil
}
