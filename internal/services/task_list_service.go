package services

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

// TaskListStore descreve o que o TaskListService precisa do repository de listas.
type TaskListStore interface {
	Create(list *models.TaskList) error
	FindAll(userID uint, workspaceID *uint, filter repository.TaskListFilter) ([]models.TaskList, int64, error)
	FindByIDAndUser(id, userID uint) (*models.TaskList, error)
	FindByID(id uint) (*models.TaskList, error)
	Update(list *models.TaskList) error
	Delete(list *models.TaskList) error
	NextPosition(userID uint, workspaceID *uint) (int, error)
	UpdatePositions(userID uint, workspaceID *uint, orderedIDs []uint) error
}

type TaskItemStore interface {
	Create(item *models.TaskItem) error
	FindByIDAndList(id, taskListID uint) (*models.TaskItem, error)
	Update(item *models.TaskItem) error
	Delete(item *models.TaskItem) error
	DeleteAllByList(taskListID uint) error
	NextPosition(taskListID uint) (int, error)
	UpdatePositions(taskListID uint, orderedIDs []uint) error
}

// WorkspaceContextStore é o subconjunto da interface de workspace que o TaskListService precisa.
type WorkspaceContextStore interface {
	FindByMemberUserID(userID uint) (*models.Workspace, error)
	GetMemberRole(workspaceID, userID uint) (models.WorkspaceRole, error)
	IsMember(workspaceID, userID uint) (bool, error)
}

type TaskListService struct {
	listRepo TaskListStore
	itemRepo TaskItemStore
	wsStore  WorkspaceContextStore
	links    CustomerLinkStore
}

func NewTaskListService(listRepo TaskListStore, itemRepo TaskItemStore, wsStore WorkspaceContextStore) *TaskListService {
	return &TaskListService{listRepo: listRepo, itemRepo: itemRepo, wsStore: wsStore}
}

func (s *TaskListService) requireManager(userID uint) error {
	if s.wsStore == nil {
		return nil
	}
	ws, err := s.wsStore.FindByMemberUserID(userID)
	if err != nil {
		return nil
	}
	role, err := s.wsStore.GetMemberRole(ws.ID, userID)
	if err != nil {
		return ErrNotFound
	}
	if role == models.RoleEditor {
		return ErrNotManager
	}
	return nil
}

// workspaceID é a oficina do usuário, ou nil se ele não participa de nenhuma.
func (s *TaskListService) workspaceID(userID uint) *uint {
	if s.wsStore == nil {
		return nil
	}
	ws, err := s.wsStore.FindByMemberUserID(userID)
	if err != nil {
		return nil
	}
	id := ws.ID
	return &id
}

func normalizePlate(plate string) string {
	return strings.ToUpper(strings.TrimSpace(plate))
}

func normalizeCustomer(customer string) string {
	return strings.TrimSpace(customer)
}

var (
	ErrTitleRequired   = errors.New("título é obrigatório")
	ErrInvalidPriority = errors.New("prioridade inválida")
	ErrInvalidMileage  = errors.New("quilometragem inválida")
)

// MaxMileage é a maior quilometragem aceita (km).
const MaxMileage = 2000000

// VehicleInput é a ficha do veículo enviada no create/update.
// Nos campos ponteiro, nil = chave ausente (no update mantém o valor atual).
type VehicleInput struct {
	Title            string
	Plate            string
	Customer         string
	CustomerCPF      *string
	CustomerCEP      *string
	CustomerWhatsApp *string
	Priority         *string
	Mileage          *int
	// Vínculo com o cadastro de clientes (0 desvincula).
	CustomerID        *uint
	CustomerVehicleID *uint
}

func parsePriority(value string) (models.VehiclePriority, error) {
	p := models.VehiclePriority(value)
	if !models.IsValidVehiclePriority(p) {
		return "", ErrInvalidPriority
	}
	return p, nil
}

// applyOptionalFields valida tudo antes de alterar a ficha, para não deixar o veículo meio atualizado.
func applyOptionalFields(list *models.TaskList, in VehicleInput) error {
	priority := list.Priority
	if in.Priority != nil {
		p, err := parsePriority(*in.Priority)
		if err != nil {
			return err
		}
		priority = p
	}
	if in.Mileage != nil && (*in.Mileage < 0 || *in.Mileage > MaxMileage) {
		return ErrInvalidMileage
	}
	if err := applyCustomerData(list, in); err != nil {
		return err
	}
	list.Priority = priority
	if in.Mileage != nil {
		km := *in.Mileage
		list.Mileage = &km
	}
	return nil
}

func applyCustomerData(list *models.TaskList, in VehicleInput) error {
	type field struct {
		value     *string
		normalize func(string) (string, error)
		target    *string
	}
	fields := []field{
		{in.CustomerCPF, normalizeCPF, &list.CustomerCPF},
		{in.CustomerCEP, normalizeCEP, &list.CustomerCEP},
		{in.CustomerWhatsApp, normalizeWhatsApp, &list.CustomerWhatsApp},
	}
	normalized := make([]string, len(fields))
	for i, f := range fields {
		if f.value == nil {
			continue
		}
		v, err := f.normalize(*f.value)
		if err != nil {
			return err
		}
		normalized[i] = v
	}
	for i, f := range fields {
		if f.value != nil {
			*f.target = normalized[i]
		}
	}
	return nil
}

// viewer é quem está pedindo: oficina e papel nela ("" sem oficina).
type viewer struct {
	userID uint
	wsID   *uint
	role   models.WorkspaceRole
}

func (s *TaskListService) viewerOf(userID uint) viewer {
	v := viewer{userID: userID, wsID: s.workspaceID(userID)}
	if v.wsID != nil {
		v.role, _ = s.wsStore.GetMemberRole(*v.wsID, userID)
	}
	return v
}

// canSeeCustomerData: CPF/CEP/WhatsApp só para dono/gerente da oficina do veículo
// (ou quem criou um veículo pessoal legado). Mecânico nunca recebe.
func (v viewer) canSeeCustomerData(list *models.TaskList) bool {
	if v.role == models.RoleEditor {
		return false
	}
	if list.WorkspaceID == nil {
		return list.UserID == v.userID
	}
	return v.wsID != nil && *list.WorkspaceID == *v.wsID &&
		(v.role == models.RoleOwner || v.role == models.RoleManager)
}

func (s *TaskListService) revealCustomerData(list *models.TaskList, userID uint) *models.TaskList {
	list.CustomerDataVisible = s.viewerOf(userID).canSeeCustomerData(list)
	return list
}

// titleMissing: sem título só vale quando o veículo do cliente pode preenchê-lo.
func titleMissing(in VehicleInput) bool {
	return in.Title == "" && (in.CustomerVehicleID == nil || *in.CustomerVehicleID == 0)
}

func (s *TaskListService) CreateList(userID uint, in VehicleInput) (*models.TaskList, error) {
	if titleMissing(in) {
		return nil, ErrTitleRequired
	}
	if err := s.requireManager(userID); err != nil {
		return nil, err
	}

	list := &models.TaskList{
		Title:       in.Title,
		Plate:       normalizePlate(in.Plate),
		Customer:    normalizeCustomer(in.Customer),
		UserID:      userID,
		WorkspaceID: s.workspaceID(userID),
		Status:      models.StatusAguardandoBox,
		Priority:    models.PriorityNormal,
		Items:       []models.TaskItem{},
		Notes:       []models.VehicleNote{},
	}
	if err := applyOptionalFields(list, in); err != nil {
		return nil, err
	}
	if err := s.applyCustomerLinks(list, in); err != nil {
		return nil, err
	}
	if list.Title == "" {
		return nil, ErrTitleRequired
	}

	position, err := s.listRepo.NextPosition(userID, list.WorkspaceID)
	if err != nil {
		return nil, err
	}
	list.Position = position

	if err := s.listRepo.Create(list); err != nil {
		return nil, err
	}
	return s.revealCustomerData(list, userID), nil
}

type PaginatedTaskLists struct {
	Lists      []models.TaskList `json:"lists"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	Total      int64             `json:"total"`
	TotalPages int               `json:"total_pages"`
}

func (s *TaskListService) ListAll(userID uint, search string, page, limit int, status string, mine bool) (*PaginatedTaskLists, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var wsID *uint
	var assignedTo *uint
	if s.wsStore != nil {
		if ws, err := s.wsStore.FindByMemberUserID(userID); err == nil {
			wsID = &ws.ID
			role, _ := s.wsStore.GetMemberRole(ws.ID, userID)
			if role == models.RoleEditor || mine {
				id := userID
				assignedTo = &id
			}
		} else if mine {
			id := userID
			assignedTo = &id
		}
	} else if mine {
		id := userID
		assignedTo = &id
	}

	filter := repository.TaskListFilter{
		Search:       search,
		Page:         page,
		Limit:        limit,
		Status:       status,
		AssignedToID: assignedTo,
	}
	lists, total, err := s.listRepo.FindAll(userID, wsID, filter)
	if err != nil {
		return nil, err
	}
	v := s.viewerOf(userID)
	for i := range lists {
		lists[i].CustomerDataVisible = v.canSeeCustomerData(&lists[i])
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &PaginatedTaskLists{
		Lists:      lists,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *TaskListService) resolveList(listID, userID uint) (*models.TaskList, error) {
	list, err := s.listRepo.FindByID(listID)
	if err != nil {
		return nil, ErrNotFound
	}
	if err := checkVehicleAccess(list, userID, s.wsStore); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *TaskListService) GetList(listID, userID uint) (*models.TaskList, error) {
	list, err := s.resolveList(listID, userID)
	if err != nil {
		return nil, err
	}
	return s.revealCustomerData(list, userID), nil
}

// UpdateList substitui title, plate e customer; os demais campos só mudam se vierem no input.
// Com customer_id / customer_vehicle_id no input, o que ficar vazio é preenchido pelo cadastro.
func (s *TaskListService) UpdateList(listID, userID uint, in VehicleInput) (*models.TaskList, error) {
	if titleMissing(in) {
		return nil, ErrTitleRequired
	}
	if err := s.requireManager(userID); err != nil {
		return nil, err
	}

	list, err := s.resolveList(listID, userID)
	if err != nil {
		return nil, err
	}

	if err := applyOptionalFields(list, in); err != nil {
		return nil, err
	}
	list.Title = in.Title
	list.Plate = normalizePlate(in.Plate)
	list.Customer = normalizeCustomer(in.Customer)
	if err := s.applyCustomerLinks(list, in); err != nil {
		return nil, err
	}
	if list.Title == "" {
		return nil, ErrTitleRequired
	}
	if err := s.listRepo.Update(list); err != nil {
		return nil, err
	}
	return s.revealCustomerData(list, userID), nil
}

// SetCheckupVideo marca/desmarca o vídeo de check-up. Mesma regra de acesso de serviços/notas:
// dono/gerente em qualquer veículo da oficina; mecânico só nos atribuídos (senão 404).
func (s *TaskListService) SetCheckupVideo(listID, userID uint, done bool) (*models.TaskList, error) {
	list, err := s.resolveList(listID, userID)
	if err != nil {
		return nil, err
	}

	if applyCheckupVideo(list, done, userID, time.Now().UTC()) {
		if err := s.listRepo.Update(list); err != nil {
			return nil, err
		}
		// Recarrega para trazer checkup_video_done_by_name, calculado na consulta.
		if list, err = s.listRepo.FindByID(listID); err != nil {
			return nil, err
		}
	}
	return s.revealCustomerData(list, userID), nil
}

// applyCheckupVideo: true grava data/autor só na primeira marcação; false limpa.
// Devolve se algo mudou.
func applyCheckupVideo(list *models.TaskList, done bool, userID uint, now time.Time) bool {
	if done {
		if list.CheckupVideoDoneAt != nil {
			return false
		}
		list.CheckupVideoDoneAt = &now
		list.CheckupVideoDoneBy = &userID
		return true
	}
	if list.CheckupVideoDoneAt == nil && list.CheckupVideoDoneBy == nil {
		return false
	}
	list.CheckupVideoDoneAt = nil
	list.CheckupVideoDoneBy = nil
	list.CheckupVideoDoneByName = nil
	return true
}

// ChangePriority não mexe na ordem do pátio: a ordenação continua por position.
func (s *TaskListService) ChangePriority(listID, userID uint, priority string) (*models.TaskList, error) {
	if err := s.requireManager(userID); err != nil {
		return nil, err
	}
	p, err := parsePriority(priority)
	if err != nil {
		return nil, err
	}

	list, err := s.resolveList(listID, userID)
	if err != nil {
		return nil, err
	}
	list.Priority = p
	if err := s.listRepo.Update(list); err != nil {
		return nil, err
	}
	return s.revealCustomerData(list, userID), nil
}

func (s *TaskListService) DeleteList(listID, userID uint) error {
	if err := s.requireManager(userID); err != nil {
		return err
	}

	list, err := s.resolveList(listID, userID)
	if err != nil {
		return err
	}

	if err := s.itemRepo.DeleteAllByList(list.ID); err != nil {
		return err
	}
	return s.listRepo.Delete(list)
}

func (s *TaskListService) AddItem(listID, userID uint, text string) (*models.TaskItem, error) {
	if text == "" {
		return nil, errors.New("texto do item é obrigatório")
	}

	list, err := s.resolveList(listID, userID)
	if err != nil {
		return nil, err
	}

	position, err := s.itemRepo.NextPosition(list.ID)
	if err != nil {
		return nil, err
	}

	item := &models.TaskItem{Text: text, TaskListID: list.ID, Position: position}
	if err := s.itemRepo.Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *TaskListService) UpdateItem(listID, itemID, userID uint, text string, completed bool) (*models.TaskItem, error) {
	if text == "" {
		return nil, errors.New("texto do item é obrigatório")
	}

	list, err := s.resolveList(listID, userID)
	if err != nil {
		return nil, err
	}

	item, err := s.itemRepo.FindByIDAndList(itemID, list.ID)
	if err != nil {
		return nil, errors.New("item não encontrado")
	}

	item.Text = text
	item.Completed = completed
	if err := s.itemRepo.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *TaskListService) DeleteItem(listID, itemID, userID uint) error {
	list, err := s.resolveList(listID, userID)
	if err != nil {
		return err
	}

	item, err := s.itemRepo.FindByIDAndList(itemID, list.ID)
	if err != nil {
		return errors.New("item não encontrado")
	}

	return s.itemRepo.Delete(item)
}

func (s *TaskListService) ReorderLists(userID uint, orderedIDs []uint) error {
	if err := s.requireManager(userID); err != nil {
		return err
	}
	if len(orderedIDs) == 0 {
		return errors.New("lista de ids vazia")
	}
	return s.listRepo.UpdatePositions(userID, s.workspaceID(userID), orderedIDs)
}

func (s *TaskListService) ReorderItems(listID, userID uint, orderedIDs []uint) error {
	list, err := s.resolveList(listID, userID)
	if err != nil {
		return err
	}
	if len(orderedIDs) == 0 {
		return errors.New("lista de ids vazia")
	}
	return s.itemRepo.UpdatePositions(list.ID, orderedIDs)
}

func (s *TaskListService) ChangeStatus(listID, userID uint, newStatus models.TaskListStatus) error {
	list, err := s.resolveList(listID, userID)
	if err != nil {
		return err
	}

	role, err := s.getUserRole(list, userID)
	if err != nil {
		return err
	}

	if !isValidTransition(role, list.Status, newStatus) {
		return errors.New("transição de status não permitida para seu papel")
	}

	applyStatus(list, newStatus, time.Now().UTC())
	return s.listRepo.Update(list)
}

// applyStatus troca o status e mantém delivered_at coerente: grava o instante
// da primeira entrada em entregue (sem sobrescrever) e limpa ao sair dele.
func applyStatus(list *models.TaskList, newStatus models.TaskListStatus, now time.Time) {
	if newStatus == models.StatusEntregue {
		if list.Status != models.StatusEntregue || list.DeliveredAt == nil {
			list.DeliveredAt = &now
		}
	} else {
		list.DeliveredAt = nil
	}
	list.Status = newStatus
}

func (s *TaskListService) getUserRole(list *models.TaskList, userID uint) (models.WorkspaceRole, error) {
	return vehicleRole(list, userID, s.wsStore)
}

// Editor pode ir para qualquer status válido (fluxo linear é só sugestão de UX).
// Owner/manager também podem setar qualquer status válido.
func isValidTransition(role models.WorkspaceRole, from, to models.TaskListStatus) bool {
	if !models.IsValidTaskListStatus(to) {
		return false
	}
	switch role {
	case models.RoleOwner, models.RoleManager, models.RoleEditor:
		return true
	default:
		return false
	}
}
