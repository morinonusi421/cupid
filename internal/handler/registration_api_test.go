package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

type MockUserServiceForAPI struct {
	mock.Mock
}

func (m *MockUserServiceForAPI) ProcessTextMessage(ctx context.Context, userID string) (string, string, string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

func (m *MockUserServiceForAPI) RegisterFromLIFF(ctx context.Context, userID, name, birthday string, confirmUnmatch bool) (bool, error) {
	args := m.Called(ctx, userID, name, birthday, confirmUnmatch)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserServiceForAPI) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string, confirmUnmatch bool) (matched bool, matchedUserName string, isFirstCrushRegistration bool, err error) {
	args := m.Called(ctx, userID, crushName, crushBirthday, confirmUnmatch)
	return args.Bool(0), args.String(1), args.Bool(2), args.Error(3)
}

func (m *MockUserServiceForAPI) HandleFollowEvent(ctx context.Context, replyToken string) error {
	args := m.Called(ctx, replyToken)
	return args.Error(0)
}

func TestRegistrationAPI_Register_Success(t *testing.T) {
	mockUserService := new(MockUserServiceForAPI)
	mockVerifier := &mockLIFFVerifier{}
	handler := NewRegistrationAPIHandler(mockUserService, mockVerifier)

	// Mock RegisterFromLIFF to succeed
	mockUserService.On("RegisterFromLIFF", mock.Anything, "U-test-user", "田中太郎", "2000-01-15", false).Return(true, nil)

	reqBody := map[string]string{
		"name":     "田中太郎",
		"birthday": "2000-01-15",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockUserService.AssertExpectations(t)
}
