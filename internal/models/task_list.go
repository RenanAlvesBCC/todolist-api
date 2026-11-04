package models

import (
	"encoding/json"
	"time"
)

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

type VehiclePriority string

const (
	PriorityNormal  VehiclePriority = "normal"
	PriorityAlta    VehiclePriority = "alta"
	PriorityUrgente VehiclePriority = "urgente"
)

func IsValidVehiclePriority(p VehiclePriority) bool {
	switch p {
	case PriorityNormal, PriorityAlta, PriorityUrgente:
		return true
	}
	return false
}

type TaskList struct {
	Base
	Title    string `gorm:"not null" json:"title"`
	Plate    string `gorm:"not null;default:''" json:"plate"`
	Customer string `gorm:"not null;default:''" json:"customer"`
	// Dados do cliente (LGPD), só dígitos. Fora do JSON por padrão: ver MarshalJSON.
	CustomerCPF      string           `gorm:"not null;default:''" json:"-"`
	CustomerCEP      string           `gorm:"not null;default:''" json:"-"`
	CustomerWhatsApp string           `gorm:"column:customer_whatsapp;not null;default:''" json:"-"`
	UserID           uint             `gorm:"not null" json:"user_id"`
	WorkspaceID      *uint            `json:"workspace_id"`
	Status           TaskListStatus   `gorm:"not null;default:'aguardando_box'" json:"status"`
	Priority         VehiclePriority  `gorm:"not null;default:'normal'" json:"priority"`
	Position         int              `gorm:"not null;default:0" json:"position"`
	DeliveredAt      *time.Time       `gorm:"index" json:"delivered_at"`
	Items            []TaskItem       `gorm:"foreignKey:TaskListID" json:"items"`
	Assignments      []ListAssignment `gorm:"foreignKey:TaskListID" json:"assignments,omitempty"`
	Notes            []VehicleNote    `gorm:"foreignKey:TaskListID" json:"notes"`

	// CustomerDataVisible é decidido pelo service conforme o papel de quem pede
	// (dono/gerente); com false os dados do cliente nem aparecem como chave no JSON.
	CustomerDataVisible bool `gorm:"-" json:"-"`
}

type taskListJSON TaskList

func (l TaskList) MarshalJSON() ([]byte, error) {
	out := struct {
		taskListJSON
		CustomerCPF      *string `json:"customer_cpf,omitempty"`
		CustomerCEP      *string `json:"customer_cep,omitempty"`
		CustomerWhatsApp *string `json:"customer_whatsapp,omitempty"`
	}{taskListJSON: taskListJSON(l)}
	if l.CustomerDataVisible {
		out.CustomerCPF = &l.CustomerCPF
		out.CustomerCEP = &l.CustomerCEP
		out.CustomerWhatsApp = &l.CustomerWhatsApp
	}
	return json.Marshal(out)
}
