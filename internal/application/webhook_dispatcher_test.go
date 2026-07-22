package application_test

import (
	"testing"
	"github.com/riverisagame/godeploy/internal/application"
)

func TestWebhookDispatcher_Dispatch(t *testing.T) {
	dispatcher := application.NewWebhookDispatcher()
	dispatcher.Start()

	// Should not crash, fail silently inside the worker
	dispatcher.Dispatch(application.WebhookEvent{URL: "http://127.0.0.1:0", Payload: []byte("{}")})
}
