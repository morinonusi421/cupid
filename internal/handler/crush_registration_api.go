package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/morinonusi421/cupid/internal/liff"
	"github.com/morinonusi421/cupid/internal/service"
)

type CrushRegistrationAPIHandler struct {
	userService service.UserService
	verifier    liff.Verifier
}

func NewCrushRegistrationAPIHandler(userService service.UserService, verifier liff.Verifier) *CrushRegistrationAPIHandler {
	return &CrushRegistrationAPIHandler{
		userService: userService,
		verifier:    verifier,
	}
}

type RegisterCrushRequest struct {
	CrushName     string `json:"crush_name"`
	CrushBirthday string `json:"crush_birthday"`
}

type RegisterCrushResponse struct {
	Status  string `json:"status"`
	Matched bool   `json:"matched"`
	Message string `json:"message"`
}

func (h *CrushRegistrationAPIHandler) RegisterCrush(w http.ResponseWriter, r *http.Request) {
	// Authorizationヘッダーからトークンを取得
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

	// リクエストボディをデコード
	var req RegisterCrushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// バリデーション
	if req.CrushName == "" || req.CrushBirthday == "" {
		log.Println("Missing crush_name or crush_birthday in request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "crush_name and crush_birthday are required"})
		return
	}

	// サービス呼び出し（user_idはトークンから取得したものを使用）
	matched, matchedName, err := h.userService.RegisterCrush(r.Context(), userID, req.CrushName, req.CrushBirthday)
	if err != nil {
		log.Printf("Failed to register crush: %v", err)

		// 自己登録エラーの場合は400を返す
		if err.Error() == "cannot register yourself" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "自分自身は登録できません"})
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// レスポンス作成
	var message string
	if matched {
		message = matchedName + "さんとマッチしました！💘"
	} else {
		message = "登録しました。相手があなたを登録したらマッチングします。"
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(RegisterCrushResponse{
		Status:  "ok",
		Matched: matched,
		Message: message,
	})

	log.Printf("Crush registration successful for user %s: crush=%s, matched=%t", userID, req.CrushName, matched)
}
