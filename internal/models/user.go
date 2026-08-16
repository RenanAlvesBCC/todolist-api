package models

// User representa a tabela de usuários no banco de dados.
type User struct {
	Base
	Username string `gorm:"unique;not null" json:"username"`
	Password string `json:"-"` // "-" evita que o hash da senha apareça em qualquer resposta JSON
}
