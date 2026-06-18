package main

import (
	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/database"
	"github.com/RenanAlvesBCC/todolist-api/internal/routes"
)

func main() {
	database.Connect()

	router := gin.Default()
	routes.SetupRoutes(router)
	router.Run(":8080")
}
