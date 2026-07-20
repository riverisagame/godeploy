package domain

import "testing"

func TestNewServer(t *testing.T) {
	_, err := NewServer("", "127.0.0.1", 22, "", "")
	if err == nil || err.Error() != "server name cannot be empty" {
		t.Errorf("Expected name error, got %v", err)
	}

	_, err = NewServer("web1", "", 22, "", "")
	if err == nil || err.Error() != "server IP cannot be empty" {
		t.Errorf("Expected IP error, got %v", err)
	}

	s, err := NewServer("web1", "10.0.0.1", 22, "admin", "/home/admin/.ssh/id_rsa")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if s.Name != "web1" || s.IP != "10.0.0.1" || s.Port != 22 || s.User != "admin" || s.KeyPath != "/home/admin/.ssh/id_rsa" {
		t.Errorf("Server initialized incorrectly: %+v", s)
	}
}
