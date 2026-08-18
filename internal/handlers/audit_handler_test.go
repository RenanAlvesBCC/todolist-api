package handlers

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
)

type mockAuditProvider struct {
	listFunc func(userID uint, category string, actorID *uint, dateFrom, dateTo *time.Time, page, limit int) (*services.PaginatedAudit, error)
}

func (m *mockAuditProvider) List(userID uint, category string, actorID *uint, dateFrom, dateTo *time.Time, page, limit int) (*services.PaginatedAudit, error) {
	if m.listFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.listFunc(userID, category, actorID, dateFrom, dateTo, page, limit)
}

func setupAuditRouter(p *mockAuditProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewAuditHandler(p)
	g := router.Group("/api")
	g.Use(withUserContext(1))
	g.GET("/audit", h.List)
	return router
}

func TestAuditHandler_List_SuccessWithDatesAndActor(t *testing.T) {
	var gotActor *uint
	var gotFrom, gotTo *time.Time
	var gotCategory string
	router := setupAuditRouter(&mockAuditProvider{
		listFunc: func(userID uint, category string, actorID *uint, dateFrom, dateTo *time.Time, page, limit int) (*services.PaginatedAudit, error) {
			gotActor, gotFrom, gotTo, gotCategory = actorID, dateFrom, dateTo, category
			return &services.PaginatedAudit{Logs: []repository.AuditLogRow{{ID: 1}}, Total: 1, Page: page, TotalPages: 1}, nil
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/audit?page=2&limit=5&category=acesso&actor_id=9&date_from=2026-01-01&date_to=2026-01-31T23:59:59Z", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, gotActor)
	assert.Equal(t, uint(9), *gotActor)
	assert.Equal(t, "acesso", gotCategory)
	require.NotNil(t, gotFrom)
	require.NotNil(t, gotTo)
}

func TestAuditHandler_List_InvalidActorIDReturns400(t *testing.T) {
	rec := jsonRequest(setupAuditRouter(&mockAuditProvider{}), http.MethodGet, "/api/audit?actor_id=abc", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_List_InvalidDateFromReturns400(t *testing.T) {
	rec := jsonRequest(setupAuditRouter(&mockAuditProvider{}), http.MethodGet, "/api/audit?date_from=not-a-date", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_List_InvalidDateToReturns400(t *testing.T) {
	rec := jsonRequest(setupAuditRouter(&mockAuditProvider{}), http.MethodGet, "/api/audit?date_to=32/13/2026", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_List_ForbiddenReturns403(t *testing.T) {
	router := setupAuditRouter(&mockAuditProvider{
		listFunc: func(userID uint, category string, actorID *uint, dateFrom, dateTo *time.Time, page, limit int) (*services.PaginatedAudit, error) {
			return nil, errors.New("apenas gerentes podem ver a auditoria")
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/audit", nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestParseOptionalDate_EmptyAndFormats(t *testing.T) {
	got, err := parseOptionalDate("")
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = parseOptionalDate("2026-08-18")
	require.NoError(t, err)
	require.NotNil(t, got)

	got, err = parseOptionalDate("2026-08-18T10:00:00Z")
	require.NoError(t, err)
	require.NotNil(t, got)
}
