package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/middleware"
	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

const validRobotToken = "rmx_valido"

type mockAuthenticator struct{ err error }

func (m *mockAuthenticator) Authenticate(token string) (*models.IntegrationToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	if token != validRobotToken {
		return nil, services.ErrInvalidIntegrationToken
	}
	return &models.IntegrationToken{ID: 1, WorkspaceID: 10}, nil
}

type mockSyncProvider struct {
	err         error
	gotToken    *models.IntegrationToken
	gotInput    services.ImportInput
	gotUser     uint
	gotRunID    uint
	gotMessage  string
	gotLimit    int
	importCalls int
}

func (m *mockSyncProvider) ImportWithToken(token *models.IntegrationToken, in services.ImportInput) (*services.ImportResult, error) {
	m.importCalls++
	m.gotToken, m.gotInput = token, in
	if m.err != nil {
		return nil, m.err
	}
	return &services.ImportResult{RunID: 12, Status: "running", Counters: services.SyncCounters{CustomersCreated: 1}}, nil
}
func (m *mockSyncProvider) ImportManual(userID uint, in services.ImportInput) (*services.ImportResult, error) {
	m.gotUser, m.gotInput = userID, in
	if m.err != nil {
		return nil, m.err
	}
	return &services.ImportResult{RunID: 13, Status: "success"}, nil
}
func (m *mockSyncProvider) FailRun(token *models.IntegrationToken, runID uint, message string) (*models.SyncRun, error) {
	m.gotToken, m.gotRunID, m.gotMessage = token, runID, message
	if m.err != nil {
		return nil, m.err
	}
	return &models.SyncRun{ID: runID, Status: models.SyncStatusError, Error: message}, nil
}
func (m *mockSyncProvider) ListRuns(userID uint, limit int) ([]models.SyncRun, error) {
	m.gotLimit = limit
	if m.err != nil {
		return nil, m.err
	}
	return []models.SyncRun{{ID: 1, Status: "success"}}, nil
}

func setupSyncRouter(p *mockSyncProvider, auth IntegrationTokenAuthenticator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewOnMotorSyncHandler(p, auth)
	robot := router.Group("/api/integrations/onmotor")
	robot.Use(h.RequireIntegrationToken())
	{
		robot.POST("/import", h.Import)
		robot.POST("/runs/:id/fail", h.FailRun)
	}
	panel := router.Group("/api")
	panel.Use(middleware.AuthRequired())
	{
		panel.POST("/integrations/onmotor/import/manual", h.ImportManual)
		panel.GET("/integrations/onmotor/runs", h.ListRuns)
	}
	return router
}

func bearerRequest(router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func userJWT(t *testing.T) string {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret-sync")
	token, err := utils.GenerateToken(1, "dono@teste.com")
	require.NoError(t, err)
	return token
}

func TestOnMotorSyncHandler_ImportWithToken(t *testing.T) {
	p := &mockSyncProvider{}
	router := setupSyncRouter(p, &mockAuthenticator{})

	rec := bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/import", validRobotToken,
		`{"trigger":"schedule","customers":[{"source_id":123,"name":"Maria","cpf":"123.456.789-09"}],"final":false}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uint(10), p.gotToken.WorkspaceID)
	require.Len(t, p.gotInput.Customers, 1)
	assert.Equal(t, "123", p.gotInput.Customers[0].SourceID.Value)
	body := decodeObject(t, rec)
	assert.Equal(t, float64(12), body["run_id"])
	assert.Equal(t, float64(1), body["counters"].(map[string]any)["customers_created"])

	runID := uint(12)
	rec = bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/import", validRobotToken, `{"run_id":12,"final":true}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, runID, *p.gotInput.RunID)
	assert.True(t, p.gotInput.Final)
}

func TestOnMotorSyncHandler_TokenAuthentication(t *testing.T) {
	p := &mockSyncProvider{}
	router := setupSyncRouter(p, &mockAuthenticator{})
	jwt := userJWT(t)

	for _, tc := range []struct{ name, header string }{
		{"sem header", ""},
		{"JWT de usuário", "Bearer " + jwt},
		{"token desconhecido", "Bearer rmx_outro"},
		{"sem Bearer", validRobotToken},
		{"Bearer vazio", "Bearer  "},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/integrations/onmotor/import", strings.NewReader(`{}`))
		if tc.header != "" {
			req.Header.Set("Authorization", tc.header)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, tc.name)
	}
	assert.Zero(t, p.importCalls, "nada é importado sem token de integração válido")

	// Token de integração não serve nas rotas com JWT.
	rec := bearerRequest(router, http.MethodGet, "/api/integrations/onmotor/runs", validRobotToken, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	rec = bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/import/manual", validRobotToken, `{}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	broken := setupSyncRouter(p, &mockAuthenticator{err: assert.AnError})
	rec = bearerRequest(broken, http.MethodPost, "/api/integrations/onmotor/import", validRobotToken, `{}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestOnMotorSyncHandler_PanelRoutes(t *testing.T) {
	p := &mockSyncProvider{}
	router := setupSyncRouter(p, &mockAuthenticator{})
	jwt := userJWT(t)

	rec := bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/import/manual", jwt, `{"customers":[{"source_id":"1","name":"A"}],"final":true}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uint(1), p.gotUser)
	assert.Len(t, p.gotInput.Customers, 1)

	rec = bearerRequest(router, http.MethodGet, "/api/integrations/onmotor/runs?limit=5", jwt, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 5, p.gotLimit)
	assert.Len(t, decodeObject(t, rec)["runs"], 1)
}

func TestOnMotorSyncHandler_FailRun(t *testing.T) {
	p := &mockSyncProvider{}
	router := setupSyncRouter(p, &mockAuthenticator{})
	rec := bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/runs/12/fail", validRobotToken, `{"error":"OnMotor fora do ar"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uint(12), p.gotRunID)
	assert.Equal(t, "OnMotor fora do ar", p.gotMessage)
	assert.Equal(t, "error", decodeObject(t, rec)["status"])

	assert.Equal(t, http.StatusBadRequest, bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/runs/x/fail", validRobotToken, `{}`).Code)
	assert.Equal(t, http.StatusBadRequest, bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/runs/1/fail", validRobotToken, `{"error":1}`).Code)
}

func TestOnMotorSyncHandler_ErrorMapping(t *testing.T) {
	jwt := userJWT(t)
	cases := []struct {
		err  error
		code int
	}{
		{services.ErrIntegrationForbidden, http.StatusForbidden},
		{services.ErrNoWorkspace, http.StatusNotFound},
		{services.ErrSyncRunNotFound, http.StatusNotFound},
		{services.ErrSyncRunFinished, http.StatusConflict},
		{services.ErrImportBatchTooLarge, http.StatusRequestEntityTooLarge},
		{services.ErrInvalidTrigger, http.StatusBadRequest},
		{assert.AnError, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		router := setupSyncRouter(&mockSyncProvider{err: tc.err}, &mockAuthenticator{})
		for _, rec := range []*httptest.ResponseRecorder{
			bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/import", validRobotToken, `{}`),
			bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/runs/1/fail", validRobotToken, `{}`),
			bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/import/manual", jwt, `{}`),
			bearerRequest(router, http.MethodGet, "/api/integrations/onmotor/runs", jwt, ""),
		} {
			assert.Equal(t, tc.code, rec.Code, tc.err.Error())
		}
	}
}

func TestOnMotorSyncHandler_BodyLimits(t *testing.T) {
	p := &mockSyncProvider{}
	router := setupSyncRouter(p, &mockAuthenticator{})
	jwt := userJWT(t)

	huge := `{"customers":[{"notes":"` + strings.Repeat("x", MaxImportBodyBytes) + `"}]}`
	rec := bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/import", validRobotToken, huge)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Equal(t, services.ErrImportBatchTooLarge.Error(), decodeObject(t, rec)["error"])

	for _, body := range []string{`{`, `{"customers":{}}`, `{"run_id":"x"}`} {
		rec = bearerRequest(router, http.MethodPost, "/api/integrations/onmotor/import", validRobotToken, body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, body)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/integrations/onmotor/import/manual", bytes.NewBufferString(`[`))
	req.Header.Set("Authorization", "Bearer "+jwt)
	manual := httptest.NewRecorder()
	router.ServeHTTP(manual, req)
	assert.Equal(t, http.StatusBadRequest, manual.Code)
	assert.Zero(t, p.importCalls)
}
