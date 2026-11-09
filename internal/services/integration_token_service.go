package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

const (
	IntegrationTokenPrefix  = "rmx_"
	integrationTokenBytes   = 32
	maxIntegrationTokenName = 100
)

var (
	ErrTokenOwnerOnly          = errors.New("apenas o dono pode gerenciar tokens de integração")
	ErrTokenNameRequired       = errors.New("nome é obrigatório")
	ErrTokenNameTooLong        = errors.New("nome deve ter no máximo 100 caracteres")
	ErrTokenNotFound           = errors.New("token não encontrado")
	ErrInvalidIntegrationToken = errors.New("token de integração inválido")
)

type IntegrationTokenStore interface {
	CreateToken(t *models.IntegrationToken) error
	ListTokens(workspaceID uint) ([]models.IntegrationToken, error)
	FindToken(workspaceID, id uint) (*models.IntegrationToken, error)
	RevokeToken(id uint, at time.Time) error
	FindActiveTokenByHash(hash string) (*models.IntegrationToken, error)
	TouchTokenUsed(id uint, at time.Time) error
}

type IntegrationTokenService struct {
	store   IntegrationTokenStore
	wsStore WorkspaceRoleStore
	now     func() time.Time
	random  io.Reader
}

func NewIntegrationTokenService(store IntegrationTokenStore, wsStore WorkspaceRoleStore) *IntegrationTokenService {
	return &IntegrationTokenService{store: store, wsStore: wsStore, now: time.Now, random: rand.Reader}
}

// CreatedIntegrationToken é a única resposta que traz o token em texto.
type CreatedIntegrationToken struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
}

func hashIntegrationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *IntegrationTokenService) ownerWorkspace(userID uint) (uint, error) {
	wsID, role, err := memberWorkspace(s.wsStore, userID)
	if err != nil {
		return 0, err
	}
	if role != models.RoleOwner {
		return 0, ErrTokenOwnerOnly
	}
	return wsID, nil
}

func (s *IntegrationTokenService) ListTokens(userID uint) ([]models.IntegrationToken, error) {
	wsID, err := s.ownerWorkspace(userID)
	if err != nil {
		return nil, err
	}
	return s.store.ListTokens(wsID)
}

func (s *IntegrationTokenService) CreateToken(userID uint, name string) (*CreatedIntegrationToken, error) {
	wsID, err := s.ownerWorkspace(userID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrTokenNameRequired
	}
	if utf8.RuneCountInString(name) > maxIntegrationTokenName {
		return nil, ErrTokenNameTooLong
	}

	raw := make([]byte, integrationTokenBytes)
	if _, err := io.ReadFull(s.random, raw); err != nil {
		return nil, err
	}
	token := IntegrationTokenPrefix + base64.RawURLEncoding.EncodeToString(raw)

	t := &models.IntegrationToken{WorkspaceID: wsID, Name: name, TokenHash: hashIntegrationToken(token), CreatedBy: userID}
	if err := s.store.CreateToken(t); err != nil {
		return nil, err
	}
	return &CreatedIntegrationToken{ID: t.ID, Name: t.Name, Token: token, CreatedAt: t.CreatedAt}, nil
}

// RevokeToken é idempotente: revogar de novo não muda a data original.
func (s *IntegrationTokenService) RevokeToken(userID, id uint) error {
	wsID, err := s.ownerWorkspace(userID)
	if err != nil {
		return err
	}
	if _, err := s.store.FindToken(wsID, id); err != nil {
		return ErrTokenNotFound
	}
	return s.store.RevokeToken(id, s.now().UTC())
}

// Authenticate confere o token do robô (hash + não revogado) e registra o uso.
func (s *IntegrationTokenService) Authenticate(token string) (*models.IntegrationToken, error) {
	if !strings.HasPrefix(token, IntegrationTokenPrefix) {
		return nil, ErrInvalidIntegrationToken
	}
	t, err := s.store.FindActiveTokenByHash(hashIntegrationToken(token))
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrInvalidIntegrationToken
	}
	now := s.now().UTC()
	if err := s.store.TouchTokenUsed(t.ID, now); err != nil {
		return nil, err
	}
	t.LastUsedAt = &now
	return t, nil
}
