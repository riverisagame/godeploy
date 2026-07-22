package api_test

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"github.com/stretchr/testify/assert"
	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/interfaces/api"
)

func TestMetricsMiddleware(t *testing.T) {
	application.InitMetrics()

	handler := api.MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
