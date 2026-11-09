package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/oficina-api/internal/handlers"
	"github.com/RenanAlvesBCC/oficina-api/internal/middleware"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

func SetupRoutes(
	router *gin.Engine,
	authHandler *handlers.AuthHandler,
	listHandler *handlers.TaskListHandler,
	workspaceHandler *handlers.WorkspaceHandler,
	quoteHandler *handlers.QuoteHandler,
	flagHandler *handlers.PendingFlagHandler,
	assignmentHandler *handlers.AssignmentHandler,
	auditHandler *handlers.AuditHandler,
	noteHandler *handlers.VehicleNoteHandler,
	customerHandler *handlers.CustomerHandler,
	customerVehicleHandler *handlers.CustomerVehicleHandler,
	tokenHandler *handlers.IntegrationTokenHandler,
	syncHandler *handlers.OnMotorSyncHandler,
	secRepo *repository.SecurityRepository,
) {
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RateLimitGlobal())
	router.GET("/", handlers.HomeHandler)
	// O Render sonda a raiz com HEAD ao subir o deploy.
	router.HEAD("/", handlers.HomeHandler)

	auth := router.Group("/")
	auth.Use(middleware.RateLimitAuth())
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Convite pode ser previsualizando sem autenticação
	router.GET("/invites/:code/preview", workspaceHandler.InvitePreview)

	// Robô de sincronização OnMotor: token de integração (rmx_...), nunca o JWT de usuário.
	integration := router.Group("/api/integrations/onmotor")
	integration.Use(middleware.RateLimitIntegration())
	integration.Use(syncHandler.RequireIntegrationToken())
	{
		integration.POST("/import", syncHandler.Import)
		integration.POST("/runs/:id/fail", syncHandler.FailRun)
	}

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

		protected.GET("/audit", auditHandler.List)

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
		protected.PUT("/lists/:id/priority", listHandler.ChangePriority)
		protected.PUT("/lists/:id/checkup-video", listHandler.SetCheckupVideo)

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

		// Informações importantes (exclusão lógica)
		protected.GET("/lists/:id/notes", noteHandler.List)
		protected.POST("/lists/:id/notes", noteHandler.Add)
		protected.DELETE("/lists/:id/notes/:noteId", noteHandler.Delete)

		// Clientes (só dono/gerente; dados pessoais nunca vão para o mecânico)
		protected.GET("/customers", customerHandler.List)
		protected.POST("/customers", customerHandler.Create)
		protected.GET("/customers/:id", customerHandler.Get)
		protected.PUT("/customers/:id", customerHandler.Update)
		protected.GET("/customers/:id/vehicles", customerVehicleHandler.List)
		protected.POST("/customers/:id/vehicles", customerVehicleHandler.Create)
		protected.GET("/customer-vehicles/lookup", customerVehicleHandler.Lookup)
		protected.PUT("/customer-vehicles/:id", customerVehicleHandler.Update)
		protected.GET("/customer-vehicles/:id/history", customerVehicleHandler.History)

		// Tokens de integração (só dono; o texto do token só aparece na criação)
		protected.GET("/integrations/tokens", tokenHandler.List)
		protected.POST("/integrations/tokens", tokenHandler.Create)
		protected.DELETE("/integrations/tokens/:id", tokenHandler.Revoke)

		// Sincronização OnMotor pelo painel (dono/gerente): planilha e status
		protected.POST("/integrations/onmotor/import/manual", syncHandler.ImportManual)
		protected.GET("/integrations/onmotor/runs", syncHandler.ListRuns)
	}
}
