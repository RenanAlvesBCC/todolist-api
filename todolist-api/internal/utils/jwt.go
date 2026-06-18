package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// getSecretKey lê a chave usada para assinar os tokens, a partir da variável de ambiente.
func getSecretKey() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "chave-temporaria-mude-isso" // só pra não travar se você esquecer o .env
	}
	return []byte(secret)
}

// GenerateToken cria um token JWT válido por 24h, contendo o ID e o username do usuário.
func GenerateToken(userID uint, username string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSecretKey())
}
