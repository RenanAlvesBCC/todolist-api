package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

type mockPendingFlagProvider struct {
	addFunc     func(listID, userID uint, flagType models.FlagType, note string) (*models.PendingFlag, error)
	resolveFunc func(listID, flagID, userID uint) error
	listFunc    func(listID, userID uint) ([]models.PendingFlag, error)
}

func (m *mockPendingFlagProvider) AddFlag(listID, userID uint, flagType models.FlagType, note string) (*models.PendingFlag, error) {
	if m.addFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.addFunc(listID, userID, flagType, note)
}
func (m *mockPendingFlagProvider) ResolveFlag(listID, flagID, userID uint) error {
	if m.resolveFunc == nil {
		return errors.New("not implemented")
	}
	return m.resolveFunc(listID, flagID, userID)
}
func (m *mockPendingFlagProvider) ListFlags(listID, userID uint) ([]models.PendingFlag, error) {
	if m.listFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.listFunc(listID, userID)
}

func setupFlagRouter(p *mockPendingFlagProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewPendingFlagHandler(p)
	g := router.Group("/api")
	g.Use(withUserContext(1))
	{
		g.POST("/lists/:id/flags", h.Add)
		g.GET("/lists/:id/flags", h.List)
		g.PATCH("/lists/:id/flags/:flagId/resolve", h.Resolve)
	}
	return router
}

func TestPendingFlagHandler_Add_Success(t *testing.T) {
	router := setupFlagRouter(&mockPendingFlagProvider{
		addFunc: func(listID, userID uint, flagType models.FlagType, note string) (*models.PendingFlag, error) {
			return &models.PendingFlag{TaskListID: listID, CreatedBy: userID, FlagType: flagType, Note: note}, nil
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/lists/3/flags", map[string]string{
		"flag_type": "procurando_peca",
		"note":      "disco",
	})
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestPendingFlagHandler_Add_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupFlagRouter(&mockPendingFlagProvider{}), http.MethodPost, "/api/lists/abc/flags", map[string]string{"flag_type": "outro", "note": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPendingFlagHandler_Add_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupFlagRouter(&mockPendingFlagProvider{}), http.MethodPost, "/api/lists/3/flags", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPendingFlagHandler_Add_ServiceErrorReturns400(t *testing.T) {
	router := setupFlagRouter(&mockPendingFlagProvider{
		addFunc: func(listID, userID uint, flagType models.FlagType, note string) (*models.PendingFlag, error) {
			return nil, errors.New("nota obrigatória")
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/lists/3/flags", map[string]string{"flag_type": "outro", "note": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPendingFlagHandler_Resolve_Success(t *testing.T) {
	router := setupFlagRouter(&mockPendingFlagProvider{
		resolveFunc: func(listID, flagID, userID uint) error { return nil },
	})
	rec := jsonRequest(router, http.MethodPatch, "/api/lists/3/flags/7/resolve", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestPendingFlagHandler_Resolve_InvalidListIDReturns400(t *testing.T) {
	rec := jsonRequest(setupFlagRouter(&mockPendingFlagProvider{}), http.MethodPatch, "/api/lists/abc/flags/7/resolve", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPendingFlagHandler_Resolve_InvalidFlagIDReturns400(t *testing.T) {
	rec := jsonRequest(setupFlagRouter(&mockPendingFlagProvider{}), http.MethodPatch, "/api/lists/3/flags/abc/resolve", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPendingFlagHandler_Resolve_ServiceErrorReturns400(t *testing.T) {
	router := setupFlagRouter(&mockPendingFlagProvider{
		resolveFunc: func(listID, flagID, userID uint) error { return errors.New("já resolvida") },
	})
	rec := jsonRequest(router, http.MethodPatch, "/api/lists/3/flags/7/resolve", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPendingFlagHandler_List_Success(t *testing.T) {
	router := setupFlagRouter(&mockPendingFlagProvider{
		listFunc: func(listID, userID uint) ([]models.PendingFlag, error) {
			return []models.PendingFlag{{FlagType: models.FlagProcurandoPeca}}, nil
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/lists/3/flags", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPendingFlagHandler_List_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupFlagRouter(&mockPendingFlagProvider{}), http.MethodGet, "/api/lists/abc/flags", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPendingFlagHandler_List_NotFoundReturns404(t *testing.T) {
	router := setupFlagRouter(&mockPendingFlagProvider{
		listFunc: func(listID, userID uint) ([]models.PendingFlag, error) {
			return nil, errors.New("lista não encontrada")
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/lists/3/flags", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
