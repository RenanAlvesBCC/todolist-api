package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func TestIntegrationRepository_Tokens(t *testing.T) {
	db := setupTestDB(t)
	repo := NewIntegrationRepository(db)
	first := &models.IntegrationToken{WorkspaceID: 1, Name: "robô", TokenHash: "h1", CreatedBy: 1}
	second := &models.IntegrationToken{WorkspaceID: 1, Name: "planilha", TokenHash: "h2", CreatedBy: 1}
	foreign := &models.IntegrationToken{WorkspaceID: 2, Name: "outra", TokenHash: "h3", CreatedBy: 5}
	for _, tk := range []*models.IntegrationToken{first, second, foreign} {
		require.NoError(t, repo.CreateToken(tk))
	}

	list, err := repo.ListTokens(1)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, second.ID, list[0].ID, "mais novo primeiro")

	_, err = repo.FindToken(1, foreign.ID)
	assert.Error(t, err)
	got, err := repo.FindToken(1, first.ID)
	require.NoError(t, err)
	assert.Equal(t, "robô", got.Name)

	active, err := repo.FindActiveTokenByHash("h1")
	require.NoError(t, err)
	assert.Equal(t, first.ID, active.ID)

	used := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	require.NoError(t, repo.TouchTokenUsed(first.ID, used))
	got, _ = repo.FindToken(1, first.ID)
	assert.True(t, used.Equal(*got.LastUsedAt))

	revokedAt := time.Date(2026, 11, 9, 11, 0, 0, 0, time.UTC)
	require.NoError(t, repo.RevokeToken(first.ID, revokedAt))
	require.NoError(t, repo.RevokeToken(first.ID, revokedAt.Add(time.Hour)), "revogar de novo mantém a data original")
	got, _ = repo.FindToken(1, first.ID)
	assert.True(t, revokedAt.Equal(*got.RevokedAt))

	active, err = repo.FindActiveTokenByHash("h1")
	require.NoError(t, err)
	assert.Nil(t, active, "revogado não autentica")
	active, err = repo.FindActiveTokenByHash("nao-existe")
	require.NoError(t, err)
	assert.Nil(t, active)
}
