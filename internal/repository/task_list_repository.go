package repository

import (
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type TaskListRepository struct {
	db *gorm.DB
}

func NewTaskListRepository(db *gorm.DB) *TaskListRepository {
	return &TaskListRepository{db: db}
}

type TaskListFilter struct {
	Search string
	Page   int
	Limit  int
}

func (r *TaskListRepository) Create(list *models.TaskList) error {
	return r.db.Create(list).Error
}

// FindAll retorna listas pessoais do usuário e, quando workspaceID não é nil,
// também todas as listas vinculadas àquele workspace.
func (r *TaskListRepository) FindAll(userID uint, workspaceID *uint, filter TaskListFilter) ([]models.TaskList, int64, error) {
	var lists []models.TaskList
	var total int64

	base := r.buildQuery(userID, workspaceID, filter)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	if err := r.buildQuery(userID, workspaceID, filter).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Order("position asc").
		Offset(offset).Limit(filter.Limit).
		Find(&lists).Error; err != nil {
		return nil, 0, err
	}

	return lists, total, nil
}

func (r *TaskListRepository) buildQuery(userID uint, workspaceID *uint, filter TaskListFilter) *gorm.DB {
	var query *gorm.DB
	if workspaceID != nil {
		query = r.db.Model(&models.TaskList{}).Where(
			"(workspace_id = ?) OR (user_id = ? AND workspace_id IS NULL)",
			*workspaceID, userID,
		)
	} else {
		query = r.db.Model(&models.TaskList{}).Where("user_id = ? AND workspace_id IS NULL", userID)
	}
	if filter.Search != "" {
		query = query.Where("title LIKE ?", "%"+filter.Search+"%")
	}
	return query
}

// FindByIDAndUser busca lista pelo id verificando posse direta (para listas pessoais).
func (r *TaskListRepository) FindByIDAndUser(id, userID uint) (*models.TaskList, error) {
	var list models.TaskList
	if err := r.db.
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Where("id = ? AND user_id = ?", id, userID).
		First(&list).Error; err != nil {
		return nil, err
	}
	return &list, nil
}

// FindByID busca lista pelo id sem restrição de usuário (usado para contexto de workspace).
func (r *TaskListRepository) FindByID(id uint) (*models.TaskList, error) {
	var list models.TaskList
	if err := r.db.
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		First(&list, id).Error; err != nil {
		return nil, err
	}
	return &list, nil
}

func (r *TaskListRepository) Update(list *models.TaskList) error {
	return r.db.Save(list).Error
}

func (r *TaskListRepository) Delete(list *models.TaskList) error {
	return r.db.Delete(list).Error
}

func (r *TaskListRepository) NextPosition(userID uint) (int, error) {
	var count int64
	if err := r.db.Model(&models.TaskList{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *TaskListRepository) UpdatePositions(userID uint, orderedIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for index, id := range orderedIDs {
			result := tx.Model(&models.TaskList{}).
				Where("id = ? AND user_id = ?", id, userID).
				Update("position", index)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}
