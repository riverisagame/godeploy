package domain_test

import (
	"testing"
	"github.com/riverisagame/godeploy/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestParsePipeline_Success(t *testing.T) {
	yamlData := []byte(`
version: "1.0"
stages:
  - pre_build
  - deploy
tasks:
  build_task:
    stage: pre_build
    type: script
    run_on: local
    script:
      - echo "building"
  sync_task:
    stage: deploy
    type: sync
`)

	p, err := domain.ParsePipeline(yamlData)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "1.0", p.Version)
	assert.Equal(t, 2, len(p.Stages))
	assert.Equal(t, "pre_build", p.Stages[0])
	
	task := p.Tasks["build_task"]
	assert.NotNil(t, task)
	assert.Equal(t, "pre_build", task.Stage)
	assert.Equal(t, "script", task.Type)
	assert.Equal(t, "local", task.RunOn)
	assert.Equal(t, "echo \"building\"", task.Script[0])
}

func TestParsePipeline_InvalidYaml(t *testing.T) {
	yamlData := []byte(`
version: "1.0"
stages: [invalid
`)
	_, err := domain.ParsePipeline(yamlData)
	assert.Error(t, err)
}
