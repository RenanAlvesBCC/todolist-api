package models

import "gorm.io/gorm"

// User representa a tabela de usuários no banco de dados.
// gorm.Model adiciona automaticamente os campos ID, CreatedAt, UpdatedAt e DeletedAt.
type User struct {
	gorm.Model
	Username string `gorm:"unique;not null" json:"username"`
	Password string `json:"-"` // "-" evita que o hash da senha apareça em qualquer resposta JSON
}
