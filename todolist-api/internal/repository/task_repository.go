package repository

import (
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// TaskFilter agrupa os critérios opcionais de busca e paginação.
type TaskFilter struct {
	Completed *bool
	Search    string
	Page      int
	Limit     int
}

// applyFilters monta a query com os filtros aplicados, mas sem executar ainda.
// Criamos uma query nova a cada chamada (em vez de reaproveitar a mesma variável
// entre Count e Find) porque o GORM pode manter cláusulas de uma chamada anterior
// — como Limit — se você reusar a mesma cadeia pra duas operações finais diferentes.
func (r *TaskRepository) applyFilters(userID uint, filter TaskFilter) *gorm.DB {
	query := r.db.Model(&models.Task{}).Where("user_id = ?", userID)

	if filter.Completed != nil {
		query = query.Where("completed = ?", *filter.Completed)
	}

	if filter.Search != "" {
		query = query.Where("title LIKE ?", "%"+filter.Search+"%")
	}

	return query
}

func (r *TaskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

// FindAllByUser devolve as tarefas já filtradas e paginadas, além do total
// de registros que combinam com o filtro (sem aplicar a paginação) —
// é esse total que o cliente usa pra saber quantas páginas existem.
func (r *TaskRepository) FindAllByUser(userID uint, filter TaskFilter) ([]models.Task, int64, error) {
	var tasks []models.Task
	var total int64

	if err := r.applyFilters(userID, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	if err := r.applyFilters(userID, filter).Offset(offset).Limit(filter.Limit).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func (r *TaskRepository) FindByIDAndUser(id uint, userID uint) (*models.Task, error) {
	var task models.Task
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

func (r *TaskRepository) Delete(task *models.Task) error {
	return r.db.Delete(task).Error
}
