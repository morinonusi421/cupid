package httputil

import (
	"encoding/json"
	"log"
	"net/http"
)

// WriteJSONError は JSON エラーレスポンスを書き込むヘルパー関数
func WriteJSONError(w http.ResponseWriter, statusCode int, errorData map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(errorData); err != nil {
		log.Printf("Failed to encode JSON error response: %v", err)
	}
}

// WriteJSONResponse は JSON レスポンスを書き込むヘルパー関数
func WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}
