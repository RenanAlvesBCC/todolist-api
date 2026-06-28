package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/handlers"
	"github.com/RenanAlvesBCC/todolist-api/internal/middleware"
	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
)

func SetupRoutes(
	router *gin.Engine,
	authHandler *handlers.AuthHandler,
	listHandler *handlers.TaskListHandler,
	workspaceHandler *handlers.WorkspaceHandler,
	quoteHandler *handlers.QuoteHandler,
	flagHandler *handlers.PendingFlagHandler,
	assignmentHandler *handlers.AssignmentHandler,
	secRepo *repository.SecurityRepository,
) {
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RateLimitGlobal())
	router.GET("/", handlers.HomeHandler)

	auth := router.Group("/")
	auth.Use(middleware.RateLimitAuth())
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Convite pode ser previsualizando sem autenticação
	router.GET("/invites/:code/preview", workspaceHandler.InvitePreview)

	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired())
	protected.Use(middleware.BlacklistCheck(secRepo))
	{
		protected.POST("/logout", authHandler.Logout)

		// Workspace
		protected.POST("/workspace", workspaceHandler.Create)
		protected.GET("/workspace", workspaceHandler.Get)
		protected.PUT("/workspace", workspaceHandler.Update)
		protected.POST("/workspace/invites", workspaceHandler.GenerateInvite)
		protected.GET("/workspace/invites", workspaceHandler.ListInvites)
		protected.POST("/invites/:code/accept", workspaceHandler.AcceptInvite)
		protected.GET("/workspace/members", workspaceHandler.ListMembers)
		protected.DELETE("/workspace/members/:userId", workspaceHandler.RemoveMember)

		// Listas e itens
		protected.GET("/lists", listHandler.List)
		protected.POST("/lists", listHandler.Create)
		protected.PUT("/lists/reorder", listHandler.ReorderLists)
		protected.GET("/lists/:id", listHandler.Get)
		protected.PUT("/lists/:id", listHandler.Update)
		protected.DELETE("/lists/:id", listHandler.Delete)
		protected.POST("/lists/:id/items", listHandler.AddItem)
		protected.PUT("/lists/:id/items/reorder", listHandler.ReorderItems)
		protected.PUT("/lists/:id/items/:itemId", listHandler.UpdateItem)
		protected.DELETE("/lists/:id/items/:itemId", listHandler.DeleteItem)
		protected.PUT("/lists/:id/status", listHandler.ChangeStatus)

		// Atribuições de mecânicos (Fase A.1)
		protected.GET("/lists/:id/assignments", assignmentHandler.List)
		protected.POST("/lists/:id/assignments", assignmentHandler.Assign)
		protected.DELETE("/lists/:id/assignments/:userId", assignmentHandler.Unassign)

		// Orçamentos (Fase C)
		protected.POST("/lists/:id/quotes", quoteHandler.Add)
		protected.GET("/lists/:id/quotes", quoteHandler.List)
		protected.DELETE("/lists/:id/quotes/:quoteId", quoteHandler.Delete)

		// Pendências (Fase C)
		protected.POST("/lists/:id/flags", flagHandler.Add)
		protected.PATCH("/lists/:id/flags/:flagId/resolve", flagHandler.Resolve)
		protected.GET("/lists/:id/flags", flagHandler.List)
	}
}
