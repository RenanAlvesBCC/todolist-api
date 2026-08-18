package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func TestSecurityRepository_BlacklistTokenAndIsBlacklisted(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSecurityRepository(db)

	require.NoError(t, repo.BlacklistToken("tok-1", time.Now().Add(time.Hour)))

	ok, err := repo.IsTokenBlacklisted("tok-1")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = repo.IsTokenBlacklisted("unknown")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSecurityRepository_IsTokenBlacklisted_ExpiredIsFalse(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSecurityRepository(db)

	require.NoError(t, repo.BlacklistToken("expired", time.Now().Add(-time.Hour)))

	ok, err := repo.IsTokenBlacklisted("expired")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSecurityRepository_CleanExpiredTokens(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSecurityRepository(db)

	require.NoError(t, repo.BlacklistToken("alive", time.Now().Add(time.Hour)))
	require.NoError(t, repo.BlacklistToken("dead", time.Now().Add(-time.Minute)))

	require.NoError(t, repo.CleanExpiredTokens())

	ok, err := repo.IsTokenBlacklisted("alive")
	require.NoError(t, err)
	assert.True(t, ok)

	var count int64
	require.NoError(t, db.Model(&models.TokenBlacklist{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestSecurityRepository_LogAction(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSecurityRepository(db)
	uid := uint(1)

	repo.LogAction(&uid, "logout", "127.0.0.1", "ua", "", true)

	require.Eventually(t, func() bool {
		var count int64
		db.Model(&models.AuditLog{}).Count(&count)
		return count == 1
	}, time.Second, 10*time.Millisecond)
}

func TestSecurityRepository_ListLogs_CategoryPaginationAndActor(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSecurityRepository(db)

	ana := &models.User{Username: "ana", Password: "x"}
	bob := &models.User{Username: "bob", Password: "x"}
	require.NoError(t, db.Create(ana).Error)
	require.NoError(t, db.Create(bob).Error)

	now := time.Now()
	logs := []models.AuditLog{
		{UserID: &ana.ID, Action: "login_success", IPAddress: "1.1.1.1", Success: true, CreatedAt: now.Add(-3 * time.Hour)},
		{UserID: &ana.ID, Action: "logout", IPAddress: "1.1.1.1", Success: true, CreatedAt: now.Add(-2 * time.Hour)},
		{UserID: &bob.ID, Action: "login_failed", IPAddress: "2.2.2.2", Success: false, CreatedAt: now.Add(-time.Hour)},
		{UserID: &ana.ID, Action: "vehicle_updated", IPAddress: "1.1.1.1", Success: true, CreatedAt: now},
		{UserID: nil, Action: "register_success", IPAddress: "3.3.3.3", Success: true, CreatedAt: now.Add(time.Minute)},
	}
	for i := range logs {
		require.NoError(t, db.Create(&logs[i]).Error)
	}

	acesso, total, err := repo.ListLogs(AuditFilter{Category: "acesso", Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, acesso, 4)
	for _, row := range acesso {
		assert.Equal(t, "acesso", row.Category)
	}

	veiculo, total, err := repo.ListLogs(AuditFilter{Category: "veiculo", Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, veiculo, 1)
	assert.Equal(t, "vehicle_updated", veiculo[0].Action)
	assert.Equal(t, "ana", veiculo[0].ActorName)

	page1, total, err := repo.ListLogs(AuditFilter{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, page1, 2)

	page3, total, err := repo.ListLogs(AuditFilter{Page: 3, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, page3, 1)

	beyond, total, err := repo.ListLogs(AuditFilter{Page: 99, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Empty(t, beyond)

	actorRows, total, err := repo.ListLogs(AuditFilter{ActorID: &bob.ID, Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, actorRows, 1)
	assert.Equal(t, bob.ID, actorRows[0].ActorID)
	assert.Equal(t, "bob", actorRows[0].ActorName)

	from := now.Add(-90 * time.Minute)
	to := now.Add(time.Hour)
	ranged, _, err := repo.ListLogs(AuditFilter{DateFrom: &from, DateTo: &to, Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(ranged), 2)

	defaults, total, err := repo.ListLogs(AuditFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, defaults, 5)
}
