package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

type mockQuoteProvider struct {
	addFunc    func(listID, userID uint, text string) (*models.QuoteItem, error)
	listFunc   func(listID, userID uint) ([]models.QuoteItem, error)
	deleteFunc func(listID, quoteID, userID uint) error
}

func (m *mockQuoteProvider) AddQuote(listID, userID uint, text string) (*models.QuoteItem, error) {
	if m.addFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.addFunc(listID, userID, text)
}
func (m *mockQuoteProvider) ListQuotes(listID, userID uint) ([]models.QuoteItem, error) {
	if m.listFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.listFunc(listID, userID)
}
func (m *mockQuoteProvider) DeleteQuote(listID, quoteID, userID uint) error {
	if m.deleteFunc == nil {
		return errors.New("not implemented")
	}
	return m.deleteFunc(listID, quoteID, userID)
}

func setupQuoteRouter(p *mockQuoteProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewQuoteHandler(p)
	g := router.Group("/api")
	g.Use(withUserContext(1))
	{
		g.POST("/lists/:id/quotes", h.Add)
		g.GET("/lists/:id/quotes", h.List)
		g.DELETE("/lists/:id/quotes/:quoteId", h.Delete)
	}
	return router
}

func TestQuoteHandler_Add_Success(t *testing.T) {
	router := setupQuoteRouter(&mockQuoteProvider{
		addFunc: func(listID, userID uint, text string) (*models.QuoteItem, error) {
			return &models.QuoteItem{TaskListID: listID, SubmittedBy: userID, Text: text}, nil
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/lists/3/quotes", map[string]string{"text": "Pastilha R$120"})
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestQuoteHandler_Add_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupQuoteRouter(&mockQuoteProvider{}), http.MethodPost, "/api/lists/abc/quotes", map[string]string{"text": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuoteHandler_Add_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupQuoteRouter(&mockQuoteProvider{}), http.MethodPost, "/api/lists/3/quotes", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuoteHandler_Add_ServiceErrorReturns400(t *testing.T) {
	router := setupQuoteRouter(&mockQuoteProvider{
		addFunc: func(listID, userID uint, text string) (*models.QuoteItem, error) {
			return nil, errors.New("texto é obrigatório")
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/lists/3/quotes", map[string]string{"text": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuoteHandler_List_Success(t *testing.T) {
	router := setupQuoteRouter(&mockQuoteProvider{
		listFunc: func(listID, userID uint) ([]models.QuoteItem, error) {
			return []models.QuoteItem{{Text: "Óleo"}}, nil
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/lists/3/quotes", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQuoteHandler_List_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupQuoteRouter(&mockQuoteProvider{}), http.MethodGet, "/api/lists/abc/quotes", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuoteHandler_List_NotFoundReturns404(t *testing.T) {
	router := setupQuoteRouter(&mockQuoteProvider{
		listFunc: func(listID, userID uint) ([]models.QuoteItem, error) {
			return nil, errors.New("lista não encontrada")
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/lists/3/quotes", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestQuoteHandler_Delete_Success(t *testing.T) {
	router := setupQuoteRouter(&mockQuoteProvider{
		deleteFunc: func(listID, quoteID, userID uint) error { return nil },
	})
	rec := jsonRequest(router, http.MethodDelete, "/api/lists/3/quotes/8", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestQuoteHandler_Delete_InvalidListIDReturns400(t *testing.T) {
	rec := jsonRequest(setupQuoteRouter(&mockQuoteProvider{}), http.MethodDelete, "/api/lists/abc/quotes/8", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuoteHandler_Delete_InvalidQuoteIDReturns400(t *testing.T) {
	rec := jsonRequest(setupQuoteRouter(&mockQuoteProvider{}), http.MethodDelete, "/api/lists/3/quotes/abc", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestQuoteHandler_Delete_NotFoundReturns404(t *testing.T) {
	router := setupQuoteRouter(&mockQuoteProvider{
		deleteFunc: func(listID, quoteID, userID uint) error { return errors.New("não encontrado") },
	})
	rec := jsonRequest(router, http.MethodDelete, "/api/lists/3/quotes/8", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
