package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

// MaxImportBodyBytes limita o corpo de um lote (3 listas de até 1000 registros cabem com folga).
const MaxImportBodyBytes = 20 << 20

const integrationTokenKey = "integration_token"

type IntegrationTokenAuthenticator interface {
	Authenticate(token string) (*models.IntegrationToken, error)
}

type OnMotorSyncProvider interface {
	ImportWithToken(token *models.IntegrationToken, in services.ImportInput) (*services.ImportResult, error)
	ImportManual(userID uint, in services.ImportInput) (*services.ImportResult, error)
	FailRun(token *models.IntegrationToken, runID uint, message string) (*models.SyncRun, error)
	ListRuns(userID uint, limit int) ([]models.SyncRun, error)
}

type OnMotorSyncHandler struct {
	svc  OnMotorSyncProvider
	auth IntegrationTokenAuthenticator
}

func NewOnMotorSyncHandler(svc OnMotorSyncProvider, auth IntegrationTokenAuthenticator) *OnMotorSyncHandler {
	return &OnMotorSyncHandler{svc: svc, auth: auth}
}

// RequireIntegrationToken autentica o robô por "Authorization: Bearer rmx_..." (não aceita JWT de usuário).
func (h *OnMotorSyncHandler) RequireIntegrationToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !found || strings.TrimSpace(token) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token de integração não fornecido"})
			return
		}
		t, err := h.auth.Authenticate(strings.TrimSpace(token))
		if err != nil {
			if errors.Is(err, services.ErrInvalidIntegrationToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "erro ao validar token de integração"})
			return
		}
		c.Set(integrationTokenKey, t)
		c.Next()
	}
}

func integrationToken(c *gin.Context) *models.IntegrationToken {
	return c.MustGet(integrationTokenKey).(*models.IntegrationToken)
}

func respondSyncError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrIntegrationForbidden):
		utils.RespondError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, services.ErrNoWorkspace), errors.Is(err, services.ErrSyncRunNotFound):
		utils.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrSyncRunFinished):
		utils.RespondError(c, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrImportBatchTooLarge):
		utils.RespondError(c, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, services.ErrInvalidTrigger):
		utils.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		utils.RespondError(c, http.StatusInternalServerError, "erro na sincronização")
	}
}

func bindImport(c *gin.Context) (services.ImportInput, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxImportBodyBytes)
	var in services.ImportInput
	if err := c.ShouldBindJSON(&in); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			utils.RespondError(c, http.StatusRequestEntityTooLarge, services.ErrImportBatchTooLarge.Error())
			return in, false
		}
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return in, false
	}
	return in, true
}

func (h *OnMotorSyncHandler) Import(c *gin.Context) {
	in, ok := bindImport(c)
	if !ok {
		return
	}
	result, err := h.svc.ImportWithToken(integrationToken(c), in)
	if err != nil {
		respondSyncError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *OnMotorSyncHandler) ImportManual(c *gin.Context) {
	in, ok := bindImport(c)
	if !ok {
		return
	}
	result, err := h.svc.ImportManual(getUserID(c), in)
	if err != nil {
		respondSyncError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

type failRunInput struct {
	Error string `json:"error"`
}

func (h *OnMotorSyncHandler) FailRun(c *gin.Context) {
	runID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var in failRunInput
	if err := c.ShouldBindJSON(&in); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}
	run, err := h.svc.FailRun(integrationToken(c), runID, in.Error)
	if err != nil {
		respondSyncError(c, err)
		return
	}
	c.JSON(http.StatusOK, run)
}

func (h *OnMotorSyncHandler) ListRuns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	runs, err := h.svc.ListRuns(getUserID(c), limit)
	if err != nil {
		respondSyncError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"runs": runs})
}
