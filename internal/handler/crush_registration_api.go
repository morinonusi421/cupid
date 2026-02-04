package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/morinonusi421/cupid/internal/service"
)

type CrushRegistrationAPIHandler struct {
	userService service.UserService
}

func NewCrushRegistrationAPIHandler(userService service.UserService) *CrushRegistrationAPIHandler {
	return &CrushRegistrationAPIHandler{
		userService: userService,
	}
}

type RegisterCrushRequest struct {
	UserID        string `json:"user_id"`
	CrushName     string `json:"crush_name"`
	CrushBirthday string `json:"crush_birthday"`
}

type RegisterCrushResponse struct {
	Status  string `json:"status"`
	Matched bool   `json:"matched"`
	Message string `json:"message"`
}

func (h *CrushRegistrationAPIHandler) RegisterCrush(w http.ResponseWriter, r *http.Request) {
	// TODO: セキュリティ改善 - ワンタイムトークン方式に変更する

	// リクエストボディをデコード
	var req RegisterCrushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// バリデーション
	if req.UserID == "" {
		log.Println("Missing user_id in request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id is required"})
		return
	}
	if req.CrushName == "" || req.CrushBirthday == "" {
		log.Println("Missing crush_name or crush_birthday in request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "crush_name and crush_birthday are required"})
		return
	}

	// サービス呼び出し
	matched, matchedName, err := h.userService.RegisterCrush(r.Context(), req.UserID, req.CrushName, req.CrushBirthday)
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

	log.Printf("Crush registration successful for user %s: crush=%s, matched=%t", req.UserID, req.CrushName, matched)
}
