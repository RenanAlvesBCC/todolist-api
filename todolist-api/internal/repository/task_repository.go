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

func (r *TaskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

// FindAllByUser busca só as tarefas que pertencem ao usuário informado.
func (r *TaskRepository) FindAllByUser(userID uint) ([]models.Task, error) {
	var tasks []models.Task
	if err := r.db.Where("user_id = ?", userID).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// FindByIDAndUser busca uma tarefa específica, mas só se ela pertencer ao usuário —
// é essa combinação (id E user_id) que impede alguém de acessar tarefa de outra pessoa.
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
