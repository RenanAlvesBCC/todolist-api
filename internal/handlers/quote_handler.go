package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/utils"
)

type QuoteProvider interface {
	AddQuote(listID, userID uint, text string) (*models.QuoteItem, error)
	ListQuotes(listID, userID uint) ([]models.QuoteItem, error)
	DeleteQuote(listID, quoteID, userID uint) error
}

type QuoteHandler struct {
	svc QuoteProvider
}

func NewQuoteHandler(svc QuoteProvider) *QuoteHandler {
	return &QuoteHandler{svc: svc}
}

type addQuoteInput struct {
	Text string `json:"text" binding:"required"`
}

func (h *QuoteHandler) Add(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var input addQuoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	q, err := h.svc.AddQuote(uint(listID), getUserID(c), input.Text)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, q)
}

func (h *QuoteHandler) List(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	items, err := h.svc.ListQuotes(uint(listID), getUserID(c))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *QuoteHandler) Delete(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}
	quoteID, err := strconv.ParseUint(c.Param("quoteId"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id de orçamento inválido")
		return
	}

	if err := h.svc.DeleteQuote(uint(listID), uint(quoteID), getUserID(c)); err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
