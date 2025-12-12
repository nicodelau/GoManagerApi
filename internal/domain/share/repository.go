package share

// Repository defines the contract for share storage operations
type Repository interface {
	Create(share *Share) error
	GetByID(id string) (*Share, error)
	GetByToken(token string) (*Share, error)
	GetByUser(userID string) ([]Share, error)
	GetByPath(path string) ([]Share, error)
	Update(share *Share) error
	Delete(id string) error
	IncrementDownloads(id string) error
}
