package service

import (
	"context"
	"errors"
	"testing"

	"github.com/aarondl/null/v8"
	"github.com/morinonusi421/cupid/internal/message"
	"github.com/morinonusi421/cupid/internal/model"
	repositorymocks "github.com/morinonusi421/cupid/internal/repository/mocks"
	servicemocks "github.com/morinonusi421/cupid/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ========================================
// ProcessTextMessage のテスト
// ========================================

func TestUserService_ProcessTextMessage(t *testing.T) {
	tests := []struct {
		name              string
		userID            string
		mockSetup         func(*repositorymocks.MockUserRepository)
		expectedReplyText string
		expectedQuickURL  string
		expectedQuickLabel string
		expectedError      bool
	}{
		{
			name:   "ユーザー未登録 - ユーザー登録フォームを案内",
			userID: "U-new",
			mockSetup: func(m *repositorymocks.MockUserRepository) {
				m.EXPECT().FindByLineID(mock.Anything, "U-new").Return(nil, nil)
			},
			expectedReplyText:  message.UnregisteredUserPrompt,
			expectedQuickURL:   "https://liff.example.com/user",
			expectedQuickLabel: "登録する",
			expectedError:      false,
		},
		{
			name:   "ユーザー登録済み、好きな人未登録 - 好きな人登録フォームを案内",
			userID: "U-alice",
			mockSetup: func(m *repositorymocks.MockUserRepository) {
				m.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(&model.User{
					LineID:   "U-alice",
					Name:     "アリス",
					Birthday: "1990-01-01",
					// CrushName未設定
				}, nil)
			},
			expectedReplyText:  message.RegistrationStep1Prompt,
			expectedQuickURL:   "https://liff.example.com/crush",
			expectedQuickLabel: "好きな人を登録",
			expectedError:      false,
		},
		{
			name:   "全て登録済み - 登録完了メッセージ",
			userID: "U-alice",
			mockSetup: func(m *repositorymocks.MockUserRepository) {
				m.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(&model.User{
					LineID:        "U-alice",
					Name:          "アリス",
					Birthday:      "1990-01-01",
					CrushName:     null.StringFrom("ボブ"),
					CrushBirthday: null.StringFrom("1995-05-05"),
				}, nil)
			},
			expectedReplyText:  message.AlreadyRegisteredMessage,
			expectedQuickURL:   "",
			expectedQuickLabel: "",
			expectedError:      false,
		},
		{
			name:   "DBエラー",
			userID: "U-alice",
			mockSetup: func(m *repositorymocks.MockUserRepository) {
				m.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(nil, errors.New("db error"))
			},
			expectedReplyText:  "",
			expectedQuickURL:   "",
			expectedQuickLabel: "",
			expectedError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repositorymocks.NewMockUserRepository(t)
			mockMatchingService := servicemocks.NewMockMatchingService(t)
			mockNotificationService := servicemocks.NewMockNotificationService(t)

			tt.mockSetup(mockRepo)

			service := NewUserService(
				mockRepo,
				"https://liff.example.com/user",
				"https://liff.example.com/crush",
				mockMatchingService,
				mockNotificationService,
			)

			replyText, quickURL, quickLabel, err := service.ProcessTextMessage(context.Background(), tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedReplyText, replyText)
				assert.Equal(t, tt.expectedQuickURL, quickURL)
				assert.Equal(t, tt.expectedQuickLabel, quickLabel)
			}
		})
	}
}

// ========================================
// RegisterUser のテスト
// ========================================

func TestUserService_RegisterUser(t *testing.T) {
	tests := []struct {
		name                  string
		userID                string
		userName              string
		birthday              string
		confirmUnmatch        bool
		mockSetup             func(*repositorymocks.MockUserRepository, *servicemocks.MockMatchingService, *servicemocks.MockNotificationService)
		expectedIsFirstReg    bool
		expectedError         bool
		expectedErrorContains string
	}{
		{
			name:           "初回登録 - 正常系",
			userID:         "U-new",
			userName:       "アリス",
			birthday:       "1990-01-01",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// 重複チェック
				repo.EXPECT().FindByNameAndBirthday(mock.Anything, "アリス", "1990-01-01").Return(nil, nil)
				// ユーザー検索（未登録）
				repo.EXPECT().FindByLineID(mock.Anything, "U-new").Return(nil, nil)
				// 新規作成
				repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
					return u.LineID == "U-new" && u.Name == "アリス" && u.Birthday == "1990-01-01"
				})).Return(nil)
				// 好きな人登録促進メッセージ送信
				notif.EXPECT().SendCrushRegistrationPrompt(mock.Anything, "U-new", "https://liff.example.com/crush").Return(nil)
			},
			expectedIsFirstReg: true,
			expectedError:      false,
		},
		{
			name:           "バリデーションエラー - 名前が不正（漢字）",
			userID:         "U-new",
			userName:       "山田太郎",
			birthday:       "1990-01-01",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// バリデーションで弾かれるため、DB操作は行われない
			},
			expectedIsFirstReg:    false,
			expectedError:         true,
			expectedErrorContains: "名前は全角カタカナ",
		},
		{
			name:           "重複エラー - 他人が同じ名前・誕生日",
			userID:         "U-new",
			userName:       "アリス",
			birthday:       "1990-01-01",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// 他人が見つかる
				repo.EXPECT().FindByNameAndBirthday(mock.Anything, "アリス", "1990-01-01").Return(&model.User{
					LineID:   "U-other",
					Name:     "アリス",
					Birthday: "1990-01-01",
				}, nil)
			},
			expectedIsFirstReg:    false,
			expectedError:         true,
			expectedErrorContains: "duplicate user",
		},
		{
			name:           "更新 - 正常系（マッチなし）",
			userID:         "U-alice",
			userName:       "アリスタロウ",
			birthday:       "1990-12-25",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// 重複チェック
				repo.EXPECT().FindByNameAndBirthday(mock.Anything, "アリスタロウ", "1990-12-25").Return(nil, nil)
				// ユーザー検索（既存）
				repo.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(&model.User{
					LineID:   "U-alice",
					Name:     "アリス",
					Birthday: "1990-01-01",
				}, nil)
				// 更新
				repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
					return u.LineID == "U-alice" && u.Name == "アリスタロウ" && u.Birthday == "1990-12-25"
				})).Return(nil)
				// 好きな人未登録なので、checkAndNotifyMatchは早期リターンし、MatchingServiceは呼ばれない
				// 更新完了メッセージ
				notif.EXPECT().SendUserInfoUpdateConfirmation(mock.Anything, "U-alice").Return(nil)
			},
			expectedIsFirstReg: false,
			expectedError:      false,
		},
		{
			name:           "更新 - マッチング中エラー（confirmUnmatch=false）",
			userID:         "U-alice",
			userName:       "アリスタロウ",
			birthday:       "1990-12-25",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// 重複チェック
				repo.EXPECT().FindByNameAndBirthday(mock.Anything, "アリスタロウ", "1990-12-25").Return(nil, nil)
				// ユーザー検索（マッチング中）
				repo.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(&model.User{
					LineID:            "U-alice",
					Name:              "アリス",
					Birthday:          "1990-01-01",
					MatchedWithUserID: null.StringFrom("U-bob"),
				}, nil)
				// 相手の情報を取得
				repo.EXPECT().FindByLineID(mock.Anything, "U-bob").Return(&model.User{
					LineID:   "U-bob",
					Name:     "ボブ",
					Birthday: "1995-05-05",
				}, nil)
			},
			expectedIsFirstReg:    false,
			expectedError:         true,
			expectedErrorContains: "matched user exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repositorymocks.NewMockUserRepository(t)
			mockMatchingService := servicemocks.NewMockMatchingService(t)
			mockNotificationService := servicemocks.NewMockNotificationService(t)

			tt.mockSetup(mockRepo, mockMatchingService, mockNotificationService)

			service := NewUserService(
				mockRepo,
				"https://liff.example.com/user",
				"https://liff.example.com/crush",
				mockMatchingService,
				mockNotificationService,
			)

			isFirst, err := service.RegisterUser(context.Background(), tt.userID, tt.userName, tt.birthday, tt.confirmUnmatch)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedErrorContains != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedIsFirstReg, isFirst)
			}
		})
	}
}

// ========================================
// RegisterCrush のテスト
// ========================================

func TestUserService_RegisterCrush(t *testing.T) {
	tests := []struct {
		name                       string
		userID                     string
		crushName                  string
		crushBirthday              string
		confirmUnmatch             bool
		mockSetup                  func(*repositorymocks.MockUserRepository, *servicemocks.MockMatchingService, *servicemocks.MockNotificationService)
		expectedMatched            bool
		expectedIsFirstCrushReg    bool
		expectedError              bool
		expectedErrorContains      string
	}{
		{
			name:           "初回登録 - 正常系（マッチなし）",
			userID:         "U-alice",
			crushName:      "ボブ",
			crushBirthday:  "1995-05-05",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// ユーザー検索
				repo.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(&model.User{
					LineID:   "U-alice",
					Name:     "アリス",
					Birthday: "1990-01-01",
					// CrushName未設定（初回登録）
				}, nil)
				// 更新
				repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
					return u.LineID == "U-alice" && u.CrushName.String == "ボブ" && u.CrushBirthday.String == "1995-05-05"
				})).Return(nil)
				// マッチング判定（マッチなし）
				matching.EXPECT().CheckAndUpdateMatch(mock.Anything, mock.Anything).Return(false, nil, nil)
				// 登録完了メッセージ（初回）
				notif.EXPECT().SendCrushRegistrationComplete(mock.Anything, "U-alice", true).Return(nil)
			},
			expectedMatched:         false,
			expectedIsFirstCrushReg: true,
			expectedError:           false,
		},
		{
			name:           "再登録 - 正常系（マッチなし）",
			userID:         "U-alice",
			crushName:      "チャーリー",
			crushBirthday:  "1992-03-15",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// ユーザー検索（既に好きな人登録済み）
				repo.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(&model.User{
					LineID:        "U-alice",
					Name:          "アリス",
					Birthday:      "1990-01-01",
					CrushName:     null.StringFrom("ボブ"),
					CrushBirthday: null.StringFrom("1995-05-05"),
				}, nil)
				// 更新
				repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
					return u.LineID == "U-alice" && u.CrushName.String == "チャーリー"
				})).Return(nil)
				// マッチング判定（マッチなし）
				matching.EXPECT().CheckAndUpdateMatch(mock.Anything, mock.Anything).Return(false, nil, nil)
				// 登録完了メッセージ（再登録）
				notif.EXPECT().SendCrushRegistrationComplete(mock.Anything, "U-alice", false).Return(nil)
			},
			expectedMatched:         false,
			expectedIsFirstCrushReg: false,
			expectedError:           false,
		},
		{
			name:           "マッチング成立",
			userID:         "U-alice",
			crushName:      "ボブ",
			crushBirthday:  "1995-05-05",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// ユーザー検索
				repo.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(&model.User{
					LineID:   "U-alice",
					Name:     "アリス",
					Birthday: "1990-01-01",
				}, nil)
				// 更新
				repo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)
				// マッチング判定（マッチあり）
				matching.EXPECT().CheckAndUpdateMatch(mock.Anything, mock.Anything).Return(true, &model.User{
					LineID:   "U-bob",
					Name:     "ボブ",
					Birthday: "1995-05-05",
				}, nil)
				// マッチ通知（両方）
				notif.EXPECT().SendMatchNotification(mock.Anything, "U-alice", "ボブ").Return(nil)
				notif.EXPECT().SendMatchNotification(mock.Anything, "U-bob", "アリス").Return(nil)
			},
			expectedMatched:         true,
			expectedIsFirstCrushReg: true,
			expectedError:           false,
		},
		{
			name:           "バリデーションエラー - 名前が不正（ひらがな）",
			userID:         "U-alice",
			crushName:      "やまだはなこ",
			crushBirthday:  "1995-05-05",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// ユーザー検索
				repo.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(&model.User{
					LineID:   "U-alice",
					Name:     "アリス",
					Birthday: "1990-01-01",
				}, nil)
			},
			expectedMatched:         false,
			expectedIsFirstCrushReg: false,
			expectedError:           true,
			expectedErrorContains:   "名前は全角カタカナ",
		},
		{
			name:           "自己登録エラー",
			userID:         "U-alice",
			crushName:      "アリス",
			crushBirthday:  "1990-01-01",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// ユーザー検索
				repo.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(&model.User{
					LineID:   "U-alice",
					Name:     "アリス",
					Birthday: "1990-01-01",
				}, nil)
			},
			expectedMatched:         false,
			expectedIsFirstCrushReg: false,
			expectedError:           true,
			expectedErrorContains:   "cannot register yourself",
		},
		{
			name:           "マッチング中エラー（confirmUnmatch=false）",
			userID:         "U-alice",
			crushName:      "チャーリー",
			crushBirthday:  "1992-03-15",
			confirmUnmatch: false,
			mockSetup: func(repo *repositorymocks.MockUserRepository, matching *servicemocks.MockMatchingService, notif *servicemocks.MockNotificationService) {
				// ユーザー検索（マッチング中）
				repo.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(&model.User{
					LineID:            "U-alice",
					Name:              "アリス",
					Birthday:          "1990-01-01",
					CrushName:         null.StringFrom("ボブ"),
					CrushBirthday:     null.StringFrom("1995-05-05"),
					MatchedWithUserID: null.StringFrom("U-bob"),
				}, nil)
				// 相手の情報を取得
				repo.EXPECT().FindByLineID(mock.Anything, "U-bob").Return(&model.User{
					LineID:   "U-bob",
					Name:     "ボブ",
					Birthday: "1995-05-05",
				}, nil)
			},
			expectedMatched:         false,
			expectedIsFirstCrushReg: false,
			expectedError:           true,
			expectedErrorContains:   "matched user exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repositorymocks.NewMockUserRepository(t)
			mockMatchingService := servicemocks.NewMockMatchingService(t)
			mockNotificationService := servicemocks.NewMockNotificationService(t)

			tt.mockSetup(mockRepo, mockMatchingService, mockNotificationService)

			service := NewUserService(
				mockRepo,
				"https://liff.example.com/user",
				"https://liff.example.com/crush",
				mockMatchingService,
				mockNotificationService,
			)

			matched, isFirstCrushReg, err := service.RegisterCrush(context.Background(), tt.userID, tt.crushName, tt.crushBirthday, tt.confirmUnmatch)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedErrorContains != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMatched, matched)
				assert.Equal(t, tt.expectedIsFirstCrushReg, isFirstCrushReg)
			}
		})
	}
}

// ========================================
// ProcessFollowEvent のテスト
// ========================================

func TestUserService_ProcessFollowEvent(t *testing.T) {
	tests := []struct {
		name          string
		replyToken    string
		mockSetup     func(*servicemocks.MockNotificationService)
		expectedError bool
	}{
		{
			name:       "正常系",
			replyToken: "reply-token-123",
			mockSetup: func(notif *servicemocks.MockNotificationService) {
				notif.EXPECT().SendFollowGreeting(mock.Anything, "reply-token-123", "https://liff.example.com/user").Return(nil)
			},
			expectedError: false,
		},
		{
			name:       "異常系 - 通知送信失敗",
			replyToken: "reply-token-123",
			mockSetup: func(notif *servicemocks.MockNotificationService) {
				notif.EXPECT().SendFollowGreeting(mock.Anything, "reply-token-123", "https://liff.example.com/user").Return(errors.New("api error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repositorymocks.NewMockUserRepository(t)
			mockMatchingService := servicemocks.NewMockMatchingService(t)
			mockNotificationService := servicemocks.NewMockNotificationService(t)

			tt.mockSetup(mockNotificationService)

			service := NewUserService(
				mockRepo,
				"https://liff.example.com/user",
				"https://liff.example.com/crush",
				mockMatchingService,
				mockNotificationService,
			)

			err := service.ProcessFollowEvent(context.Background(), tt.replyToken)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ========================================
// ProcessJoinEvent のテスト
// ========================================

func TestUserService_ProcessJoinEvent(t *testing.T) {
	tests := []struct {
		name          string
		replyToken    string
		mockSetup     func(*servicemocks.MockNotificationService)
		expectedError bool
	}{
		{
			name:       "正常系",
			replyToken: "reply-token-456",
			mockSetup: func(notif *servicemocks.MockNotificationService) {
				notif.EXPECT().SendJoinGroupGreeting(mock.Anything, "reply-token-456").Return(nil)
			},
			expectedError: false,
		},
		{
			name:       "異常系 - 通知送信失敗",
			replyToken: "reply-token-456",
			mockSetup: func(notif *servicemocks.MockNotificationService) {
				notif.EXPECT().SendJoinGroupGreeting(mock.Anything, "reply-token-456").Return(errors.New("api error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := repositorymocks.NewMockUserRepository(t)
			mockMatchingService := servicemocks.NewMockMatchingService(t)
			mockNotificationService := servicemocks.NewMockNotificationService(t)

			tt.mockSetup(mockNotificationService)

			service := NewUserService(
				mockRepo,
				"https://liff.example.com/user",
				"https://liff.example.com/crush",
				mockMatchingService,
				mockNotificationService,
			)

			err := service.ProcessJoinEvent(context.Background(), tt.replyToken)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
