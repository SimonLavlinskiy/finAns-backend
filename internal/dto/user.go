package dto

type CreateUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type UserResponse struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}
