package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

func newList(db interface{ Create(interface{}) interface{ Error() error } }, userID uint) *models.TaskList {
	// helper abaixo usa listRepo diretamente
	return nil
}

func TestListAssignmentRepository_AssignAndList(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	repo := NewListAssignmentRepository(db)

	list := &models.TaskList{Title: "Fusca", UserID: 1, Status: models.StatusEmAndamento}
	require.NoError(t, listRepo.Create(list))

	a := &models.ListAssignment{
		TaskListID: list.ID,
		UserID:     5,
		AssignedBy: 1,
		AssignedAt: time.Now(),
	}
	require.NoError(t, repo.Assign(a))

	assignments, err := repo.ListByTaskList(list.ID)
	require.NoError(t, err)
	assert.Len(t, assignments, 1)
	assert.Equal(t, uint(5), assignments[0].UserID)
}

func TestListAssignmentRepository_IsAssigned(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	repo := NewListAssignmentRepository(db)

	list := &models.TaskList{Title: "Gol", UserID: 1, Status: models.StatusEmAndamento}
	require.NoError(t, listRepo.Create(list))

	ok, err := repo.IsAssigned(list.ID, 5)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, repo.Assign(&models.ListAssignment{
		TaskListID: list.ID, UserID: 5, AssignedBy: 1, AssignedAt: time.Now(),
	}))

	ok, err = repo.IsAssigned(list.ID, 5)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestListAssignmentRepository_Unassign(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	repo := NewListAssignmentRepository(db)

	list := &models.TaskList{Title: "Uno", UserID: 1, Status: models.StatusEmAndamento}
	require.NoError(t, listRepo.Create(list))

	require.NoError(t, repo.Assign(&models.ListAssignment{
		TaskListID: list.ID, UserID: 5, AssignedBy: 1, AssignedAt: time.Now(),
	}))
	require.NoError(t, repo.Assign(&models.ListAssignment{
		TaskListID: list.ID, UserID: 6, AssignedBy: 1, AssignedAt: time.Now(),
	}))

	require.NoError(t, repo.Unassign(list.ID, 5))

	all, _ := repo.ListByTaskList(list.ID)
	assert.Len(t, all, 1)
	assert.Equal(t, uint(6), all[0].UserID)
}

func TestListAssignmentRepository_UnassignAll(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	repo := NewListAssignmentRepository(db)

	list := &models.TaskList{Title: "Celta", UserID: 1, Status: models.StatusEmAndamento}
	require.NoError(t, listRepo.Create(list))

	for _, uid := range []uint{5, 6, 7} {
		require.NoError(t, repo.Assign(&models.ListAssignment{
			TaskListID: list.ID, UserID: uid, AssignedBy: 1, AssignedAt: time.Now(),
		}))
	}

	require.NoError(t, repo.UnassignAll(list.ID))

	all, _ := repo.ListByTaskList(list.ID)
	assert.Empty(t, all)
}
