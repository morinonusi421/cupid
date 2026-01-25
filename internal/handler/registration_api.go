package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/morinonusi421/cupid/internal/service"
)

type RegistrationAPIHandler struct {
	userService service.UserService
}

func NewRegistrationAPIHandler(userService service.UserService) *RegistrationAPIHandler {
	return &RegistrationAPIHandler{
		userService: userService,
	}
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Birthday string `json:"birthday"`
}

func (h *RegistrationAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Extract LIFF access token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		log.Println("Missing or invalid Authorization header")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	// Verify LIFF access token
	userID, err := h.userService.VerifyLIFFToken(accessToken)
	if err != nil {
		log.Printf("Failed to verify LIFF token: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
		return
	}

	// Decode request body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// Save user data using userService
	if err := h.userService.RegisterFromLIFF(r.Context(), userID, req.Name, req.Birthday); err != nil {
		log.Printf("Failed to register user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "registration failed"})
		return
	}

	log.Printf("Registration successful for user %s: name=%s, birthday=%s", userID, req.Name, req.Birthday)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
