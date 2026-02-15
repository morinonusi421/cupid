package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/morinonusi421/cupid/internal/message"
	"github.com/morinonusi421/cupid/internal/middleware"
	"github.com/morinonusi421/cupid/internal/service"
	"github.com/morinonusi421/cupid/pkg/httputil"
)

type CrushRegistrationAPIHandler struct {
	userService service.UserService
	userLiffURL string
}

func NewCrushRegistrationAPIHandler(userService service.UserService, userLiffURL string) *CrushRegistrationAPIHandler {
	return &CrushRegistrationAPIHandler{
		userService: userService,
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
	// context から user_id を取得
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		log.Printf("Failed to get user_id from context")
		httputil.WriteJSONError(w, http.StatusUnauthorized, map[string]string{"error": "認証に失敗しました"})
		return
	}

	// リクエストボディをデコード
	var req RegisterCrushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		httputil.WriteJSONError(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	// バリデーション
	if req.CrushName == "" || req.CrushBirthday == "" {
		log.Println("Missing crush_name or crush_birthday in request")
		httputil.WriteJSONError(w, http.StatusBadRequest, map[string]string{"error": "crush_name and crush_birthday are required"})
		return
	}

	// サービス呼び出し（user_idはcontextから取得したものを使用）
	matched, isFirstCrushRegistration, err := h.userService.RegisterCrush(r.Context(), userID, req.CrushName, req.CrushBirthday, req.ConfirmUnmatch)
	if err != nil {
		log.Printf("Failed to register crush: %v", err)
		log.Printf("[DEBUG] Error type: %T, ErrUserNotFound: %v, errors.Is result: %v", err, service.ErrUserNotFound, errors.Is(err, service.ErrUserNotFound))

		// user_not_foundエラーの場合は特別なレスポンス（ユーザー登録を促す）
		if errors.Is(err, service.ErrUserNotFound) {
			log.Printf("[DEBUG] Matched ErrUserNotFound, returning 428")
			httputil.WriteJSONError(w, http.StatusPreconditionRequired, map[string]string{
				"error":         "user_not_found",
				"message":       message.CrushRegistrationUserNotFound(h.userLiffURL),
				"user_liff_url": h.userLiffURL,
			})
			return
		}

		// matched_user_existsエラーの場合は特別なレスポンス
		var matchedErr *service.MatchedUserExistsError
		if errors.As(err, &matchedErr) {
			message := fmt.Sprintf("%sさんとマッチング中です。変更するとマッチングが解除されます。", matchedErr.MatchedUserName)
			httputil.WriteJSONError(w, http.StatusConflict, map[string]string{
				"error":   "matched_user_exists",
				"message": message,
			})
			return
		}

		// 自己登録エラーの場合は400を返す
		if errors.Is(err, service.ErrCannotRegisterYourself) {
			httputil.WriteJSONError(w, http.StatusBadRequest, map[string]string{"error": "cannot_register_yourself"})
			return
		}

		httputil.WriteJSONError(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// レスポンス作成
	httputil.WriteJSONResponse(w, http.StatusOK, RegisterCrushResponse{
		Status:              "ok",
		Matched:             matched,
		IsFirstRegistration: isFirstCrushRegistration,
	})

	if matched {
		log.Printf("Match established for user %s with %s", userID, req.CrushName)
	} else {
		log.Printf("Crush registration successful for user %s: crush=%s", userID, req.CrushName)
	}
}
