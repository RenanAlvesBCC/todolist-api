package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

type mockAuthRegistrar struct {
	registerFunc func(username, password string) error
	loginFunc    func(username, password string) (string, error)
}

func (m *mockAuthRegistrar) Register(username, password string) error {
	if m.registerFunc == nil {
		return nil
	}
	return m.registerFunc(username, password)
}
func (m *mockAuthRegistrar) Login(username, password string) (string, error) {
	if m.loginFunc == nil {
		return "", errors.New("fail")
	}
	return m.loginFunc(username, password)
}

type mockTokenAuditor struct {
	actions      []string
	blacklistErr error
	blacklisted  string
}

func (m *mockTokenAuditor) LogAction(userID *uint, action, ip, userAgent, details string, success bool) {
	m.actions = append(m.actions, action)
}
func (m *mockTokenAuditor) BlacklistToken(token string, expiresAt time.Time) error {
	if m.blacklistErr != nil {
		return m.blacklistErr
	}
	m.blacklisted = token
	return nil
}

func setupAuthRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/register", h.Register)
	router.POST("/login", h.Login)
	api := router.Group("/api")
	api.Use(withUserContext(1))
	api.POST("/logout", h.Logout)
	return router
}

func TestAuthHandler_Register_Success(t *testing.T) {
	auditor := &mockTokenAuditor{}
	h := &AuthHandler{authService: &mockAuthRegistrar{}, securityRepo: auditor}
	rec := jsonRequest(setupAuthRouter(h), http.MethodPost, "/register", map[string]string{
		"username": "ana", "password": "senha123",
	})
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "usuário criado com sucesso")
	assert.Equal(t, []string{"register_success"}, auditor.actions)
}

func TestAuthHandler_Register_InvalidBodyReturns400(t *testing.T) {
	h := &AuthHandler{authService: &mockAuthRegistrar{}, securityRepo: &mockTokenAuditor{}}
	rec := jsonRequest(setupAuthRouter(h), http.MethodPost, "/register", map[string]string{"username": "ana"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Register_ConflictReturns409(t *testing.T) {
	auditor := &mockTokenAuditor{}
	h := &AuthHandler{
		authService: &mockAuthRegistrar{
			registerFunc: func(username, password string) error { return errors.New("duplicado") },
		},
		securityRepo: auditor,
	}
	rec := jsonRequest(setupAuthRouter(h), http.MethodPost, "/register", map[string]string{
		"username": "ana", "password": "senha123",
	})
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, []string{"register_failed"}, auditor.actions)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-handlers")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	token, err := utils.GenerateToken(1, "ana")
	require.NoError(t, err)

	auditor := &mockTokenAuditor{}
	h := &AuthHandler{
		authService: &mockAuthRegistrar{
			loginFunc: func(username, password string) (string, error) { return token, nil },
		},
		securityRepo: auditor,
	}
	rec := jsonRequest(setupAuthRouter(h), http.MethodPost, "/login", map[string]string{
		"username": "ana", "password": "senha123",
	})
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"token"`)
	assert.Equal(t, []string{"login_success"}, auditor.actions)
}

func TestAuthHandler_Login_InvalidBodyReturns400(t *testing.T) {
	h := &AuthHandler{authService: &mockAuthRegistrar{}, securityRepo: &mockTokenAuditor{}}
	rec := jsonRequest(setupAuthRouter(h), http.MethodPost, "/login", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Login_UnauthorizedReturns401(t *testing.T) {
	auditor := &mockTokenAuditor{}
	h := &AuthHandler{
		authService: &mockAuthRegistrar{
			loginFunc: func(username, password string) (string, error) { return "", errors.New("usuário ou senha incorretos") },
		},
		securityRepo: auditor,
	}
	rec := jsonRequest(setupAuthRouter(h), http.MethodPost, "/login", map[string]string{
		"username": "ana", "password": "errada",
	})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, []string{"login_failed"}, auditor.actions)
}

func TestAuthHandler_Logout_SuccessReturns204(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-handlers")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	token, err := utils.GenerateToken(1, "ana")
	require.NoError(t, err)

	auditor := &mockTokenAuditor{}
	h := &AuthHandler{authService: &mockAuthRegistrar{}, securityRepo: auditor}
	router := setupAuthRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Equal(t, token, auditor.blacklisted)
	assert.Equal(t, []string{"logout"}, auditor.actions)
}

func TestAuthHandler_Logout_MissingTokenReturns400(t *testing.T) {
	h := &AuthHandler{authService: &mockAuthRegistrar{}, securityRepo: &mockTokenAuditor{}}
	rec := jsonRequest(setupAuthRouter(h), http.MethodPost, "/api/logout", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Logout_InvalidTokenReturns400(t *testing.T) {
	h := &AuthHandler{authService: &mockAuthRegistrar{}, securityRepo: &mockTokenAuditor{}}
	router := setupAuthRouter(h)
	req := httptest.NewRequest(http.MethodPost, "/api/logout", bytes.NewReader(nil))
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuthHandler_Logout_BlacklistErrorReturns500(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-handlers")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	token, err := utils.GenerateToken(1, "ana")
	require.NoError(t, err)

	h := &AuthHandler{
		authService:  &mockAuthRegistrar{},
		securityRepo: &mockTokenAuditor{blacklistErr: errors.New("db")},
	}
	router := setupAuthRouter(h)
	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAuthHandler_Login_InvalidJWTSkipsSuccessLog(t *testing.T) {
	auditor := &mockTokenAuditor{}
	h := &AuthHandler{
		authService: &mockAuthRegistrar{
			loginFunc: func(username, password string) (string, error) { return "not-a-jwt", nil },
		},
		securityRepo: auditor,
	}
	rec := jsonRequest(setupAuthRouter(h), http.MethodPost, "/login", map[string]string{
		"username": "ana", "password": "senha123",
	})
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, auditor.actions)
}

func TestNewAuthHandler_AcceptsConcreteTypes(t *testing.T) {
	h := NewAuthHandler(nil, nil)
	require.NotNil(t, h)
}
