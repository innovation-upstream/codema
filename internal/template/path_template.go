package template

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/innovation-upstream/codema/internal/config"
	"github.com/pkg/errors"
)

type (
	PathTemplateString struct {
		rawValue string
		template *template.Template
	}

	MicroservicePathTemplateInput struct {
		Label        string
		Microservice config.MicroserviceDefinition
		Api          config.ApiDefinition
	}

	ApiPathTemplateInput struct {
		Label string
		Api   config.ApiDefinition
	}
)

func NewPathTemplateString(outPath string) (*PathTemplateString, error) {
	pathTmpl, err := template.New("outpath").Parse(outPath)
	if err != nil {
		msg := fmt.Sprintf("Error executing template: %+v", err)
		return nil, errors.New(msg)
	}

	return &PathTemplateString{
		rawValue: outPath,
		template: pathTmpl,
	}, nil
}

func (ps PathTemplateString) ExecuteMicroservicePathTemplate(
	input MicroservicePathTemplateInput,
) (string, error) {
	var pathSb strings.Builder
	err := ps.template.Execute(&pathSb, input)
	if err != nil {
		msg := fmt.Sprintf("Error executing path template: %+v", err)
		return "", errors.New(msg)
	}

	return pathSb.String(), nil
}

func (ps PathTemplateString) ExecuteApiPathTemplate(
	input ApiPathTemplateInput,
) (string, error) {
	var pathSb strings.Builder
	err := ps.template.Execute(&pathSb, input)
	if err != nil {
		msg := fmt.Sprintf("Error executing path template: %+v", err)
		return "", errors.New(msg)
	}

	return pathSb.String(), nil
}
