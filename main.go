package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

type (
	TargetFlags []string
)

func (t TargetFlags) Includes(s string) bool {
	if t == nil || len(t) == 0 {
		return false
	}

	head := t[0]
	tail := t[1:]

	if string(head) == s {
		return true
	}

	return tail.Includes(s)
}

func main() {
	var targetsRaw string
	flag.StringVar(&targetsRaw, "t", "*", "Targets to render")
	flag.Parse()

	isAllTargets := targetsRaw == "*"
	targetsToRender := TargetFlags(strings.Split(targetsRaw, ","))
	renderedTargets := TargetFlags{}
	logRenderTargets := strings.Join([]string(targetsToRender), ", ")
	if isAllTargets {
		logRenderTargets = "ALL"
	}

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

	fmt.Printf("Will render target(s): %s\n", logRenderTargets)
	for _, t := range config.Targets {
		if !isAllTargets {
			enabledByFlag := targetsToRender.Includes(t.Label)
			if !enabledByFlag {
				continue
			}

			renderedTargets = append(renderedTargets, t.Label)
		}

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

		fmt.Printf("Rendered target: %s\n", t.Label)
	}

	if len(renderedTargets) != len(targetsToRender) {
		for _, tr := range targetsToRender {
			if !renderedTargets.Includes(tr) {
				fmt.Printf("WARN Skipped target: %s because it was not defined\n", tr)
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
