package services

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

func newTokenFixture(t *testing.T) (*IntegrationTokenService, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc := NewIntegrationTokenService(repository.NewIntegrationRepository(db), fixtureWorkspaceStore())
	svc.now = func() time.Time { return time.Date(2026, 11, 9, 15, 0, 0, 0, time.UTC) }
	return svc, db
}

func TestIntegrationTokenService_CreateStoresOnlyHash(t *testing.T) {
	svc, db := newTokenFixture(t)
	created, err := svc.CreateToken(fxOwner, "  Robô OnMotor ")
	require.NoError(t, err)
	assert.Equal(t, "Robô OnMotor", created.Name)
	assert.True(t, strings.HasPrefix(created.Token, IntegrationTokenPrefix))
	assert.Len(t, created.Token, len(IntegrationTokenPrefix)+43, "32 bytes em base64url")

	var stored models.IntegrationToken
	require.NoError(t, db.First(&stored, created.ID).Error)
	assert.Equal(t, hashIntegrationToken(created.Token), stored.TokenHash)
	assert.Len(t, stored.TokenHash, 64, "SHA-256 em hex")
	assert.NotContains(t, stored.TokenHash, created.Token)
	assert.Equal(t, fxWorkspaceID, stored.WorkspaceID)
	assert.Equal(t, fxOwner, stored.CreatedBy)

	other, err := svc.CreateToken(fxOwner, "outro")
	require.NoError(t, err)
	assert.NotEqual(t, created.Token, other.Token)

	list, err := svc.ListTokens(fxOwner)
	require.NoError(t, err)
	require.Len(t, list, 2)
	raw, err := json.Marshal(list)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), created.Token, "listagem nunca traz o token")
	assert.NotContains(t, string(raw), stored.TokenHash, "nem o hash")
	assert.NotContains(t, string(raw), "token_hash")
}

func TestIntegrationTokenService_OwnerOnly(t *testing.T) {
	svc, _ := newTokenFixture(t)
	created, err := svc.CreateToken(fxOwner, "robô")
	require.NoError(t, err)

	for _, user := range []uint{fxManager, fxEditor} {
		_, err := svc.ListTokens(user)
		assert.ErrorIs(t, err, ErrTokenOwnerOnly)
		_, err = svc.CreateToken(user, "x")
		assert.ErrorIs(t, err, ErrTokenOwnerOnly)
		assert.ErrorIs(t, svc.RevokeToken(user, created.ID), ErrTokenOwnerOnly)
	}
	_, err = svc.ListTokens(fxOutsider)
	assert.ErrorIs(t, err, ErrNoWorkspace)
}

func TestIntegrationTokenService_Validation(t *testing.T) {
	svc, _ := newTokenFixture(t)
	_, err := svc.CreateToken(fxOwner, "  ")
	assert.ErrorIs(t, err, ErrTokenNameRequired)
	_, err = svc.CreateToken(fxOwner, strings.Repeat("a", 101))
	assert.ErrorIs(t, err, ErrTokenNameTooLong)

	svc.random = strings.NewReader("curto")
	_, err = svc.CreateToken(fxOwner, "robô")
	assert.Error(t, err, "falha no gerador aleatório")
}

func TestIntegrationTokenService_AuthenticateAndRevoke(t *testing.T) {
	svc, db := newTokenFixture(t)
	created, err := svc.CreateToken(fxOwner, "robô")
	require.NoError(t, err)

	tok, err := svc.Authenticate(created.Token)
	require.NoError(t, err)
	assert.Equal(t, fxWorkspaceID, tok.WorkspaceID)
	var stored models.IntegrationToken
	require.NoError(t, db.First(&stored, created.ID).Error)
	require.NotNil(t, stored.LastUsedAt, "uso registrado")

	for _, bad := range []string{"", "eyJhbGciOiJIUzI1NiJ9.x.y", IntegrationTokenPrefix + "inventado", strings.TrimPrefix(created.Token, IntegrationTokenPrefix)} {
		_, err := svc.Authenticate(bad)
		assert.ErrorIs(t, err, ErrInvalidIntegrationToken, bad)
	}

	require.NoError(t, svc.RevokeToken(fxOwner, created.ID))
	require.NoError(t, svc.RevokeToken(fxOwner, created.ID), "idempotente")
	_, err = svc.Authenticate(created.Token)
	assert.ErrorIs(t, err, ErrInvalidIntegrationToken, "revogado não autentica")

	foreign := &models.IntegrationToken{WorkspaceID: fxOtherWorkspc, Name: "x", TokenHash: "zz", CreatedBy: 77}
	require.NoError(t, db.Create(foreign).Error)
	assert.ErrorIs(t, svc.RevokeToken(fxOwner, foreign.ID), ErrTokenNotFound, "outra oficina → 404")
	assert.ErrorIs(t, svc.RevokeToken(fxOwner, 999), ErrTokenNotFound)
}

type tokenErrStore struct {
	IntegrationTokenStore
	findErr, touchErr, createErr error
}

func (s *tokenErrStore) FindActiveTokenByHash(hash string) (*models.IntegrationToken, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	return &models.IntegrationToken{ID: 1, WorkspaceID: fxWorkspaceID}, nil
}
func (s *tokenErrStore) TouchTokenUsed(id uint, at time.Time) error { return s.touchErr }
func (s *tokenErrStore) CreateToken(t *models.IntegrationToken) error {
	return s.createErr
}

func TestIntegrationTokenService_StoreErrors(t *testing.T) {
	boom := errors.New("banco fora")
	svc := NewIntegrationTokenService(&tokenErrStore{findErr: boom}, fixtureWorkspaceStore())
	_, err := svc.Authenticate(IntegrationTokenPrefix + "x")
	assert.ErrorIs(t, err, boom)

	svc = NewIntegrationTokenService(&tokenErrStore{touchErr: boom}, fixtureWorkspaceStore())
	_, err = svc.Authenticate(IntegrationTokenPrefix + "x")
	assert.ErrorIs(t, err, boom)

	svc = NewIntegrationTokenService(&tokenErrStore{createErr: boom}, fixtureWorkspaceStore())
	_, err = svc.CreateToken(fxOwner, "robô")
	assert.ErrorIs(t, err, boom)
}
