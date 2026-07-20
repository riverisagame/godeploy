package domain

type ServerRepository interface {
	Save(server *Server) error
	FindAll() ([]*Server, error)
	FindByID(id uint) (*Server, error)
	Delete(id uint) error
}
