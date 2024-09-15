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

func NewTagRegistery(initialTags map[string]config.TagDefinition) TagRegistry {
	defaultTags := initialTags
	if defaultTags == nil {
		defaultTags = make(map[string]config.TagDefinition)
	}

	return &tagRegistry{
		tags: defaultTags,
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
	fmt.Printf("Registered Tag: %s\n", tag.Name)

	return nil
}
