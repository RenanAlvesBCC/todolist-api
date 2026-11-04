package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

// TaskListProvider descreve o que o handler precisa do service de listas.
type TaskListProvider interface {
	CreateList(userID uint, in services.VehicleInput) (*models.TaskList, error)
	ListAll(userID uint, search string, page, limit int, status string, mine bool) (*services.PaginatedTaskLists, error)
	GetList(listID, userID uint) (*models.TaskList, error)
	UpdateList(listID, userID uint, in services.VehicleInput) (*models.TaskList, error)
	DeleteList(listID, userID uint) error
	AddItem(listID, userID uint, text string) (*models.TaskItem, error)
	UpdateItem(listID, itemID, userID uint, text string, completed bool) (*models.TaskItem, error)
	DeleteItem(listID, itemID, userID uint) error
	ReorderLists(userID uint, orderedIDs []uint) error
	ReorderItems(listID, userID uint, orderedIDs []uint) error
	ChangeStatus(listID, userID uint, newStatus models.TaskListStatus) error
	ChangePriority(listID, userID uint, priority string) (*models.TaskList, error)
}

type TaskListHandler struct {
	listService TaskListProvider
}

func NewTaskListHandler(listService TaskListProvider) *TaskListHandler {
	return &TaskListHandler{listService: listService}
}

func getUserID(c *gin.Context) uint {
	userIDFloat := c.MustGet("user_id").(float64)
	return uint(userIDFloat)
}

// Ponteiros distinguem chave ausente (nil: no PUT mantém o valor) de chave enviada, mesmo "".
type listInput struct {
	Title            string  `json:"title" binding:"required"`
	Plate            string  `json:"plate"`
	Customer         string  `json:"customer"`
	CustomerCPF      *string `json:"customer_cpf"`
	CustomerCEP      *string `json:"customer_cep"`
	CustomerWhatsApp *string `json:"customer_whatsapp"`
	Priority         *string `json:"priority"`
}

func (in listInput) toVehicleInput() services.VehicleInput {
	return services.VehicleInput{
		Title:            in.Title,
		Plate:            in.Plate,
		Customer:         in.Customer,
		CustomerCPF:      in.CustomerCPF,
		CustomerCEP:      in.CustomerCEP,
		CustomerWhatsApp: in.CustomerWhatsApp,
		Priority:         in.Priority,
	}
}

func isVehicleValidationError(err error) bool {
	for _, target := range []error{
		services.ErrTitleRequired,
		services.ErrInvalidPriority,
		services.ErrInvalidCPF,
		services.ErrInvalidCEP,
		services.ErrInvalidWhatsApp,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

type createItemInput struct {
	Text string `json:"text" binding:"required"`
}

type updateItemInput struct {
	Text      string `json:"text" binding:"required"`
	Completed bool   `json:"completed"`
}

func (h *TaskListHandler) Create(c *gin.Context) {
	var input listInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	list, err := h.listService.CreateList(getUserID(c), input.toVehicleInput())
	if err != nil {
		if errors.Is(err, services.ErrNotManager) {
			utils.RespondError(c, http.StatusForbidden, err.Error())
			return
		}
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, list)
}

func (h *TaskListHandler) List(c *gin.Context) {
	search := c.Query("search")
	status := c.Query("status")
	mine := c.Query("mine") == "true"
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := h.listService.ListAll(getUserID(c), search, page, limit, status, mine)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "erro ao buscar listas")
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *TaskListHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	list, err := h.listService.GetList(uint(id), getUserID(c))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *TaskListHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var input listInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	list, err := h.listService.UpdateList(uint(id), getUserID(c), input.toVehicleInput())
	if err != nil {
		respondVehicleMutationError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *TaskListHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	if err := h.listService.DeleteList(uint(id), getUserID(c)); err != nil {
		if errors.Is(err, services.ErrNotManager) {
			utils.RespondError(c, http.StatusForbidden, err.Error())
			return
		}
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *TaskListHandler) AddItem(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var input createItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	item, err := h.listService.AddItem(uint(listID), getUserID(c), input.Text)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *TaskListHandler) UpdateItem(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}
	itemID, err := strconv.ParseUint(c.Param("itemId"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id de item inválido")
		return
	}

	var input updateItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	item, err := h.listService.UpdateItem(uint(listID), uint(itemID), getUserID(c), input.Text, input.Completed)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *TaskListHandler) DeleteItem(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}
	itemID, err := strconv.ParseUint(c.Param("itemId"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id de item inválido")
		return
	}

	if err := h.listService.DeleteItem(uint(listID), uint(itemID), getUserID(c)); err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

type reorderInput struct {
	IDs []uint `json:"ids" binding:"required"`
}

func (h *TaskListHandler) ReorderLists(c *gin.Context) {
	var input reorderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("ReorderLists bind error: %v", err)
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	if err := h.listService.ReorderLists(getUserID(c), input.IDs); err != nil {
		log.Printf("ReorderLists error: %v", err)
		if errors.Is(err, services.ErrNotManager) {
			utils.RespondError(c, http.StatusForbidden, err.Error())
			return
		}
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TaskListHandler) ReorderItems(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var input reorderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	if err := h.listService.ReorderItems(uint(listID), getUserID(c), input.IDs); err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

type changeStatusInput struct {
	Status models.TaskListStatus `json:"status" binding:"required"`
}

func (h *TaskListHandler) ChangeStatus(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var input changeStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	if err := h.listService.ChangeStatus(uint(listID), getUserID(c), input.Status); err != nil {
		utils.RespondError(c, http.StatusForbidden, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// respondVehicleMutationError: 403 mutação de gerente, 400 validação, 404 o resto (inclui sem acesso).
func respondVehicleMutationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotManager):
		utils.RespondError(c, http.StatusForbidden, err.Error())
	case isVehicleValidationError(err):
		utils.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		utils.RespondError(c, http.StatusNotFound, err.Error())
	}
}

type changePriorityInput struct {
	Priority string `json:"priority" binding:"required"`
}

func (h *TaskListHandler) ChangePriority(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var input changePriorityInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	list, err := h.listService.ChangePriority(uint(listID), getUserID(c), input.Priority)
	if err != nil {
		respondVehicleMutationError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}
