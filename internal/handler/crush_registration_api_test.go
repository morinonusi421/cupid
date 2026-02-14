package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/morinonusi421/cupid/internal/service"
	servicemocks "github.com/morinonusi421/cupid/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockLIFFVerifier is a mock implementation for testing
type mockCrushLIFFVerifier struct{}

func (m *mockCrushLIFFVerifier) VerifyAccessToken(accessToken string) (string, error) {
	return "", nil
}

func (m *mockCrushLIFFVerifier) VerifyIDToken(idToken string) (string, error) {
	// For specific test tokens
	if idToken == "valid-token" {
		return "U_TEST", nil
	}
	if len(idToken) > 11 && idToken[:11] == "test-token-" {
		return idToken[11:], nil
	}
	return "", nil
}

func TestCrushRegistrationAPIHandler_RegisterCrush_NoMatch(t *testing.T) {
	mockUserService := servicemocks.NewMockUserService(t)
	mockVerifier := &mockCrushLIFFVerifier{}
	handler := NewCrushRegistrationAPIHandler(mockUserService, mockVerifier, "https://example.com/register")

	// Mock RegisterCrush to return no match
	mockUserService.On("RegisterCrush", mock.Anything, "U_TEST", "佐藤花子", "1992-02-02", false).Return(false, "", true, nil)

	// リクエスト作成
	reqBody := RegisterCrushRequest{
		CrushName:     "佐藤花子",
		CrushBirthday: "1992-02-02",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/register-crush", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token")

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
	assert.True(t, resp.IsFirstRegistration)

	mockUserService.AssertExpectations(t)
}

func TestCrushRegistrationAPIHandler_RegisterCrush_SelfRegistrationError(t *testing.T) {
	mockUserService := servicemocks.NewMockUserService(t)
	mockVerifier := &mockCrushLIFFVerifier{}
	handler := NewCrushRegistrationAPIHandler(mockUserService, mockVerifier, "https://example.com/register")

	// Mock RegisterCrush to return self-registration error
	mockUserService.On("RegisterCrush", mock.Anything, "U_SELF", "山田太郎", "1990-01-01", false).Return(false, "", false, service.ErrCannotRegisterYourself)

	reqBody := RegisterCrushRequest{
		CrushName:     "山田太郎",
		CrushBirthday: "1990-01-01",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/register-crush", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token-U_SELF")

	w := httptest.NewRecorder()
	handler.RegisterCrush(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "自分自身は登録できません", resp["error"])

	mockUserService.AssertExpectations(t)
}
