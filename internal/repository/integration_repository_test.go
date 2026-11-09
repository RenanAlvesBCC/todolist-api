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

func TestIntegrationRepository_Runs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewIntegrationRepository(db)
	base := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.CreateRun(&models.SyncRun{WorkspaceID: 1, StartedAt: base.Add(time.Duration(i) * time.Hour), Warnings: models.StringList{}}))
	}
	foreign := &models.SyncRun{WorkspaceID: 2, StartedAt: base}
	require.NoError(t, repo.CreateRun(foreign))

	runs, err := repo.ListRuns(1, 2)
	require.NoError(t, err)
	require.Len(t, runs, 2)
	assert.True(t, runs[0].StartedAt.After(runs[1].StartedAt), "mais recente primeiro")

	run, err := repo.FindRun(1, runs[0].ID)
	require.NoError(t, err)
	run.Status = models.SyncStatusPartial
	run.CustomersCreated = 5
	run.Warnings = models.StringList{"CPF duplicado em 2 clientes (OnMotor 1, cliente OnMotor 2); mantido no primeiro"}
	require.NoError(t, repo.SaveRun(run))
	again, err := repo.FindRun(1, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SyncStatusPartial, again.Status)
	assert.Equal(t, 5, again.CustomersCreated)
	assert.Equal(t, run.Warnings, again.Warnings)

	_, err = repo.FindRun(1, foreign.ID)
	assert.Error(t, err, "outra oficina")
}
