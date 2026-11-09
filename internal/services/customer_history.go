package services

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

const (
	HistorySourceOnMotor = "onmotor"
	HistorySourceRMAXX   = "rmaxx"
)

const maxSummaryLength = 200

// HistoryRecord é o formato comum do histórico: OS do OnMotor ou atendimento nosso.
type HistoryRecord struct {
	Source      string          `json:"source"`
	ID          uint            `json:"id"`
	OpenedAt    *time.Time      `json:"opened_at"`
	DeliveredAt *time.Time      `json:"delivered_at"`
	Mileage     *int            `json:"mileage"`
	Status      string          `json:"status"`
	Type        string          `json:"type"`
	Summary     string          `json:"summary"`
	Items       models.JSONText `json:"items"`
	Total       *float64        `json:"total"`
	ListID      *uint           `json:"list_id"`
}

type VehicleHistory struct {
	Records []HistoryRecord `json:"records"`
}

type historyItem struct {
	Description string `json:"description"`
}

// VehicleHistory junta as OS importadas e os nossos atendimentos do veículo, mais recentes primeiro.
func (s *CustomerService) VehicleHistory(userID, vehicleID uint) (*VehicleHistory, error) {
	wsID, err := s.workspace(userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.findVehicle(wsID, vehicleID); err != nil {
		return nil, err
	}
	services, err := s.store.ServiceRecordsByVehicle(wsID, vehicleID)
	if err != nil {
		return nil, err
	}
	lists, err := s.store.TaskListsByCustomerVehicle(wsID, vehicleID)
	if err != nil {
		return nil, err
	}

	records := make([]HistoryRecord, 0, len(services)+len(lists))
	for _, sr := range services {
		records = append(records, fromServiceRecord(sr))
	}
	for _, l := range lists {
		records = append(records, fromTaskList(l))
	}
	sortHistory(records)
	return &VehicleHistory{Records: records}, nil
}

func fromServiceRecord(sr models.ServiceRecord) HistoryRecord {
	items := sr.Items
	if !json.Valid([]byte(items)) {
		items = "[]"
	}
	summary := sr.Notes
	if strings.TrimSpace(summary) == "" {
		var parsed []historyItem
		_ = json.Unmarshal([]byte(items), &parsed)
		var parts []string
		for _, it := range parsed {
			if d := strings.TrimSpace(it.Description); d != "" && len(parts) < 3 {
				parts = append(parts, d)
			}
		}
		summary = strings.Join(parts, "; ")
	}
	return HistoryRecord{
		Source:      HistorySourceOnMotor,
		ID:          sr.ID,
		OpenedAt:    utcPtr(sr.OpenedAt),
		DeliveredAt: utcPtr(sr.DeliveredAt),
		Mileage:     sr.Mileage,
		Status:      sr.Status,
		Type:        sr.Type,
		Summary:     truncateRunes(strings.TrimSpace(summary), maxSummaryLength),
		Items:       items,
		Total:       sr.Total,
	}
}

func fromTaskList(l models.TaskList) HistoryRecord {
	var parts []string
	for _, it := range l.Items {
		if t := strings.TrimSpace(it.Text); t != "" && len(parts) < 3 {
			parts = append(parts, t)
		}
	}
	summary := strings.Join(parts, "; ")
	if summary == "" {
		summary = l.Title
	}
	opened := l.CreatedAt.UTC()
	id := l.ID
	return HistoryRecord{
		Source:      HistorySourceRMAXX,
		ID:          l.ID,
		OpenedAt:    &opened,
		DeliveredAt: utcPtr(l.DeliveredAt),
		Mileage:     l.Mileage,
		Status:      string(l.Status),
		Summary:     truncateRunes(summary, maxSummaryLength),
		Items:       "null",
		ListID:      &id,
	}
}

// sortHistory: abertura mais recente primeiro; sem data vai para o fim.
func sortHistory(records []HistoryRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		a, b := records[i].OpenedAt, records[j].OpenedAt
		switch {
		case a == nil && b == nil:
			return records[i].ID > records[j].ID
		case a == nil:
			return false
		case b == nil:
			return true
		case !a.Equal(*b):
			return a.After(*b)
		}
		return records[i].ID > records[j].ID
	})
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max-1]) + "…"
}
