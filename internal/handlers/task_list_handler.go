package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/services"
	"github.com/RenanAlvesBCC/todolist-api/internal/utils"
)

// TaskListProvider descreve o que o handler precisa do service de listas.
type TaskListProvider interface {
	CreateList(userID uint, title string) (*models.TaskList, error)
	ListAll(userID uint, search string, page, limit int) (*services.PaginatedTaskLists, error)
	GetList(listID, userID uint) (*models.TaskList, error)
	UpdateList(listID, userID uint, title string) (*models.TaskList, error)
	DeleteList(listID, userID uint) error
	AddItem(listID, userID uint, text string) (*models.TaskItem, error)
	UpdateItem(listID, itemID, userID uint, text string, completed bool) (*models.TaskItem, error)
	DeleteItem(listID, itemID, userID uint) error
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

type listInput struct {
	Title string `json:"title" binding:"required"`
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

	list, err := h.listService.CreateList(getUserID(c), input.Title)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, list)
}

func (h *TaskListHandler) List(c *gin.Context) {
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := h.listService.ListAll(getUserID(c), search, page, limit)
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

	list, err := h.listService.UpdateList(uint(id), getUserID(c), input.Title)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
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
