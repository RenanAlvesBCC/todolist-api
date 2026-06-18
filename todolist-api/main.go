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

	taskRepo := repository.NewTaskRepository(database.DB)
	taskService := services.NewTaskService(taskRepo)
	taskHandler := handlers.NewTaskHandler(taskService)

	router := gin.Default()
	routes.SetupRoutes(router, authHandler, taskHandler)
	router.Run(":8080")
}
