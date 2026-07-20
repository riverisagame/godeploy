package domain

import "errors"

type Server struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	User    string `json:"user"`     // SSH 用户名，默认 root
	KeyPath string `json:"key_path"` // SSH 私钥路径，空则使用默认 ~/.ssh/id_rsa
}

func NewServer(name, ip string, port int, user, keyPath string) (*Server, error) {
	if name == "" {
		return nil, errors.New("server name cannot be empty")
	}
	if ip == "" {
		return nil, errors.New("server IP cannot be empty")
	}
	if port <= 0 {
		port = 22
	}
	if user == "" {
		user = "root"
	}

	return &Server{
		Name:    name,
		IP:      ip,
		Port:    port,
		User:    user,
		KeyPath: keyPath,
	}, nil
}
