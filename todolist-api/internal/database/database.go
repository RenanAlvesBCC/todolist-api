package database

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

// DB é a conexão global com o banco, usada pelos repositories.
var DB *gorm.DB

// Connect abre a conexão com o SQLite e roda as migrations.
func Connect() {
	db, err := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Falha ao conectar ao banco de dados: ", err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatal("Falha ao migrar o banco de dados: ", err)
	}

	DB = db
	log.Println("Banco de dados conectado e migrado com sucesso")
}
