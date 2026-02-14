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
type mockLIFFVerifier struct{}

func (m *mockLIFFVerifier) VerifyAccessToken(accessToken string) (string, error) {
	return "", nil
}

func (m *mockLIFFVerifier) VerifyIDToken(idToken string) (string, error) {
	// Accept tokens in format "test-token-{userID}"
	if len(idToken) > 11 && idToken[:11] == "test-token-" {
		return idToken[11:], nil
	}
	// For specific test tokens
	if idToken == "valid-token" {
		return "U-test-user", nil
	}
	return "", nil
}

func TestRegistrationAPI_Register_Success(t *testing.T) {
	mockUserService := servicemocks.NewMockUserService(t)
	mockVerifier := &mockLIFFVerifier{}
	handler := NewUserRegistrationAPIHandler(mockUserService, mockVerifier)

	// Mock RegisterUser to succeed
	mockUserService.On("RegisterUser", mock.Anything, "U-test-user", "田中太郎", "2000-01-15", false).Return(true, nil)

	reqBody := map[string]string{
		"name":     "田中太郎",
		"birthday": "2000-01-15",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/register-user", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockUserService.AssertExpectations(t)
}

func TestRegistrationAPI_Register_MatchedUserExists(t *testing.T) {
	mockUserService := servicemocks.NewMockUserService(t)
	mockVerifier := &mockLIFFVerifier{}
	handler := NewUserRegistrationAPIHandler(mockUserService, mockVerifier)

	// Mock RegisterUser to return MatchedUserExistsError
	matchedErr := &service.MatchedUserExistsError{
		MatchedUserName: "サトウハナコ",
	}
	mockUserService.On("RegisterUser", mock.Anything, "U-test-user", "ヤマダタロウ", "2000-01-15", false).Return(false, matchedErr)

	reqBody := map[string]interface{}{
		"name":            "ヤマダタロウ",
		"birthday":        "2000-01-15",
		"confirm_unmatch": false,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/register-user", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	// ステータスコードが409であることを確認
	assert.Equal(t, http.StatusConflict, rr.Code)

	// レスポンスボディを確認
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "matched_user_exists", resp["error"])
	assert.Contains(t, resp["message"], "サトウハナコさんとマッチング中です")

	mockUserService.AssertExpectations(t)
}
