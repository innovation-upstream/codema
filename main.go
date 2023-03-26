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

func (t TargetFlags) TrimSpace() TargetFlags {
	if t == nil || len(t) == 0 {
		return t
	}

	head := t[0]
	tail := t[1:]
	chunk := tail.TrimSpace()

	return append(chunk, strings.TrimSpace(head))
}

func main() {
	var targetsRaw string
	flag.StringVar(&targetsRaw, "t", "*", "Targets to render")
	flag.Parse()

	isAllTargets := targetsRaw == "*"
	targetsToRender := TargetFlags(strings.Split(targetsRaw, ","))
	targetsToRender = targetsToRender.TrimSpace()

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
	templateBasePath := expandTemplatePath(config.TemplateDir)

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
			if t.Each {
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
					templateVersion := getTemplateVersion(t.DefaultVerion, ta.Version)

					var tmplPath string
					if t.TemplateDir == "" {
						tmplPath = getLegacyTemplatePath(templateBasePath, t.TemplatePath)
					} else if templateVersion != "" {
						tmplPath = getTemplatePath(templateBasePath, t.TemplateDir, templateVersion)
					} else {
						desc := "You specified templateDir without specifing a template version!  You must specify either Target.DefaultVersion or a TargetApi.Version"
						msg := fmt.Sprintf(
							"Failed to render target: %s for api: %s. Message: %s",
							t.Label,
							ta.Label,
							desc,
						)
						err := errors.New(msg)
						panic(err)
					}

					isDir, err := isDir(path)
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

					args := ta.Args[m.Label]

					err = renderEachFile(path, string(tmplRaw), a, m, args)
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
				templateVersion := getTemplateVersion(t.DefaultVerion, ta.Version)

				var tmplPath string
				if t.TemplateDir == "" {
					tmplPath = getLegacyTemplatePath(templateBasePath, t.TemplatePath)
				} else if templateVersion != "" {
					tmplPath = getTemplatePath(templateBasePath, t.TemplateDir, templateVersion)
				} else {
					err := errors.New(
						"You specified templateDir without specifing a template version!  You must specify either Target.DefaultVersion or a TargetApi.Version",
					)
					panic(err)
				}

				isDir, err := isDir(path)
				if err != nil {
					panic(err)
				}

				if isDir {
					panic(fmt.Sprintf("ERROR: %s is a directory, aborting", path))
				}

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

	if !isAllTargets && len(renderedTargets) != len(targetsToRender) {
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
	args map[string]map[string]string,
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

func isDir(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		// This means the file doesn't exist
		return false, nil
	}

	isDir := fileInfo.IsDir()

	return isDir, nil
}

func getTemplateVersion(defaultVersion, version string) string {
	if version == "" {
		return defaultVersion
	} else {
		return version
	}
}

func getLegacyTemplatePath(basePath, templatePath string) string {
	tmplPath := basePath + templatePath
	return tmplPath
}

func getTemplatePath(basePath, templateDir, templateVersion string) string {
	tmplPath := basePath + templateDir + "/" + templateVersion + ".template"
	return tmplPath
}
