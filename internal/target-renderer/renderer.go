package targetrenderer

type TargetRendererType uint32

const (
	TargetRendererType_GoTemplate = 1
	TargetRendererType_Plush      = 2
)

type TargetRenderer interface {
	Render(templateContent string, data interface{}) (string, error)
	GetType() TargetRendererType
}
