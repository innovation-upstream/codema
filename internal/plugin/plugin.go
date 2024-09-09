package plugin

import (
	"context"
)

type Plugin interface {
	Name() string
	PreWriteFile(ctx context.Context, filename string, content []byte) ([]byte, error)
	PreExecTemplate(ctx context.Context, templateContent []byte) ([]byte, error)
}

type PluginRegistry struct {
	plugins map[string][]Plugin
}

func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string][]Plugin),
	}
}

func (r *PluginRegistry) Register(targetLabel string, plugin Plugin) {
	if _, ok := r.plugins[targetLabel]; !ok {
		r.plugins[targetLabel] = []Plugin{}
	}
	r.plugins[targetLabel] = append(r.plugins[targetLabel], plugin)
}

func (r *PluginRegistry) GetPlugins(targetLabel string) []Plugin {
	return r.plugins[targetLabel]
}

func (r *PluginRegistry) GetRegister() map[string][]Plugin {
	return r.plugins
}
