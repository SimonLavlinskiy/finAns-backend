package dto

type HealthResponse struct {
	Status  string `json:"status"`
	DB      string `json:"db"`
	Version string `json:"version"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}
