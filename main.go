package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/RenanAlvesBCC/todolist-api/internal/database"
	"github.com/RenanAlvesBCC/todolist-api/internal/handlers"
	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
	"github.com/RenanAlvesBCC/todolist-api/internal/routes"
	"github.com/RenanAlvesBCC/todolist-api/internal/services"
	"github.com/RenanAlvesBCC/todolist-api/internal/utils"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: arquivo .env não encontrado")
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

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, secRepo)
	listHandler := handlers.NewTaskListHandler(listService)
	wsHandler := handlers.NewWorkspaceHandler(wsService)
	quoteHandler := handlers.NewQuoteHandler(quoteService)
	flagHandler := handlers.NewPendingFlagHandler(flagService)
	assignmentHandler := handlers.NewAssignmentHandler(assignService)

	router := gin.Default()
	routes.SetupRoutes(router, authHandler, listHandler, wsHandler, quoteHandler, flagHandler, assignmentHandler, secRepo)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor rodando na porta %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Erro ao iniciar servidor: ", err)
	}
}
