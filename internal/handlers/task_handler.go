package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/services"
	"github.com/RenanAlvesBCC/todolist-api/internal/utils"
)

type TaskHandler struct {
	taskService *services.TaskService
}

func NewTaskHandler(taskService *services.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

type taskInput struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func getUserID(c *gin.Context) uint {
	userIDFloat := c.MustGet("user_id").(float64)
	return uint(userIDFloat)
}

func (h *TaskHandler) Create(c *gin.Context) {
	var input taskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	task, err := h.taskService.Create(getUserID(c), input.Title, input.Description)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, task)
}

// List lê filtros e paginação da query string, por exemplo:
// /api/tasks?completed=false&search=estudar&page=1&limit=10
func (h *TaskHandler) List(c *gin.Context) {
	var completed *bool
	if value := c.Query("completed"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "completed deve ser 'true' ou 'false'")
			return
		}
		completed = &parsed
	}

	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := h.taskService.List(getUserID(c), completed, search, page, limit)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "erro ao buscar tarefas")
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *TaskHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	task, err := h.taskService.Get(uint(id), getUserID(c))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var input taskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	task, err := h.taskService.Update(uint(id), getUserID(c), input.Title, input.Description, input.Completed)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	if err := h.taskService.Delete(uint(id), getUserID(c)); err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
