package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/services"
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

// getUserID lê o user_id que o middleware guardou no contexto (fase 4).
// Atenção: claims de JWT decodificam números como float64, não como uint —
// por isso o type assertion é pra float64 antes de converter pra uint.
func getUserID(c *gin.Context) uint {
	userIDFloat := c.MustGet("user_id").(float64)
	return uint(userIDFloat)
}

func (h *TaskHandler) Create(c *gin.Context) {
	var input taskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	task, err := h.taskService.Create(getUserID(c), input.Title, input.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *TaskHandler) List(c *gin.Context) {
	tasks, err := h.taskService.List(getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao buscar tarefas"})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *TaskHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	task, err := h.taskService.Get(uint(id), getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var input taskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	task, err := h.taskService.Update(uint(id), getUserID(c), input.Title, input.Description, input.Completed)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	if err := h.taskService.Delete(uint(id), getUserID(c)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
