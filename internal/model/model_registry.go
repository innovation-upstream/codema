package model

import (
	"fmt"

	"github.com/innovation-upstream/codema/internal/config"
	"github.com/pkg/errors"
)

type ModelRegistry interface {
	GetModelByName(name string) config.ModelDefinition
	RegisterModel(model config.ModelDefinition) error
}

type modelRegistry struct {
	models map[string]config.ModelDefinition
}

func NewModelRegistery(initialModels map[string]config.ModelDefinition) ModelRegistry {
	defaultModels := initialModels
	if defaultModels == nil {
		defaultModels = make(map[string]config.ModelDefinition)
	}

	return &modelRegistry{
		models: defaultModels,
	}
}

func (r *modelRegistry) GetModelByName(name string) config.ModelDefinition {
	m, ok := r.models[name]
	if ok {
		return m
	}

	return config.ModelDefinition{}
}

func (r *modelRegistry) RegisterModel(model config.ModelDefinition) error {
	if _, ok := r.models[model.Name]; ok {
		return errors.New(fmt.Sprintf("Model with name %s already registered", model.Name))
	}

	r.models[model.Name] = model
	fmt.Printf("Registered Model: %s\n", model.Name)

	return nil
}
