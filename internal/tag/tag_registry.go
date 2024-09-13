package tag

import (
	"errors"
	"fmt"

	"github.com/innovation-upstream/codema/internal/config"
)

type TagRegistry interface {
	GetTagByName(name string) config.TagDefinition
	RegisterTag(tag config.TagDefinition) error
}

type tagRegistry struct {
	tags map[string]config.TagDefinition
}

func NewTagRegistery() TagRegistry {
	return &tagRegistry{
		tags: make(map[string]config.TagDefinition),
	}
}

func (r *tagRegistry) GetTagByName(name string) config.TagDefinition {
	return r.tags[name]
}

func (r *tagRegistry) RegisterTag(tag config.TagDefinition) error {
	if _, ok := r.tags[tag.Name]; ok {
		return errors.New(fmt.Sprintf("Tag with name %s already registered", tag.Name))
	}

	r.tags[tag.Name] = tag

	return nil
}
