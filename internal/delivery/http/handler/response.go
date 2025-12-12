package handler

import (
	"encoding/json"
	"net/http"
)

// Response represents a standard API response
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// SendJSON sends a JSON response
func SendJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// SendSuccess sends a successful JSON response
func SendSuccess(w http.ResponseWriter, message string, data any) {
	SendJSON(w, http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SendError sends an error JSON response
func SendError(w http.ResponseWriter, message string, statusCode int) {
	SendJSON(w, statusCode, Response{
		Success: false,
		Message: message,
	})
}
