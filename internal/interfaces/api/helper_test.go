package api

import (
	"bytes"
	"errors"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type failWriter struct {
	*httptest.ResponseRecorder
}

func (fw *failWriter) Write(buf []byte) (int, error) {
	return 0, errors.New("simulated write error")
}

func TestRespondJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()
	RespondJSON(w, map[string]string{"msg": "ok"})

	if !strings.Contains(w.Body.String(), `"msg":"ok"`) {
		t.Errorf("expected body to contain json, got: %s", w.Body.String())
	}
}

func TestRespondJSON_Error(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	w := &failWriter{httptest.NewRecorder()}
	RespondJSON(w, map[string]string{"msg": "ok"})

	if !strings.Contains(buf.String(), "simulated write error") {
		t.Errorf("expected log to contain 'simulated write error', got: %s", buf.String())
	}
}
