package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

// setupTestDB cria um banco SQLite novo e isolado, só pra esse teste.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// SQLite em memória + pool de conexões do Go é uma combinação traiçoeira:
	// se o driver abrir mais de uma conexão, cada uma enxerga um banco em
	// memória DIFERENTE e vazio. Forçamos uma única conexão pra evitar isso.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	err = db.AutoMigrate(
		&models.User{},
		&models.Workspace{},
		&models.WorkspaceMember{},
		&models.WorkspaceInvite{},
		&models.TaskList{},
		&models.TaskItem{},
		&models.TokenBlacklist{},
		&models.AuditLog{},
	)
	require.NoError(t, err)

	return db
}
