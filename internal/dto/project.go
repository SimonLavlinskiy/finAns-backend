package dto

type CreateProjectRequest struct {
	Name                   string  `json:"name"`
	InitialBalanceKopecks  *int64  `json:"initial_balance_kopecks"`
	StartedAt              *string `json:"started_at"`
}

type ProjectResponse struct {
	ID                     int64   `json:"id"`
	Name                   string  `json:"name"`
	InitialBalanceKopecks  int64   `json:"initial_balance_kopecks"`
	StartedAt              *string `json:"started_at"`
	CreatedAt              string  `json:"created_at"`
}

type ProjectMemberResponse struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type AddMemberRequest struct {
	Username string `json:"username"`
}

type ProjectWithMembersResponse struct {
	ProjectResponse
	Members []ProjectMemberResponse `json:"members"`
}
