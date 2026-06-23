package utils

import (
	"log"
	"time"

	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
)

// StartTokenCleanup roda em background e remove tokens expirados
// da blacklist a cada hora, evitando crescimento infinito da tabela.
func StartTokenCleanup(secRepo *repository.SecurityRepository) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := secRepo.CleanExpiredTokens(); err != nil {
				log.Printf("Erro ao limpar tokens expirados: %v", err)
			} else {
				log.Println("Tokens expirados removidos da blacklist")
			}
		}
	}()
}
