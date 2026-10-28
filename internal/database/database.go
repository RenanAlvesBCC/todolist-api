package database

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
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

	migrateLegacyStatuses(db)
	if err := backfillDeliveredAt(db); err != nil {
		log.Println("Aviso: falha ao preencher delivered_at: ", err)
	}

	DB = db
	log.Println("Banco de dados conectado e migrado com sucesso")
}

func migrateLegacyStatuses(db *gorm.DB) {
	_ = db.Model(&models.TaskList{}).Where("status = ?", "aprovado").Update("status", models.StatusAguardandoRetirada)
	// No SET, updated_at do lado direito ainda é o valor anterior (Postgres e SQLite).
	_ = db.Model(&models.TaskList{}).Where("status = ?", "concluido").Updates(map[string]interface{}{
		"status":       models.StatusEntregue,
		"delivered_at": gorm.Expr("COALESCE(delivered_at, updated_at)"),
	})
}

// backfillDeliveredAt usa updated_at como melhor estimativa da entrega para
// veículos já entregues antes de delivered_at existir. Idempotente.
func backfillDeliveredAt(db *gorm.DB) error {
	return db.Model(&models.TaskList{}).
		Where("status = ? AND delivered_at IS NULL", models.StatusEntregue).
		UpdateColumn("delivered_at", gorm.Expr("updated_at")).Error
}

func openDB() (*gorm.DB, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	}
	return gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
}
