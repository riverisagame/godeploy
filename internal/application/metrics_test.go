package application_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/riverisagame/godeploy/internal/application"
)

func TestMetricsInit(t *testing.T) {
	application.InitMetrics()
	assert.NotNil(t, application.HttpRequestsTotal, "HttpRequestsTotal should be initialized")
	assert.NotNil(t, application.HttpRequestDuration, "HttpRequestDuration should be initialized")
	assert.NotNil(t, application.DeployTotal, "DeployTotal should be initialized")
	assert.NotNil(t, application.DeployRunningCurrent, "DeployRunningCurrent should be initialized")
	assert.NotNil(t, application.DeployDuration, "DeployDuration should be initialized")
}
