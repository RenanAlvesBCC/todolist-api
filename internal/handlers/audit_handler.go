package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/oficina-api/internal/services"
	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

type AuditProvider interface {
	List(userID uint, category string, actorID *uint, dateFrom, dateTo *time.Time, page, limit int) (*services.PaginatedAudit, error)
}

type AuditHandler struct {
	svc AuditProvider
}

func NewAuditHandler(svc AuditProvider) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	category := c.Query("category")

	var actorID *uint
	if raw := c.Query("actor_id"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "actor_id inválido")
			return
		}
		id := uint(parsed)
		actorID = &id
	}

	dateFrom, err := parseOptionalDate(c.Query("date_from"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "date_from inválido")
		return
	}
	dateTo, err := parseOptionalDate(c.Query("date_to"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "date_to inválido")
		return
	}

	result, err := h.svc.List(getUserID(c), category, actorID, dateFrom, dateTo, page, limit)
	if err != nil {
		utils.RespondError(c, http.StatusForbidden, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func parseOptionalDate(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, err
		}
	}
	return &parsed, nil
}
