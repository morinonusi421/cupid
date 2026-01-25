package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/morinonusi421/cupid/internal/service"
)

type RegistrationAPIHandler struct {
	messageService service.MessageService
}

func NewRegistrationAPIHandler(messageService service.MessageService) *RegistrationAPIHandler {
	return &RegistrationAPIHandler{
		messageService: messageService,
	}
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Birthday string `json:"birthday"`
}

func (h *RegistrationAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
	// TODO: LIFF token verification will be added later

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// For now, just return 200 OK
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
