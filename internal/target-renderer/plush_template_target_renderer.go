package targetrenderer

import (
	"fmt"

	"github.com/gobuffalo/plush"
	"github.com/pkg/errors"
)

type PlushTemplateTargetRenderer struct{}

func (r *PlushTemplateTargetRenderer) Render(templateContent string, data interface{}) (string, error) {
	ctx := plush.NewContext()
	ctx.Set("data", data)

	for name, fn := range templateFuncs() {
		ctx.Set(name, fn)
	}

	fmt.Printf("templateContent: %+v\n", templateContent)
	result, err := plush.Render(templateContent, ctx)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return result, nil
}

func (r *PlushTemplateTargetRenderer) GetType() TargetRendererType {
	return TargetRendererType_Plush
}
