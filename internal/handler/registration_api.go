package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/morinonusi421/cupid/internal/liff"
	"github.com/morinonusi421/cupid/internal/service"
)

type RegistrationAPIHandler struct {
	messageService service.MessageService
	liffVerifier   *liff.Verifier
}

func NewRegistrationAPIHandler(messageService service.MessageService, liffVerifier *liff.Verifier) *RegistrationAPIHandler {
	return &RegistrationAPIHandler{
		messageService: messageService,
		liffVerifier:   liffVerifier,
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
	userID, err := h.liffVerifier.VerifyAccessToken(accessToken)
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

	// TODO: Save user data using messageService
	log.Printf("Registration request for user %s: name=%s, birthday=%s", userID, req.Name, req.Birthday)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
