package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/morinonusi421/cupid/internal/liff"
	"github.com/morinonusi421/cupid/internal/service"
)

type UserRegistrationAPIHandler struct {
	userService service.UserService
	verifier    liff.Verifier
}

func NewUserRegistrationAPIHandler(userService service.UserService, verifier liff.Verifier) *UserRegistrationAPIHandler {
	return &UserRegistrationAPIHandler{
		userService: userService,
		verifier:    verifier,
	}
}

type RegisterUserRequest struct {
	Name           string `json:"name"`
	Birthday       string `json:"birthday"`
	ConfirmUnmatch bool   `json:"confirm_unmatch"`
}

type RegisterUserResponse struct {
	Status              string `json:"status"`
	IsFirstRegistration bool   `json:"is_first_registration"`
}

func (h *UserRegistrationAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
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

	// リクエストボディからname, birthday, confirm_unmatchを取得
	var req RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// user_idはトークンから取得したものを使用
	isFirstRegistration, err := h.userService.RegisterFromLIFF(r.Context(), userID, req.Name, req.Birthday, req.ConfirmUnmatch)
	if err != nil {
		log.Printf("Failed to register user: %v", err)

		// matched_user_existsエラーの場合は特別なレスポンス
		if err.Error() == "matched_user_exists" {
			// 相手の名前を取得するためにユーザー情報を取得
			// TODO: サービスからエラーと一緒に相手の名前を返すようにリファクタリング
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "matched_user_exists",
				"message": "現在マッチング中です。変更するとマッチングが解除されます。",
			})
			return
		}

		// duplicate_userエラーの場合は特別なレスポンス
		if err.Error() == "duplicate_user" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "duplicate_user",
				"message": "同じ名前・誕生日のユーザーが既に登録されています。",
			})
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("Registration successful for user %s: name=%s, birthday=%s", userID, req.Name, req.Birthday)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(RegisterUserResponse{
		Status:              "ok",
		IsFirstRegistration: isFirstRegistration,
	})
}
