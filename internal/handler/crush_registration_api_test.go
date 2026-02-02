package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCrushRegistrationAPIHandler_RegisterCrush_NoMatch(t *testing.T) {
	mockUserService := new(MockUserServiceForAPI)
	handler := NewCrushRegistrationAPIHandler(mockUserService)

	// Mock RegisterCrush to return no match
	mockUserService.On("RegisterCrush", mock.Anything, "U_TEST", "佐藤花子", "1992-02-02").Return(false, "", nil)

	// リクエスト作成
	reqBody := RegisterCrushRequest{
		UserID:        "U_TEST",
		CrushName:     "佐藤花子",
		CrushBirthday: "1992-02-02",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/register-crush", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// レスポンス記録
	w := httptest.NewRecorder()

	// ハンドラー実行
	handler.RegisterCrush(w, req)

	// ステータスコード確認
	assert.Equal(t, http.StatusOK, w.Code)

	// レスポンスボディ確認
	var resp RegisterCrushResponse
	json.NewDecoder(w.Body).Decode(&resp)

	assert.Equal(t, "ok", resp.Status)
	assert.False(t, resp.Matched)
	assert.Equal(t, "登録しました。相手があなたを登録したらマッチングします。", resp.Message)

	mockUserService.AssertExpectations(t)
}

func TestCrushRegistrationAPIHandler_RegisterCrush_SelfRegistrationError(t *testing.T) {
	mockUserService := new(MockUserServiceForAPI)
	handler := NewCrushRegistrationAPIHandler(mockUserService)

	// Mock RegisterCrush to return self-registration error
	selfRegError := errors.New("cannot register yourself")
	mockUserService.On("RegisterCrush", mock.Anything, "U_SELF", "山田太郎", "1990-01-01").Return(false, "", selfRegError)

	reqBody := RegisterCrushRequest{
		UserID:        "U_SELF",
		CrushName:     "山田太郎",
		CrushBirthday: "1990-01-01",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/register-crush", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.RegisterCrush(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "自分自身は登録できません", resp["error"])

	mockUserService.AssertExpectations(t)
}
