package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func createPositioned(t *testing.T, repo *TaskListRepository, title string, userID uint, wsID *uint, position int) *models.TaskList {
	t.Helper()
	list := &models.TaskList{Title: title, UserID: userID, WorkspaceID: wsID, Position: position}
	require.NoError(t, repo.Create(list))
	return list
}

func yardTitles(t *testing.T, repo *TaskListRepository, userID uint, wsID *uint) []string {
	t.Helper()
	lists, _, err := repo.FindAll(userID, wsID, TaskListFilter{Page: 1, Limit: 100})
	require.NoError(t, err)
	titles := make([]string, 0, len(lists))
	for _, l := range lists {
		titles = append(titles, l.Title)
	}
	return titles
}

func positionOf(t *testing.T, repo *TaskListRepository, id uint) int {
	t.Helper()
	got, err := repo.FindByID(id)
	require.NoError(t, err)
	return got.Position
}

func TestTaskListRepository_UpdatePositions_SubsetKeepsOtherVehiclesInPlace(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	wsID := uint(10)

	a := createPositioned(t, repo, "A", 1, &wsID, 0)
	b := createPositioned(t, repo, "B entregue", 1, &wsID, 1)
	c := createPositioned(t, repo, "C", 1, &wsID, 2)
	d := createPositioned(t, repo, "D entregue", 1, &wsID, 3)
	e := createPositioned(t, repo, "E", 1, &wsID, 4)

	// Pátio mostra só A, C, E (não entregues); arrastar E para o topo.
	require.NoError(t, repo.UpdatePositions(1, &wsID, []uint{e.ID, a.ID, c.ID}))

	assert.Equal(t, []string{"E", "B entregue", "A", "D entregue", "C"}, yardTitles(t, repo, 1, &wsID))
	assert.Equal(t, 1, positionOf(t, repo, b.ID), "veículo fora do subconjunto mantém a posição")
	assert.Equal(t, 3, positionOf(t, repo, d.ID))
}

func TestTaskListRepository_UpdatePositions_ManagerReordersVehiclesCreatedByOthers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	wsID := uint(10)
	owner, manager := uint(1), uint(2)

	a := createPositioned(t, repo, "A", owner, &wsID, 0)
	b := createPositioned(t, repo, "B", manager, &wsID, 1)
	c := createPositioned(t, repo, "C", owner, &wsID, 2)

	require.NoError(t, repo.UpdatePositions(manager, &wsID, []uint{c.ID, b.ID, a.ID}))

	assert.Equal(t, []string{"C", "B", "A"}, yardTitles(t, repo, manager, &wsID))
	assert.Equal(t, []string{"C", "B", "A"}, yardTitles(t, repo, owner, &wsID))
}

func TestTaskListRepository_UpdatePositions_IgnoresForeignUnknownAndDuplicateIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	wsID, otherWS := uint(10), uint(20)

	a := createPositioned(t, repo, "A", 1, &wsID, 0)
	b := createPositioned(t, repo, "B", 1, &wsID, 1)
	foreign := createPositioned(t, repo, "Outra oficina", 9, &otherWS, 7)

	require.NoError(t, repo.UpdatePositions(1, &wsID, []uint{b.ID, foreign.ID, 9999, b.ID, a.ID}))

	assert.Equal(t, []string{"B", "A"}, yardTitles(t, repo, 1, &wsID))
	assert.Equal(t, 7, positionOf(t, repo, foreign.ID), "veículo de outra oficina não é tocado")
}

func TestTaskListRepository_UpdatePositions_NormalizesLegacyTiesAndGaps(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	wsID := uint(10)

	a := createPositioned(t, repo, "A", 1, &wsID, 0)
	b := createPositioned(t, repo, "B", 2, &wsID, 0)
	c := createPositioned(t, repo, "C", 1, &wsID, 5)
	d := createPositioned(t, repo, "D", 2, &wsID, 5)

	require.NoError(t, repo.UpdatePositions(1, &wsID, []uint{d.ID, c.ID}))

	assert.Equal(t, []string{"A", "B", "D", "C"}, yardTitles(t, repo, 1, &wsID))
	assert.Equal(t, []int{0, 1, 2, 3}, []int{
		positionOf(t, repo, a.ID), positionOf(t, repo, b.ID),
		positionOf(t, repo, d.ID), positionOf(t, repo, c.ID),
	})
}

func TestTaskListRepository_UpdatePositions_PersonalScopeWithoutWorkspace(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	wsID := uint(10)

	a := createPositioned(t, repo, "A", 1, nil, 0)
	b := createPositioned(t, repo, "B", 1, nil, 1)
	other := createPositioned(t, repo, "Outro usuário", 2, nil, 0)
	inWorkspace := createPositioned(t, repo, "Na oficina", 1, &wsID, 0)

	require.NoError(t, repo.UpdatePositions(1, nil, []uint{b.ID, a.ID, other.ID, inWorkspace.ID}))

	assert.Equal(t, []string{"B", "A"}, yardTitles(t, repo, 1, nil))
	assert.Equal(t, 0, positionOf(t, repo, other.ID))
	assert.Equal(t, 0, positionOf(t, repo, inWorkspace.ID))
}

func TestTaskListRepository_FindAll_TiesBrokenByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	wsID := uint(10)

	createPositioned(t, repo, "primeiro", 1, &wsID, 0)
	createPositioned(t, repo, "segundo", 2, &wsID, 0)
	createPositioned(t, repo, "topo", 1, &wsID, -1)

	assert.Equal(t, []string{"topo", "primeiro", "segundo"}, yardTitles(t, repo, 1, &wsID))
}

func TestTaskListRepository_NextPosition_IsMaxOfWorkspacePlusOne(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	wsID, otherWS := uint(10), uint(20)

	pos, err := repo.NextPosition(2, &wsID)
	require.NoError(t, err)
	assert.Equal(t, 0, pos, "oficina vazia começa em 0")

	createPositioned(t, repo, "A", 1, &wsID, 0)
	createPositioned(t, repo, "B", 1, &wsID, 4)
	createPositioned(t, repo, "Outra oficina", 3, &otherWS, 50)

	pos, err = repo.NextPosition(2, &wsID)
	require.NoError(t, err)
	assert.Equal(t, 5, pos, "novo veículo vai para o fim do pátio, mesmo criado por outro gerente")
}

func TestMergeOrder(t *testing.T) {
	cases := []struct {
		name      string
		current   []uint
		requested []uint
		want      []uint
	}{
		{"lista completa", []uint{1, 2, 3}, []uint{3, 1, 2}, []uint{3, 1, 2}},
		{"subconjunto ocupa as mesmas vagas", []uint{1, 2, 3, 4, 5}, []uint{5, 1, 3}, []uint{5, 2, 1, 4, 3}},
		{"ids desconhecidos e repetidos", []uint{1, 2}, []uint{2, 9, 2, 1}, []uint{2, 1}},
		{"nenhum id conhecido", []uint{1, 2}, []uint{9}, []uint{1, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, mergeOrder(tc.current, tc.requested))
		})
	}
}
