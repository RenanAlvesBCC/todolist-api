package repository

import (
	"database/sql"
	"strings"

	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

type TaskListRepository struct {
	db *gorm.DB
}

func NewTaskListRepository(db *gorm.DB) *TaskListRepository {
	return &TaskListRepository{db: db}
}

type TaskListFilter struct {
	Search       string
	Page         int
	Limit        int
	Status       string
	AssignedToID *uint
}

// Nome de quem marcou o vídeo de check-up, no mesmo formato de author_name das notas.
const checkupVideoDoneByNameSelect = "(SELECT COALESCE(NULLIF(TRIM(users.first_name || ' ' || users.last_name), ''), users.username)" +
	" FROM users WHERE users.id = task_lists.checkup_video_done_by) AS checkup_video_done_by_name"

// vehicleDetails carrega o veículo completo: serviços, atribuições, notas ativas e campos calculados.
func vehicleDetails(db *gorm.DB) *gorm.DB {
	return db.
		Select("task_lists.*, "+checkupVideoDoneByNameSelect).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Preload("Assignments").
		Preload("Notes", notesWithAuthorNewestFirst)
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
		Scopes(vehicleDetails).
		Order("position asc").
		Order("id asc").
		Offset(offset).Limit(filter.Limit).
		Find(&lists).Error; err != nil {
		return nil, 0, err
	}

	return lists, total, nil
}

func (r *TaskListRepository) buildQuery(userID uint, workspaceID *uint, filter TaskListFilter) *gorm.DB {
	var query *gorm.DB

	// Esteira (AssignedToID): veículo atribuído no workspace atual, ou legado sem workspace_id.
	if filter.AssignedToID != nil {
		sub := r.db.Model(&models.ListAssignment{}).
			Select("task_list_id").
			Where("user_id = ?", *filter.AssignedToID)
		if workspaceID != nil {
			query = r.db.Model(&models.TaskList{}).Where(
				"id IN (?) AND (workspace_id = ? OR workspace_id IS NULL)",
				sub, *workspaceID,
			)
		} else {
			query = r.db.Model(&models.TaskList{}).Where("id IN (?)", sub)
		}
	} else {
		query = yardQuery(r.db, userID, workspaceID)
	}

	if filter.Search != "" {
		// LIKE no Postgres diferencia maiúsculas (no SQLite não); comparamos os dois lados em minúsculas.
		like := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where(
			"(LOWER(title) LIKE ? OR LOWER(plate) LIKE ? OR LOWER(customer) LIKE ?)",
			like, like, like,
		)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	return query
}

// FindByIDAndUser busca lista pelo id verificando posse direta (para listas pessoais).
func (r *TaskListRepository) FindByIDAndUser(id, userID uint) (*models.TaskList, error) {
	var list models.TaskList
	if err := r.db.
		Scopes(vehicleDetails).
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
		Scopes(vehicleDetails).
		First(&list, id).Error; err != nil {
		return nil, err
	}
	return &list, nil
}

// Update não regrava Notes: informações importantes têm ciclo próprio (inclusive exclusão lógica).
func (r *TaskListRepository) Update(list *models.TaskList) error {
	return r.db.Omit("Notes").Save(list).Error
}

func (r *TaskListRepository) Delete(list *models.TaskList) error {
	return r.db.Delete(list).Error
}

// yardQuery é o pátio do usuário: todos os veículos da oficina e os pessoais legados
// (sem workspace_id) que ele criou. Sem oficina, só os pessoais.
func yardQuery(db *gorm.DB, userID uint, workspaceID *uint) *gorm.DB {
	if workspaceID != nil {
		return db.Model(&models.TaskList{}).Where(
			"(workspace_id = ?) OR (user_id = ? AND workspace_id IS NULL)",
			*workspaceID, userID,
		)
	}
	return db.Model(&models.TaskList{}).Where("user_id = ? AND workspace_id IS NULL", userID)
}

// NextPosition devolve a posição logo após o último veículo do pátio, independente de quem o criou.
func (r *TaskListRepository) NextPosition(userID uint, workspaceID *uint) (int, error) {
	var maxPosition sql.NullInt64
	if err := yardQuery(r.db, userID, workspaceID).
		Select("MAX(position)").
		Row().Scan(&maxPosition); err != nil {
		return 0, err
	}
	if !maxPosition.Valid {
		return 0, nil
	}
	return int(maxPosition.Int64) + 1, nil
}

// UpdatePositions aceita o pátio inteiro ou um subconjunto (ex.: só os não entregues).
// Os ids pedidos ocupam, na nova ordem, as mesmas vagas que já ocupavam na ordem atual;
// os demais veículos não mudam de lugar. Ao final o pátio fica numerado 0..n-1, sem empates.
// Ids fora do pátio do usuário são ignorados.
func (r *TaskListRepository) UpdatePositions(userID uint, workspaceID *uint, orderedIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var current []models.TaskList
		if err := yardQuery(tx, userID, workspaceID).
			Select("id", "position").
			Order("position asc").
			Order("id asc").
			Find(&current).Error; err != nil {
			return err
		}

		currentIDs := make([]uint, len(current))
		oldPosition := make(map[uint]int, len(current))
		for i, l := range current {
			currentIDs[i] = l.ID
			oldPosition[l.ID] = l.Position
		}
		requested := make(map[uint]bool, len(orderedIDs))
		for _, id := range orderedIDs {
			requested[id] = true
		}

		for index, id := range mergeOrder(currentIDs, orderedIDs) {
			if oldPosition[id] == index {
				continue
			}
			update := tx.Model(&models.TaskList{}).Where("id = ?", id)
			var err error
			if requested[id] {
				err = update.Update("position", index).Error
			} else {
				// Só renumeração de dados antigos (empates/buracos): não conta como edição do veículo.
				err = update.UpdateColumn("position", index).Error
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// mergeOrder coloca os ids pedidos (conhecidos, sem repetição) nas vagas que eles já
// ocupavam em current, na ordem pedida; os outros ids continuam onde estavam.
func mergeOrder(current, requested []uint) []uint {
	known := make(map[uint]bool, len(current))
	for _, id := range current {
		known[id] = true
	}
	picked := make([]uint, 0, len(requested))
	seen := make(map[uint]bool, len(requested))
	for _, id := range requested {
		if known[id] && !seen[id] {
			seen[id] = true
			picked = append(picked, id)
		}
	}

	result := append([]uint(nil), current...)
	next := 0
	for i, id := range result {
		if seen[id] {
			result[i] = picked[next]
			next++
		}
	}
	return result
}
