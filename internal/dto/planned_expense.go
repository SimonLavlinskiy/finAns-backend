package dto

type PlannedExpenseCategoryResponse struct {
	ID        int64                    `json:"id"`
	Name      string                   `json:"name"`
	Color     string                   `json:"color"`
	SortOrder int                      `json:"sort_order"`
	Items     []PlannedExpenseResponse `json:"items"`
}

type PlannedExpenseResponse struct {
	ID                int64   `json:"id"`
	CategoryID        int64   `json:"category_id"`
	Title             string  `json:"title"`
	CostKopecks       *int64  `json:"cost_kopecks,omitempty"`
	DueDate           *string `json:"due_date,omitempty"`
	URL               *string `json:"url,omitempty"`
	Priority          string  `json:"priority"`
	EffectivePriority string  `json:"effective_priority"`
	IsDueSoon         bool    `json:"is_due_soon"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	ArchivedAt        *string `json:"archived_at,omitempty"`
}

type CreatePlannedExpenseCategoryRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type ReorderPlannedExpenseCategoriesRequest struct {
	IDs []int64 `json:"ids"`
}

type CreatePlannedExpenseRequest struct {
	Title       string                               `json:"title"`
	CostKopecks *int64                               `json:"cost_kopecks"`
	DueDate     *string                              `json:"due_date"`
	URL         *string                              `json:"url"`
	Priority    string                               `json:"priority"`
	CategoryID  *int64                               `json:"category_id"`
	NewCategory *CreatePlannedExpenseCategoryRequest `json:"new_category"`
}

type UpdatePlannedExpenseRequest struct {
	Title       string                               `json:"title"`
	CostKopecks *int64                               `json:"cost_kopecks"`
	DueDate     *string                              `json:"due_date"`
	URL         *string                              `json:"url"`
	Priority    string                               `json:"priority"`
	CategoryID  *int64                               `json:"category_id"`
	NewCategory *CreatePlannedExpenseCategoryRequest `json:"new_category"`
}
