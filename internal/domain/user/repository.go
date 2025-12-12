package user

// Repository defines the contract for user storage operations
type Repository interface {
	Create(user *User) error
	GetByID(id string) (*User, error)
	GetByEmail(email string) (*User, error)
	GetByUsername(username string) (*User, error)
	GetByGoogleID(googleID string) (*User, error)
	Update(user *User) error
	Delete(id string) error
	List() ([]User, error)
	Count() (int, error)
}
