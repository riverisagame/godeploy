package domain

import "errors"

type Server struct {
	ID   uint
	Name string
	IP   string
	Port int
}

func NewServer(name, ip string, port int) (*Server, error) {
	if name == "" {
		return nil, errors.New("server name cannot be empty")
	}
	if ip == "" {
		return nil, errors.New("server IP cannot be empty")
	}
	if port <= 0 {
		port = 22
	}

	return &Server{
		Name: name,
		IP:   ip,
		Port: port,
	}, nil
}
