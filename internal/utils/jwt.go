package utils

import (
	"errors"
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

// ValidateToken verifica a assinatura e a validade do token, retornando os dados (claims) contidos nele.
func ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return getSecretKey(), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token inválido")
	}

	return claims, nil
}

// TokenExpiration extrai a data de expiração de um token válido.
// Usada na hora de fazer logout pra saber até quando guardar na blacklist.
func TokenExpiration(tokenString string) (time.Time, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return time.Time{}, err
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return time.Time{}, errors.New("campo exp ausente no token")
	}

	return time.Unix(int64(exp), 0), nil
}
