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

func TestUserRegistrationAPIHandler_Register(t *testing.T) {
	tests := []struct {
		name               string
		requestBody        interface{}
		hasUserID          bool
		userID             string
		mockSetup          func(*servicemocks.MockUserService)
		expectedStatusCode int
		expectedError      string
		expectedStatus     string
	}{
		{
			name: "正常系 - 初回登録成功",
			requestBody: map[string]interface{}{
				"name":     "ヤマダタロウ",
				"birthday": "2000-01-15",
			},
			hasUserID: true,
			userID:    "U-test-user",
			mockSetup: func(m *servicemocks.MockUserService) {
				m.EXPECT().RegisterUser(mock.Anything, "U-test-user", "ヤマダタロウ", "2000-01-15", false).
					Return(true, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedStatus:     "ok",
		},
		{
			name: "正常系 - 情報更新成功",
			requestBody: map[string]interface{}{
				"name":     "タナカハナコ",
				"birthday": "1995-05-05",
			},
			hasUserID: true,
			userID:    "U-existing-user",
			mockSetup: func(m *servicemocks.MockUserService) {
				m.EXPECT().RegisterUser(mock.Anything, "U-existing-user", "タナカハナコ", "1995-05-05", false).
					Return(false, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedStatus:     "ok",
		},
		{
			name: "異常系 - マッチング中（confirmUnmatch=false）",
			requestBody: map[string]interface{}{
				"name":            "ヤマダタロウ",
				"birthday":        "2000-01-15",
				"confirm_unmatch": false,
			},
			hasUserID: true,
			userID:    "U-matched-user",
			mockSetup: func(m *servicemocks.MockUserService) {
				matchedErr := &service.MatchedUserExistsError{
					MatchedUserName: "サトウハナコ",
				}
				m.EXPECT().RegisterUser(mock.Anything, "U-matched-user", "ヤマダタロウ", "2000-01-15", false).
					Return(false, matchedErr)
			},
			expectedStatusCode: http.StatusConflict,
			expectedError:      "matched_user_exists",
		},
		{
			name: "異常系 - 重複ユーザー",
			requestBody: map[string]interface{}{
				"name":     "スズキイチロウ",
				"birthday": "1990-01-01",
			},
			hasUserID: true,
			userID:    "U-duplicate-user",
			mockSetup: func(m *servicemocks.MockUserService) {
				m.EXPECT().RegisterUser(mock.Anything, "U-duplicate-user", "スズキイチロウ", "1990-01-01", false).
					Return(false, service.ErrDuplicateUser)
			},
			expectedStatusCode: http.StatusConflict,
			expectedError:      "duplicate_user",
		},
		{
			name: "異常系 - バリデーションエラー",
			requestBody: map[string]interface{}{
				"name":     "山田太郎",
				"birthday": "2000-01-15",
			},
			hasUserID: true,
			userID:    "U-validation-user",
			mockSetup: func(m *servicemocks.MockUserService) {
				validationErr := &service.ValidationError{
					Message: "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
				}
				m.EXPECT().RegisterUser(mock.Anything, "U-validation-user", "山田太郎", "2000-01-15", false).
					Return(false, validationErr)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "名前は全角カタカナ2〜20文字で入力してください（スペース不可）",
		},
		{
			name: "異常系 - contextにUserIDがない",
			requestBody: map[string]interface{}{
				"name":     "ヤマダタロウ",
				"birthday": "2000-01-15",
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
			handler := NewUserRegistrationAPIHandler(mockUserService)

			// リクエストボディ作成
			var bodyBytes []byte
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/register-user", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// contextにUserIDを設定（必要な場合）
			if tt.hasUserID {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			handler.Register(rr, req)

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

			mockUserService.AssertExpectations(t)
		})
	}
}
