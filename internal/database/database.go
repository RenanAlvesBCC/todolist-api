package database

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

var DB *gorm.DB

func Connect() {
	db, err := openDB()
	if err != nil {
		log.Fatal("Falha ao conectar ao banco de dados: ", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Workspace{},
		&models.WorkspaceMember{},
		&models.WorkspaceInvite{},
		&models.TaskList{},
		&models.TaskItem{},
		&models.ListAssignment{},
		&models.QuoteItem{},
		&models.PendingFlag{},
		&models.TokenBlacklist{},
		&models.AuditLog{},
	); err != nil {
		log.Fatal("Falha ao migrar o banco de dados: ", err)
	}

	DB = db
	log.Println("Banco de dados conectado e migrado com sucesso")
}

func openDB() (*gorm.DB, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	}
	return gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
}
