package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/morinonusi421/cupid/internal/liff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMessageServiceForAPI struct {
	mock.Mock
}

func (m *MockMessageServiceForAPI) ProcessTextMessage(ctx context.Context, userID, text string) (string, error) {
	args := m.Called(ctx, userID, text)
	return args.String(0), args.Error(1)
}

func TestRegistrationAPI_Register_Success(t *testing.T) {
	mockMessageService := new(MockMessageServiceForAPI)
	mockVerifier := liff.NewVerifier("test-channel-id")
	handler := NewRegistrationAPIHandler(mockMessageService, mockVerifier)

	reqBody := map[string]string{
		"name":     "田中太郎",
		"birthday": "2000-01-15",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-liff-token")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
