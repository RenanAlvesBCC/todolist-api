package repository

import "time"

type AuditFilter struct {
	Category string
	ActorID  *uint
	DateFrom *time.Time
	DateTo   *time.Time
	Page     int
	Limit    int
}

type AuditLogRow struct {
	ID          uint      `json:"id"`
	ActorID     uint      `json:"actor_id"`
	ActorName   string    `json:"actor_name"`
	Category    string    `json:"category"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r *SecurityRepository) ListLogs(filter AuditFilter) ([]AuditLogRow, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}

	query := r.db.Table("audit_logs").
		Select("audit_logs.id, COALESCE(audit_logs.user_id, 0) as actor_id, COALESCE(users.username, '') as actor_name, audit_logs.action, audit_logs.details as description, audit_logs.created_at").
		Joins("LEFT JOIN users ON users.id = audit_logs.user_id")

	if filter.ActorID != nil {
		query = query.Where("audit_logs.user_id = ?", *filter.ActorID)
	}
	if filter.DateFrom != nil {
		query = query.Where("audit_logs.created_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("audit_logs.created_at <= ?", *filter.DateTo)
	}

	var rows []AuditLogRow
	if err := query.Order("audit_logs.created_at desc").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	for i := range rows {
		rows[i].Category = categoryOf(rows[i].Action)
	}

	if filter.Category != "" {
		filtered := make([]AuditLogRow, 0, len(rows))
		for _, row := range rows {
			if row.Category == filter.Category {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}

	total := int64(len(rows))
	start := (filter.Page - 1) * filter.Limit
	if start >= len(rows) {
		return []AuditLogRow{}, total, nil
	}
	end := start + filter.Limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[start:end], total, nil
}

func categoryOf(action string) string {
	switch action {
	case "login_success", "login_failed", "logout", "register_success", "register_failed":
		return "acesso"
	default:
		return "veiculo"
	}
}
