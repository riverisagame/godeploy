package domain

import (
	"gopkg.in/yaml.v3"
)

type TaskConfig struct {
	Stage  string   `yaml:"stage"`
	Type   string   `yaml:"type"`   // "script" or "sync"
	RunOn  string   `yaml:"run_on"` // "local" or "remote"
	Script []string `yaml:"script"`
}

type Pipeline struct {
	Version string                 `yaml:"version"`
	Stages  []string               `yaml:"stages"`
	Tasks   map[string]*TaskConfig `yaml:"tasks"`
}

func ParsePipeline(data []byte) (*Pipeline, error) {
	var p Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
