package services

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
	"github.com/RenanAlvesBCC/todolist-api/internal/utils"
)

// AuthService contém a regra de negócio de autenticação: nem sabe que existe HTTP,
// nem sabe como o usuário é salvo — só usa o repository pra isso.
type AuthService struct {
	userRepo *repository.UserRepository
}

// NewAuthService cria o service recebendo o repository que ele vai usar.
func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

// Register gera o hash da senha e cria o usuário através do repository.
func (s *AuthService) Register(username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &models.User{
		Username: username,
		Password: string(hash),
	}

	return s.userRepo.Create(user)
}

// Login busca o usuário, compara a senha e, se tudo bater, gera o token.
func (s *AuthService) Login(username, password string) (string, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return "", errors.New("usuário ou senha incorretos")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("usuário ou senha incorretos")
	}

	return utils.GenerateToken(user.ID, user.Username)
}
