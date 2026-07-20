package domain

type User struct {
	ID           uint
	Username     string
	PasswordHash string
}

type UserRepository interface {
	Save(u *User) error
	FindByUsername(username string) (*User, error)
}
