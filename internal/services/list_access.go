package services

import (
	"errors"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

var ErrNotManager = errors.New("apenas gerentes podem gerenciar veículos")
var ErrNotFound = errors.New("lista não encontrada")

func isAssigned(list *models.TaskList, userID uint) bool {
	for _, a := range list.Assignments {
		if a.UserID == userID {
			return true
		}
	}
	return false
}

// checkVehicleAccess: dono da lista ou membro do workspace.
// editor só acessa veículos em que está atribuído.
func checkVehicleAccess(list *models.TaskList, userID uint, wsStore WorkspaceContextStore) error {
	if list.UserID == userID {
		return nil
	}
	if list.WorkspaceID != nil && wsStore != nil {
		ok, _ := wsStore.IsMember(*list.WorkspaceID, userID)
		if !ok {
			return ErrNotFound
		}
		role, err := wsStore.GetMemberRole(*list.WorkspaceID, userID)
		if err != nil {
			return ErrNotFound
		}
		if role == models.RoleEditor && !isAssigned(list, userID) {
			return ErrNotFound
		}
		return nil
	}
	return ErrNotFound
}
