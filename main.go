package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/RenanAlvesBCC/todolist-api/internal/database"
	"github.com/RenanAlvesBCC/todolist-api/internal/handlers"
	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
	"github.com/RenanAlvesBCC/todolist-api/internal/routes"
	"github.com/RenanAlvesBCC/todolist-api/internal/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: arquivo .env não encontrado, usando variáveis do sistema")
	}

	database.Connect()

	userRepo := repository.NewUserRepository(database.DB)
	authService := services.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authService)

	listRepo := repository.NewTaskListRepository(database.DB)
	itemRepo := repository.NewTaskItemRepository(database.DB)
	listService := services.NewTaskListService(listRepo, itemRepo)
	listHandler := handlers.NewTaskListHandler(listService)

	router := gin.Default()
	routes.SetupRoutes(router, authHandler, listHandler)
	router.Run(":8080")
}
