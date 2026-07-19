package ssh

import (
	"testing"
)

func TestLocalRunner_Run(t *testing.T) {
	r := NewLocalRunner()
	out, err := r.Run("echo hello")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if out != "hello\n" && out != "hello\r\n" {
		t.Errorf("Expected 'hello\\n', got '%q'", out)
	}
}
