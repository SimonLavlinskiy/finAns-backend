package domain

import (
	"context"
	"time"
)

const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

type Project struct {
	ID                     int64
	Name                   string
	InitialBalanceKopecks  int64
	StartedAt              *time.Time
	CreatedAt              time.Time
}

type ProjectMember struct {
	ProjectID int64
	UserID    int64
	Role      string
}

type ProjectWithMembers struct {
	Project
	Members []ProjectMember
}

type ProjectRepository interface {
	Create(ctx context.Context, name string, initialBalance int64, startedAt *time.Time) (Project, error)
	Get(ctx context.Context, id int64) (Project, error)
	ListByUser(ctx context.Context, userID int64) ([]Project, error)
	AddMember(ctx context.Context, projectID, userID int64, role string) error
	RemoveMember(ctx context.Context, projectID, userID int64) error
	GetMember(ctx context.Context, projectID, userID int64) (ProjectMember, error)
	ListMembers(ctx context.Context, projectID int64) ([]ProjectMember, error)
	CountOwners(ctx context.Context, projectID int64) (int64, error)
}
