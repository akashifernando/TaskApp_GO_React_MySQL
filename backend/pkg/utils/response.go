package utils

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success    bool        `json:"success,omitempty"`
	StatusCode int         `json:"statusCode"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
}

func SendJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func SendResponse(w http.ResponseWriter, status int, msg string, data interface{}) {
	SendJSON(w, status, Response{
		StatusCode: status,
		Message:    msg,
		Data:       data,
	})
}
