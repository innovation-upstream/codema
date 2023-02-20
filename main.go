package main

import (
	"fmt"
	"go/format"
	"os"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

func main() {
	config, err := getConfig()
	if err != nil {
		panic(err)
	}

	modulePath := expandModulePath(config.ModuleDir)
	templatePath := expandTemplatePath(config.TemplateDir)

	apis := make(map[string]ApiDefinition)

	for _, a := range config.Apis {
		apis[a.Label] = a
	}

	for _, t := range config.Targets {
		for _, ta := range t.Apis {
			a := apis[ta.Label]

			pathTmpl, err := template.New("outpath").Parse(ta.OutPath)
			if err != nil {
				panic(fmt.Sprintf("Error executing template: %+v", err))
			}

			var pathSb strings.Builder
			if ta.Each {
			msRender:
				for _, m := range a.Microservices {
					for _, sl := range ta.SkipLabels {
						if sl == m.Label {
							continue msRender
						}
					}

					pathSb.Reset()
					err = pathTmpl.Execute(&pathSb, struct {
						Label        string
						Microservice MicroserviceDefinition
						Api          ApiDefinition
					}{
						Label:        a.Label,
						Microservice: m,
						Api:          a,
					})
					if err != nil {
						panic(fmt.Sprintf("Error executing template: %+v", err))
					}

					pathExpanded := pathSb.String()
					path := modulePath + pathExpanded
					tmplPath := templatePath + t.TemplatePath

					tmplRaw, err := os.ReadFile(tmplPath)
					if err != nil {
						fmt.Println("Error reading file:", err)
						return
					}

					err = renderEachFile(path, string(tmplRaw), a, m)
					if err != nil {
						panic(err)
					}
				}
			} else {
				err = pathTmpl.Execute(&pathSb, struct {
					Label string
					Api   ApiDefinition
				}{
					Label: a.Label,
					Api:   a,
				})
				if err != nil {
					panic(fmt.Sprintf("Error executing template: %+v", err))
				}

				pathExpanded := pathSb.String()
				path := modulePath + pathExpanded
				tmplPath := templatePath + t.TemplatePath

				tmplRaw, err := os.ReadFile(tmplPath)
				if err != nil {
					fmt.Println("Error reading file:", err)
					return
				}

				err = renderSingleFile(path, string(tmplRaw), a)
				if err != nil {
					fmt.Println(fmt.Errorf("%+v\n", err))
					return
				}
			}
		}
	}
}

func renderSingleFile(path, templateStr string, api ApiDefinition) error {
	os.Chmod(path, 0666)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := template.New(path).Parse(templateStr)
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

func renderEachFile(
	path, templateRaw string,
	api ApiDefinition,
	ms MicroserviceDefinition,
) error {
	os.Chmod(path, 0666)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := template.New(path).Parse(templateRaw)
	if err != nil {
		return err
	}

	var sb strings.Builder
	err = tmpl.Execute(&sb, struct {
		Api          ApiDefinition
		Microservice MicroserviceDefinition
	}{
		Api:          api,
		Microservice: ms,
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
