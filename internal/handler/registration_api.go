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
	userService service.UserService
	verifier    liff.Verifier
}

func NewRegistrationAPIHandler(userService service.UserService, verifier liff.Verifier) *RegistrationAPIHandler {
	return &RegistrationAPIHandler{
		userService: userService,
		verifier:    verifier,
	}
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Birthday string `json:"birthday"`
}

func (h *RegistrationAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Authorizationヘッダーからトークン取得
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "認証が必要です"})
		return
	}

	// "Bearer {token}" 形式からトークン抽出
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader { // Bearerプレフィックスがない
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "無効な認証形式です"})
		return
	}

	// トークン検証してuser_id取得
	userID, err := h.verifier.VerifyIDToken(token)
	if err != nil {
		log.Printf("Token verification failed: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "認証に失敗しました"})
		return
	}

	// リクエストボディからname, birthdayのみ取得
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// user_idはトークンから取得したものを使用
	if err := h.userService.RegisterFromLIFF(r.Context(), userID, req.Name, req.Birthday); err != nil {
		log.Printf("Failed to register user: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("Registration successful for user %s: name=%s, birthday=%s", userID, req.Name, req.Birthday)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
