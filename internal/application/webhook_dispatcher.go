package application

import (
	"bytes"
	"log"
	"net/http"
	"time"
)

type WebhookEvent struct {
	URL     string
	Payload []byte
}

type WebhookDispatcher struct {
	queue chan WebhookEvent
}

func NewWebhookDispatcher() *WebhookDispatcher {
	return &WebhookDispatcher{
		queue: make(chan WebhookEvent, 100),
	}
}

// @Ref: docs/sps/plans/20260722_v3.0_monitor_ir.md | @Date: 2026-07-22
func (w *WebhookDispatcher) Start() {
	go func() {
		for event := range w.queue {
			w.sendWithRetry(event)
		}
	}()
}

func (w *WebhookDispatcher) Dispatch(event WebhookEvent) {
	if event.URL == "" {
		return
	}
	select {
	case w.queue <- event:
	default:
		log.Printf("WebhookDispatcher queue is full, dropping event for URL: %s", event.URL)
	}
}

func (w *WebhookDispatcher) sendWithRetry(event WebhookEvent) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		req, _ := http.NewRequest("POST", event.URL, bytes.NewBuffer(event.Payload))
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return // Success
			}
		}
		
		time.Sleep(time.Duration(i*2+1) * time.Second) // Backoff
	}
	log.Printf("Webhook to %s failed after %d retries.", event.URL, maxRetries)
}
