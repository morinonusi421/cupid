package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/morinonusi421/cupid/internal/message"
	"github.com/morinonusi421/cupid/internal/middleware"
	"github.com/morinonusi421/cupid/internal/service"
	"github.com/morinonusi421/cupid/pkg/httputil"
)

type UserRegistrationAPIHandler struct {
	userService service.UserService
}

func NewUserRegistrationAPIHandler(userService service.UserService) *UserRegistrationAPIHandler {
	return &UserRegistrationAPIHandler{
		userService: userService,
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
	// context から user_id を取得
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		log.Printf("Failed to get user_id from context")
		httputil.WriteJSONError(w, http.StatusUnauthorized, map[string]string{"error": "認証に失敗しました"})
		return
	}

	// リクエストボディからname, birthday, confirm_unmatchを取得
	var req RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		httputil.WriteJSONError(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	// user_idはcontextから取得したものを使用
	isFirstRegistration, err := h.userService.RegisterUser(r.Context(), userID, req.Name, req.Birthday, req.ConfirmUnmatch)
	if err != nil {
		log.Printf("Failed to register user: %v", err)

		// matched_user_existsエラーの場合は特別なレスポンス
		var matchedErr *service.MatchedUserExistsError
		if errors.As(err, &matchedErr) {
			warningMsg := message.MatchedUserExistsWarning(matchedErr.MatchedUserName)
			httputil.WriteJSONError(w, http.StatusConflict, map[string]string{
				"error":   "matched_user_exists",
				"message": warningMsg,
			})
			return
		}

		// duplicate_userエラーの場合は特別なレスポンス
		if errors.Is(err, service.ErrDuplicateUser) {
			httputil.WriteJSONError(w, http.StatusConflict, map[string]string{
				"error":   "duplicate_user",
				"message": message.DuplicateUserError,
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

	log.Printf("Registration successful for user %s: name=%s, birthday=%s", userID, req.Name, req.Birthday)

	httputil.WriteJSONResponse(w, http.StatusOK, RegisterUserResponse{
		Status:              "ok",
		IsFirstRegistration: isFirstRegistration,
	})
}
