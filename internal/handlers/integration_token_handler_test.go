package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
)

type mockTokenProvider struct {
	err       error
	gotName   string
	revokedID uint
}

func (m *mockTokenProvider) ListTokens(userID uint) ([]models.IntegrationToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []models.IntegrationToken{{ID: 1, Name: "robô", TokenHash: "segredo-hash"}}, nil
}
func (m *mockTokenProvider) CreateToken(userID uint, name string) (*services.CreatedIntegrationToken, error) {
	m.gotName = name
	if m.err != nil {
		return nil, m.err
	}
	return &services.CreatedIntegrationToken{ID: 2, Name: name, Token: "rmx_abc", CreatedAt: time.Now()}, nil
}
func (m *mockTokenProvider) RevokeToken(userID, id uint) error {
	m.revokedID = id
	return m.err
}

func setupTokenRouter(p *mockTokenProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewIntegrationTokenHandler(p)
	g := router.Group("/api")
	g.Use(withUserContext(1))
	{
		g.GET("/integrations/tokens", h.List)
		g.POST("/integrations/tokens", h.Create)
		g.DELETE("/integrations/tokens/:id", h.Revoke)
	}
	return router
}

func TestIntegrationTokenHandler_HappyPaths(t *testing.T) {
	p := &mockTokenProvider{}
	router := setupTokenRouter(p)

	rec := jsonRequest(router, http.MethodGet, "/api/integrations/tokens", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "segredo-hash")
	assert.Len(t, decodeObject(t, rec)["tokens"], 1)

	rec = rawRequest(router, http.MethodPost, "/api/integrations/tokens", `{"name":"Robô OnMotor"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "Robô OnMotor", p.gotName)
	body := decodeObject(t, rec)
	assert.Equal(t, "rmx_abc", body["token"])
	assert.Equal(t, float64(2), body["id"])

	rec = jsonRequest(router, http.MethodDelete, "/api/integrations/tokens/2", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, uint(2), p.revokedID)
}

func TestIntegrationTokenHandler_Errors(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{services.ErrTokenOwnerOnly, http.StatusForbidden},
		{services.ErrNoWorkspace, http.StatusNotFound},
		{services.ErrTokenNotFound, http.StatusNotFound},
		{services.ErrTokenNameRequired, http.StatusBadRequest},
		{services.ErrTokenNameTooLong, http.StatusBadRequest},
		{assert.AnError, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		router := setupTokenRouter(&mockTokenProvider{err: tc.err})
		for _, rec := range []*httptest.ResponseRecorder{
			jsonRequest(router, http.MethodGet, "/api/integrations/tokens", nil),
			rawRequest(router, http.MethodPost, "/api/integrations/tokens", `{"name":"x"}`),
			jsonRequest(router, http.MethodDelete, "/api/integrations/tokens/1", nil),
		} {
			assert.Equal(t, tc.code, rec.Code, tc.err.Error())
		}
	}

	router := setupTokenRouter(&mockTokenProvider{})
	assert.Equal(t, http.StatusBadRequest, rawRequest(router, http.MethodPost, "/api/integrations/tokens", `{"name":1}`).Code)
	assert.Equal(t, http.StatusBadRequest, jsonRequest(router, http.MethodDelete, "/api/integrations/tokens/x", nil).Code)
}
