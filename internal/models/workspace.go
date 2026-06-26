package models

import (
	"time"

	"gorm.io/gorm"
)

type WorkspaceRole string

const (
	RoleOwner   WorkspaceRole = "owner"
	RoleManager WorkspaceRole = "manager"
	RoleEditor  WorkspaceRole = "editor"
)

type Workspace struct {
	gorm.Model
	Name        string            `gorm:"not null" json:"name"`
	Description string            `json:"description"`
	OwnerID     uint              `gorm:"not null" json:"owner_id"`
	Members     []WorkspaceMember `gorm:"foreignKey:WorkspaceID" json:"members,omitempty"`
}

type WorkspaceMember struct {
	ID          uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	WorkspaceID uint          `gorm:"not null;uniqueIndex:idx_workspace_user" json:"workspace_id"`
	UserID      uint          `gorm:"not null;uniqueIndex:idx_workspace_user" json:"user_id"`
	Role        WorkspaceRole `gorm:"not null;default:'editor'" json:"role"`
	JoinedAt    time.Time     `json:"joined_at"`
	LastSeenAt  *time.Time    `json:"last_seen_at"`
}

type WorkspaceInvite struct {
	ID          uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	WorkspaceID uint          `gorm:"not null;index" json:"workspace_id"`
	InvitedBy   uint          `gorm:"not null" json:"invited_by"`
	Code        string        `gorm:"uniqueIndex;not null" json:"code"`
	Role        WorkspaceRole `gorm:"not null;default:'editor'" json:"role"`
	ExpiresAt   time.Time     `json:"expires_at"`
	UsedAt      *time.Time    `json:"used_at"`
	UsedBy      *uint         `json:"used_by"`
	CreatedAt   time.Time     `json:"created_at"`
}
