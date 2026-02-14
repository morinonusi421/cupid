package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/morinonusi421/cupid/internal/liff"
	"github.com/morinonusi421/cupid/internal/message"
	"github.com/morinonusi421/cupid/internal/service"
)

type CrushRegistrationAPIHandler struct {
	userService service.UserService
	verifier    liff.Verifier
	userLiffURL string
}

func NewCrushRegistrationAPIHandler(userService service.UserService, verifier liff.Verifier, userLiffURL string) *CrushRegistrationAPIHandler {
	return &CrushRegistrationAPIHandler{
		userService: userService,
		verifier:    verifier,
		userLiffURL: userLiffURL,
	}
}

type RegisterCrushRequest struct {
	CrushName      string `json:"crush_name"`
	CrushBirthday  string `json:"crush_birthday"`
	ConfirmUnmatch bool   `json:"confirm_unmatch"`
}

type RegisterCrushResponse struct {
	Status              string `json:"status"`
	Matched             bool   `json:"matched"`
	IsFirstRegistration bool   `json:"is_first_registration"`
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
	matched, matchedName, isFirstCrushRegistration, err := h.userService.RegisterCrush(r.Context(), userID, req.CrushName, req.CrushBirthday, req.ConfirmUnmatch)
	if err != nil {
		log.Printf("Failed to register crush: %v", err)
		log.Printf("[DEBUG] Error type: %T, ErrUserNotFound: %v, errors.Is result: %v", err, service.ErrUserNotFound, errors.Is(err, service.ErrUserNotFound))

		// user_not_foundエラーの場合は特別なレスポンス（ユーザー登録を促す）
		if errors.Is(err, service.ErrUserNotFound) {
			log.Printf("[DEBUG] Matched ErrUserNotFound, returning 428")
			w.WriteHeader(http.StatusPreconditionRequired) // 428 Precondition Required
			json.NewEncoder(w).Encode(map[string]string{
				"error":         "user_not_found",
				"message":       message.CrushRegistrationUserNotFound(h.userLiffURL),
				"user_liff_url": h.userLiffURL,
			})
			return
		}

		// matched_user_existsエラーの場合は特別なレスポンス
		var matchedErr *service.MatchedUserExistsError
		if errors.As(err, &matchedErr) {
			w.WriteHeader(http.StatusConflict)
			message := fmt.Sprintf("%sさんとマッチング中です。変更するとマッチングが解除されます。", matchedErr.MatchedUserName)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "matched_user_exists",
				"message": message,
			})
			return
		}

		// 自己登録エラーの場合は400を返す
		if errors.Is(err, service.ErrCannotRegisterYourself) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "自分自身は登録できません"})
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// レスポンス作成
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(RegisterCrushResponse{
		Status:              "ok",
		Matched:             matched,
		IsFirstRegistration: isFirstCrushRegistration,
	})

	if matched {
		log.Printf("Crush registration successful for user %s: crush=%s, matched=%t, matched_with=%s", userID, req.CrushName, matched, matchedName)
	} else {
		log.Printf("Crush registration successful for user %s: crush=%s, matched=%t", userID, req.CrushName, matched)
	}
}
