package models

type TaskListStatus string

const (
	StatusAguardandoBox       TaskListStatus = "aguardando_box"
	StatusEmDiagnostico       TaskListStatus = "em_diagnostico"
	StatusAguardandoOrcamento TaskListStatus = "aguardando_orcamento"
	StatusEmAndamento         TaskListStatus = "em_andamento"
	StatusAguardandoPeca      TaskListStatus = "aguardando_peca"
	StatusTestDrive           TaskListStatus = "test_drive"
	StatusLavaJato            TaskListStatus = "lava_jato"
	StatusAguardandoCheckup   TaskListStatus = "aguardando_checkup"
	StatusAguardandoRetirada  TaskListStatus = "aguardando_retirada"
	StatusEntregue            TaskListStatus = "entregue"
)

// AllTaskListStatuses é a ordem canônica da esteira (fluxo linear sugerido).
var AllTaskListStatuses = []TaskListStatus{
	StatusAguardandoBox,
	StatusEmDiagnostico,
	StatusAguardandoOrcamento,
	StatusEmAndamento,
	StatusAguardandoPeca,
	StatusTestDrive,
	StatusLavaJato,
	StatusAguardandoCheckup,
	StatusAguardandoRetirada,
	StatusEntregue,
}

func IsValidTaskListStatus(s TaskListStatus) bool {
	for _, v := range AllTaskListStatuses {
		if v == s {
			return true
		}
	}
	return false
}

type TaskList struct {
	Base
	Title       string           `gorm:"not null" json:"title"`
	Plate       string           `gorm:"not null;default:''" json:"plate"`
	Customer    string           `gorm:"not null;default:''" json:"customer"`
	UserID      uint             `gorm:"not null" json:"user_id"`
	WorkspaceID *uint            `json:"workspace_id"`
	Status      TaskListStatus   `gorm:"not null;default:'aguardando_box'" json:"status"`
	Position    int              `gorm:"not null;default:0" json:"position"`
	Items       []TaskItem       `gorm:"foreignKey:TaskListID" json:"items"`
	Assignments []ListAssignment `gorm:"foreignKey:TaskListID" json:"assignments,omitempty"`
}
