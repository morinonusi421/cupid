package handler

import (
	"encoding/json"
	"log"
	"net/http"

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
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Birthday string `json:"birthday"`
}

func (h *RegistrationAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
	// TODO: セキュリティ改善 - ワンタイムトークン方式に変更する
	// 現在はリクエストボディに直接user_idを含めているが、なりすまし可能
	// 将来的にはサーバー生成のワンタイムトークンを使用すべき

	// Decode request body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// Validate user_id
	if req.UserID == "" {
		log.Println("Missing user_id in request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id is required"})
		return
	}

	// Save user data using userService
	if err := h.userService.RegisterFromLIFF(r.Context(), req.UserID, req.Name, req.Birthday); err != nil {
		log.Printf("Failed to register user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "registration failed"})
		return
	}

	log.Printf("Registration successful for user %s: name=%s, birthday=%s", req.UserID, req.Name, req.Birthday)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
