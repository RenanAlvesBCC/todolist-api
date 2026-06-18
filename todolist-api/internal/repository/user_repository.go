package repository

import (
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

// UserRepository encapsula o acesso à tabela de usuários.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository cria uma instância do repositório recebendo a conexão do banco.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create insere um novo usuário no banco.
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// FindByUsername busca um usuário pelo username; retorna erro se não existir.
func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
