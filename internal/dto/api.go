package dto

type DataResponse[T any] struct {
	Data T `json:"data"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type UserResponse struct {
	Login string `json:"login"`
}

type TagParent struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type TagResponse struct {
	ID       int64         `json:"id"`
	Name     string        `json:"name"`
	Color    string        `json:"color"`
	ParentID *int64        `json:"parent_id,omitempty"`
	Parent   *TagParent    `json:"parent,omitempty"`
	Children []TagResponse `json:"children,omitempty"`
}

type CreateTagRequest struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	ParentID *int64 `json:"parent_id"`
}

type UpdateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type TagUsageResponse struct {
	Count int64 `json:"count"`
}

type TransactionTag struct {
	ID     int64      `json:"id"`
	Name   string     `json:"name"`
	Color  string     `json:"color"`
	Parent *TagParent `json:"parent,omitempty"`
}

type TransactionResponse struct {
	ID          int64           `json:"id"`
	Title       string          `json:"title"`
	Amount      int64           `json:"amount"`
	Date        string          `json:"date"`
	Tag         TransactionTag  `json:"tag"`
	Category    string          `json:"category"`
	Specificity string          `json:"specificity"`
	Comment     *string         `json:"comment"`
	URL         *string         `json:"url"`
	File        *FileAttachment `json:"file"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

type FileAttachment struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
}

type CreateTransactionRequest struct {
	Title       string  `json:"title"`
	Amount      int64   `json:"amount"`
	Date        string  `json:"date"`
	TagID       int64   `json:"tag_id"`
	Category    string  `json:"category"`
	Specificity string  `json:"specificity"`
	Comment     *string `json:"comment"`
	URL         *string `json:"url"`
}

type UpdateTransactionRequest struct {
	Title       string  `json:"title"`
	Amount      int64   `json:"amount"`
	Date        string  `json:"date"`
	TagID       int64   `json:"tag_id"`
	Category    string  `json:"category"`
	Specificity string  `json:"specificity"`
	Comment     *string `json:"comment"`
	URL         *string `json:"url"`
}

type BalanceResponse struct {
	Balance       int64 `json:"balance"`
	InitialAmount int64 `json:"initial_amount"`
	TotalIncome   int64 `json:"total_income"`
	TotalExpense  int64 `json:"total_expense"`
}

type UpdateBalanceRequest struct {
	Balance       *int64 `json:"balance"`
	InitialAmount *int64 `json:"initial_amount"`
}
