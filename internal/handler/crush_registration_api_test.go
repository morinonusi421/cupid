package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/morinonusi421/cupid/internal/middleware"
	"github.com/morinonusi421/cupid/internal/service"
	servicemocks "github.com/morinonusi421/cupid/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCrushRegistrationAPIHandler_RegisterCrush(t *testing.T) {
	tests := []struct {
		name                    string
		requestBody             interface{}
		hasUserID               bool
		userID                  string
		mockSetup               func(*servicemocks.MockUserService)
		expectedStatusCode      int
		expectedMatched         *bool
		expectedFirstReg        *bool
		expectedError           string
		expectedStatus          string
	}{
		{
			name: "正常系 - マッチなし（初回登録）",
			requestBody: map[string]interface{}{
				"crush_name":     "サトウハナコ",
				"crush_birthday": "1992-02-02",
			},
			hasUserID: true,
			userID:    "U-test-user",
			mockSetup: func(m *servicemocks.MockUserService) {
				m.EXPECT().RegisterCrush(mock.Anything, "U-test-user", "サトウハナコ", "1992-02-02", false).
					Return(false, true, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedMatched:    boolPtr(false),
			expectedFirstReg:   boolPtr(true),
			expectedStatus:     "ok",
		},
		{
			name: "正常系 - マッチなし（更新）",
			requestBody: map[string]interface{}{
				"crush_name":     "タナカタロウ",
				"crush_birthday": "1990-01-01",
			},
			hasUserID: true,
			userID:    "U-existing-user",
			mockSetup: func(m *servicemocks.MockUserService) {
				m.EXPECT().RegisterCrush(mock.Anything, "U-existing-user", "タナカタロウ", "1990-01-01", false).
					Return(false, false, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedMatched:    boolPtr(false),
			expectedFirstReg:   boolPtr(false),
			expectedStatus:     "ok",
		},
		{
			name: "正常系 - マッチング成立",
			requestBody: map[string]interface{}{
				"crush_name":     "スズキイチロウ",
				"crush_birthday": "1988-08-08",
			},
			hasUserID: true,
			userID:    "U-matched-user",
			mockSetup: func(m *servicemocks.MockUserService) {
				m.EXPECT().RegisterCrush(mock.Anything, "U-matched-user", "スズキイチロウ", "1988-08-08", false).
					Return(true, false, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedMatched:    boolPtr(true),
			expectedFirstReg:   boolPtr(false),
			expectedStatus:     "ok",
		},
		{
			name: "異常系 - 自分自身を登録",
			requestBody: map[string]interface{}{
				"crush_name":     "ヤマダタロウ",
				"crush_birthday": "1990-01-01",
			},
			hasUserID: true,
			userID:    "U-self-user",
			mockSetup: func(m *servicemocks.MockUserService) {
				m.EXPECT().RegisterCrush(mock.Anything, "U-self-user", "ヤマダタロウ", "1990-01-01", false).
					Return(false, false, service.ErrCannotRegisterYourself)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "cannot_register_yourself",
		},
		{
			name: "異常系 - バリデーションエラー（名前）",
			requestBody: map[string]interface{}{
				"crush_name":     "山田太郎",
				"crush_birthday": "1990-01-01",
			},
			hasUserID: true,
			userID:    "U-validation-user",
			mockSetup: func(m *servicemocks.MockUserService) {
				validationErr := &service.ValidationError{
					Message: "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
				}
				m.EXPECT().RegisterCrush(mock.Anything, "U-validation-user", "山田太郎", "1990-01-01", false).
					Return(false, false, validationErr)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name: "異常系 - contextにUserIDがない",
			requestBody: map[string]interface{}{
				"crush_name":     "サトウハナコ",
				"crush_birthday": "1992-02-02",
			},
			hasUserID:          false,
			mockSetup:          func(m *servicemocks.MockUserService) {},
			expectedStatusCode: http.StatusUnauthorized,
			expectedError:      "認証に失敗しました",
		},
		{
			name:               "異常系 - 不正なJSON",
			requestBody:        "invalid json",
			hasUserID:          true,
			userID:             "U-test-user",
			mockSetup:          func(m *servicemocks.MockUserService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserService := servicemocks.NewMockUserService(t)
			tt.mockSetup(mockUserService)
			handler := NewCrushRegistrationAPIHandler(mockUserService, "https://example.com/register")

			// リクエストボディ作成
			var bodyBytes []byte
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/register-crush", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// contextにUserIDを設定（必要な場合）
			if tt.hasUserID {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			handler.RegisterCrush(rr, req)

			// ステータスコード確認
			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			// レスポンスボディ確認
			var resp map[string]interface{}
			err := json.NewDecoder(rr.Body).Decode(&resp)
			assert.NoError(t, err)

			if tt.expectedError != "" {
				assert.Contains(t, resp["error"], tt.expectedError)
			}
			if tt.expectedStatus != "" {
				assert.Equal(t, tt.expectedStatus, resp["status"])
			}
			if tt.expectedMatched != nil {
				assert.Equal(t, *tt.expectedMatched, resp["matched"])
			}
			if tt.expectedFirstReg != nil {
				assert.Equal(t, *tt.expectedFirstReg, resp["is_first_registration"])
			}

			mockUserService.AssertExpectations(t)
		})
	}
}

// boolPtr returns a pointer to the given bool value
func boolPtr(b bool) *bool {
	return &b
}
