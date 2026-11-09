package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/RenanAlvesBCC/oficina-api/internal/handlers"
	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

// newApp monta a API como o main.go, sobre SQLite em memória.
func newApp(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	t.Setenv("JWT_SECRET", "routes-test-secret")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&models.User{}, &models.Workspace{}, &models.WorkspaceMember{}, &models.WorkspaceInvite{},
		&models.TaskList{}, &models.TaskItem{}, &models.ListAssignment{}, &models.QuoteItem{},
		&models.PendingFlag{}, &models.VehicleNote{}, &models.TokenBlacklist{}, &models.AuditLog{},
		&models.Customer{}, &models.CustomerVehicle{}, &models.ServiceRecord{}, &models.SyncRun{}, &models.IntegrationToken{},
	))

	userRepo := repository.NewUserRepository(db)
	listRepo := repository.NewTaskListRepository(db)
	itemRepo := repository.NewTaskItemRepository(db)
	secRepo := repository.NewSecurityRepository(db)
	wsRepo := repository.NewWorkspaceRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	integrationRepo := repository.NewIntegrationRepository(db)

	listService := services.NewTaskListService(listRepo, itemRepo, wsRepo).WithCustomerLinks(customerRepo)
	customerService := services.NewCustomerService(customerRepo, wsRepo)
	tokenService := services.NewIntegrationTokenService(integrationRepo, wsRepo)
	syncService := services.NewOnMotorSyncService(customerRepo, integrationRepo, wsRepo)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupRoutes(router,
		handlers.NewAuthHandler(services.NewAuthService(userRepo), secRepo),
		handlers.NewTaskListHandler(listService),
		handlers.NewWorkspaceHandler(services.NewWorkspaceService(wsRepo)),
		handlers.NewQuoteHandler(services.NewQuoteService(repository.NewQuoteRepository(db), listRepo, wsRepo)),
		handlers.NewPendingFlagHandler(services.NewPendingFlagService(repository.NewPendingFlagRepository(db), listRepo, wsRepo)),
		handlers.NewAssignmentHandler(services.NewAssignmentService(repository.NewListAssignmentRepository(db), wsRepo, listRepo)),
		handlers.NewAuditHandler(services.NewAuditService(secRepo, wsRepo)),
		handlers.NewVehicleNoteHandler(services.NewVehicleNoteService(repository.NewVehicleNoteRepository(db), listRepo, wsRepo)),
		handlers.NewCustomerHandler(customerService),
		handlers.NewCustomerVehicleHandler(customerService),
		handlers.NewIntegrationTokenHandler(tokenService),
		handlers.NewOnMotorSyncHandler(syncService, tokenService),
		secRepo,
	)
	return router, db
}

func member(t *testing.T, db *gorm.DB, wsID, userID uint, role models.WorkspaceRole) string {
	t.Helper()
	username := fmt.Sprintf("user%d@teste.com", userID)
	require.NoError(t, db.Create(&models.User{Base: models.Base{ID: userID}, Username: username, Password: "x"}).Error)
	require.NoError(t, db.Create(&models.WorkspaceMember{WorkspaceID: wsID, UserID: userID, Role: role}).Error)
	token, err := utils.GenerateToken(userID, username)
	require.NoError(t, err)
	return token
}

func call(t *testing.T, router http.Handler, method, path, token, body string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out, rec.Body.String()
}

const robotBatch = `{
  "trigger": "schedule",
  "customers": [{"source_id": "123", "name": "Maria Souza", "cpf": "123.456.789-09", "mobile": "11987654321", "email": "maria@exemplo.com"}],
  "vehicles": [{"source_id": "889", "customer_source_id": "123", "plate": "ABC-1D23", "manufacturer": "VW", "model": "Gol"}],
  "services": [{"source_id": "5001", "vehicle_source_id": "889", "opened_at": "2026-03-10 09:30", "mileage": 80000,
                "items": [{"kind": "servico", "description": "Troca de óleo"}]}],
  "final": true
}`

func TestRoutes_CustomerBaseEndToEnd(t *testing.T) {
	router, db := newApp(t)
	require.NoError(t, db.Create(&models.Workspace{Base: models.Base{ID: 1}, Name: "RMAXX", OwnerID: 1}).Error)
	require.NoError(t, db.Create(&models.Workspace{Base: models.Base{ID: 2}, Name: "Outra", OwnerID: 4}).Error)
	owner := member(t, db, 1, 1, models.RoleOwner)
	manager := member(t, db, 1, 2, models.RoleManager)
	mechanic := member(t, db, 1, 3, models.RoleEditor)
	otherOwner := member(t, db, 2, 4, models.RoleOwner)

	// Token de integração: só o dono cria; o texto só aparece aqui.
	code, _, _ := call(t, router, http.MethodPost, "/api/integrations/tokens", manager, `{"name":"robô"}`)
	assert.Equal(t, http.StatusForbidden, code)
	code, created, _ := call(t, router, http.MethodPost, "/api/integrations/tokens", owner, `{"name":"Robô OnMotor"}`)
	require.Equal(t, http.StatusCreated, code)
	robot := created["token"].(string)
	require.True(t, strings.HasPrefix(robot, "rmx_"))
	_, _, listed := call(t, router, http.MethodGet, "/api/integrations/tokens", owner, "")
	assert.NotContains(t, listed, robot)

	// Importação: JWT não vale; token de integração vale.
	code, _, _ = call(t, router, http.MethodPost, "/api/integrations/onmotor/import", owner, robotBatch)
	assert.Equal(t, http.StatusUnauthorized, code)
	code, imported, _ := call(t, router, http.MethodPost, "/api/integrations/onmotor/import", robot, robotBatch)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "success", imported["status"])

	// Token de integração não vale nas rotas de usuário.
	code, _, _ = call(t, router, http.MethodGet, "/api/customers", robot, "")
	assert.Equal(t, http.StatusUnauthorized, code)

	// Dono/gerente buscam pela placa; mecânico não acessa; outra oficina não enxerga.
	code, page, _ := call(t, router, http.MethodGet, "/api/customers?search=abc1d23", manager, "")
	require.Equal(t, http.StatusOK, code)
	customers := page["customers"].([]any)
	require.Len(t, customers, 1)
	maria := customers[0].(map[string]any)
	customerID := uint(maria["id"].(float64))
	vehicleID := uint(maria["vehicles"].([]any)[0].(map[string]any)["id"].(float64))
	assert.NotNil(t, maria["last_service_at"])

	for _, path := range []string{
		"/api/customers", fmt.Sprintf("/api/customers/%d", customerID), fmt.Sprintf("/api/customers/%d/vehicles", customerID),
		"/api/customer-vehicles/lookup?plate=ABC1D23", fmt.Sprintf("/api/customer-vehicles/%d/history", vehicleID),
		"/api/integrations/onmotor/runs",
	} {
		code, _, body := call(t, router, http.MethodGet, path, mechanic, "")
		assert.Equal(t, http.StatusForbidden, code, path)
		assert.NotContains(t, body, "12345678909", path)
	}
	code, _, _ = call(t, router, http.MethodGet, fmt.Sprintf("/api/customers/%d", customerID), otherOwner, "")
	assert.Equal(t, http.StatusNotFound, code)

	// Atendimento criado a partir do veículo do cliente; o mecânico atribuído só vê ids.
	code, list, _ := call(t, router, http.MethodPost, "/api/lists", owner, fmt.Sprintf(`{"customer_vehicle_id":%d}`, vehicleID))
	require.Equal(t, http.StatusCreated, code)
	assert.Equal(t, "VW Gol", list["title"])
	assert.Equal(t, "12345678909", list["customer_cpf"])
	listID := uint(list["id"].(float64))
	code, _, _ = call(t, router, http.MethodPost, fmt.Sprintf("/api/lists/%d/assignments", listID), owner, `{"user_id":3}`)
	require.Equal(t, http.StatusCreated, code)

	code, seen, raw := call(t, router, http.MethodGet, fmt.Sprintf("/api/lists/%d", listID), mechanic, "")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, float64(customerID), seen["customer_id"])
	assert.Equal(t, float64(vehicleID), seen["customer_vehicle_id"])
	for _, secret := range []string{"customer_cpf", "customer_whatsapp", "12345678909", "11987654321", "maria@exemplo.com"} {
		assert.NotContains(t, raw, secret)
	}

	code, history, _ := call(t, router, http.MethodGet, fmt.Sprintf("/api/customer-vehicles/%d/history", vehicleID), owner, "")
	require.Equal(t, http.StatusOK, code)
	records := history["records"].([]any)
	require.Len(t, records, 2)
	assert.Equal(t, "rmaxx", records[0].(map[string]any)["source"])
	assert.Equal(t, "onmotor", records[1].(map[string]any)["source"])

	code, runs, _ := call(t, router, http.MethodGet, "/api/integrations/onmotor/runs", manager, "")
	require.Equal(t, http.StatusOK, code)
	assert.Len(t, runs["runs"], 1)

	// Token revogado deixa de funcionar.
	code, _, _ = call(t, router, http.MethodDelete, fmt.Sprintf("/api/integrations/tokens/%v", created["id"]), owner, "")
	require.Equal(t, http.StatusNoContent, code)
	code, _, _ = call(t, router, http.MethodPost, "/api/integrations/onmotor/import", robot, robotBatch)
	assert.Equal(t, http.StatusUnauthorized, code)
}
