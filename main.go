package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/RenanAlvesBCC/oficina-api/internal/database"
	"github.com/RenanAlvesBCC/oficina-api/internal/handlers"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
	"github.com/RenanAlvesBCC/oficina-api/internal/routes"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: arquivo .env não encontrado")
	}

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET não configurado")
	}

	database.Connect()

	// Repositories
	userRepo := repository.NewUserRepository(database.DB)
	listRepo := repository.NewTaskListRepository(database.DB)
	itemRepo := repository.NewTaskItemRepository(database.DB)
	secRepo := repository.NewSecurityRepository(database.DB)
	wsRepo := repository.NewWorkspaceRepository(database.DB)
	quoteRepo := repository.NewQuoteRepository(database.DB)
	flagRepo := repository.NewPendingFlagRepository(database.DB)
	assignRepo := repository.NewListAssignmentRepository(database.DB)

	// Limpeza periódica de tokens expirados em background
	utils.StartTokenCleanup(secRepo)

	// Services
	authService := services.NewAuthService(userRepo)
	listService := services.NewTaskListService(listRepo, itemRepo, wsRepo)
	wsService := services.NewWorkspaceService(wsRepo)
	quoteService := services.NewQuoteService(quoteRepo, listRepo, wsRepo)
	flagService := services.NewPendingFlagService(flagRepo, listRepo, wsRepo)
	assignService := services.NewAssignmentService(assignRepo, wsRepo, listRepo)
	auditService := services.NewAuditService(secRepo, wsRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, secRepo)
	listHandler := handlers.NewTaskListHandler(listService)
	wsHandler := handlers.NewWorkspaceHandler(wsService)
	quoteHandler := handlers.NewQuoteHandler(quoteService)
	flagHandler := handlers.NewPendingFlagHandler(flagService)
	assignmentHandler := handlers.NewAssignmentHandler(assignService)
	auditHandler := handlers.NewAuditHandler(auditService)

	router := gin.Default()
	routes.SetupRoutes(router, authHandler, listHandler, wsHandler, quoteHandler, flagHandler, assignmentHandler, auditHandler, secRepo)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor rodando na porta %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Erro ao iniciar servidor: ", err)
	}
}
