package service

import (
	"context"
	"errors"
	"testing"

	"github.com/aarondl/null/v8"
	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ========================================
// CheckAndUpdateMatch のテスト
// ========================================

func TestMatchingService_CheckAndUpdateMatch(t *testing.T) {
	tests := []struct {
		name              string
		currentUser       *model.User
		mockSetup         func(*mocks.MockUserRepository)
		expectedMatched   bool
		expectedUserName  string
		expectedError     bool
		expectedErrorMsg  string
	}{
		{
			name: "マッチなし - 相手がユーザーテーブルに未登録",
			currentUser: &model.User{
				LineID:        "U-alice",
				Name:          "アリス",
				Birthday:      "1990-01-01",
				CrushName:     null.StringFrom("ボブ"),
				CrushBirthday: null.StringFrom("1995-05-05"),
			},
			mockSetup: func(m *mocks.MockUserRepository) {
				m.EXPECT().FindMatchingUser(mock.Anything, mock.Anything).Return(nil, nil)
			},
			expectedMatched: false,
			expectedError:   false,
		},
		{
			name: "マッチなし - 相手が自分を登録していない",
			currentUser: &model.User{
				LineID:        "U-alice",
				Name:          "アリス",
				Birthday:      "1990-01-01",
				CrushName:     null.StringFrom("ボブ"),
				CrushBirthday: null.StringFrom("1995-05-05"),
			},
			mockSetup: func(m *mocks.MockUserRepository) {
				m.EXPECT().FindMatchingUser(mock.Anything, mock.Anything).Return(nil, nil)
			},
			expectedMatched: false,
			expectedError:   false,
		},
		{
			name: "マッチング成立",
			currentUser: &model.User{
				LineID:        "U-alice",
				Name:          "アリス",
				Birthday:      "1990-01-01",
				CrushName:     null.StringFrom("ボブ"),
				CrushBirthday: null.StringFrom("1995-05-05"),
			},
			mockSetup: func(m *mocks.MockUserRepository) {
				matchedUser := &model.User{
					LineID:        "U-bob",
					Name:          "ボブ",
					Birthday:      "1995-05-05",
					CrushName:     null.StringFrom("アリス"),
					CrushBirthday: null.StringFrom("1990-01-01"),
				}
				m.EXPECT().FindMatchingUser(mock.Anything, mock.Anything).Return(matchedUser, nil)
				m.EXPECT().
					Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
						return (u.LineID == "U-alice" && u.MatchedWithUserID.String == "U-bob") ||
							(u.LineID == "U-bob" && u.MatchedWithUserID.String == "U-alice")
					})).
					Return(nil).
					Times(2)
			},
			expectedMatched:  true,
			expectedUserName: "ボブ",
			expectedError:    false,
		},
		{
			name: "異常系 - FindMatchingUserエラー",
			currentUser: &model.User{
				LineID:        "U-alice",
				Name:          "アリス",
				Birthday:      "1990-01-01",
				CrushName:     null.StringFrom("ボブ"),
				CrushBirthday: null.StringFrom("1995-05-05"),
			},
			mockSetup: func(m *mocks.MockUserRepository) {
				m.EXPECT().FindMatchingUser(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))
			},
			expectedMatched:  false,
			expectedError:    true,
			expectedErrorMsg: "db error",
		},
		{
			name: "異常系 - currentUser更新エラー",
			currentUser: &model.User{
				LineID:        "U-alice",
				Name:          "アリス",
				Birthday:      "1990-01-01",
				CrushName:     null.StringFrom("ボブ"),
				CrushBirthday: null.StringFrom("1995-05-05"),
			},
			mockSetup: func(m *mocks.MockUserRepository) {
				matchedUser := &model.User{
					LineID:        "U-bob",
					Name:          "ボブ",
					Birthday:      "1995-05-05",
					CrushName:     null.StringFrom("アリス"),
					CrushBirthday: null.StringFrom("1990-01-01"),
				}
				m.EXPECT().FindMatchingUser(mock.Anything, mock.Anything).Return(matchedUser, nil)
				m.EXPECT().
					Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
						return u.LineID == "U-alice"
					})).
					Return(errors.New("db error"))
			},
			expectedMatched:  false,
			expectedError:    true,
			expectedErrorMsg: "db error",
		},
		{
			name: "異常系 - matchedUser更新エラー",
			currentUser: &model.User{
				LineID:        "U-alice",
				Name:          "アリス",
				Birthday:      "1990-01-01",
				CrushName:     null.StringFrom("ボブ"),
				CrushBirthday: null.StringFrom("1995-05-05"),
			},
			mockSetup: func(m *mocks.MockUserRepository) {
				matchedUser := &model.User{
					LineID:        "U-bob",
					Name:          "ボブ",
					Birthday:      "1995-05-05",
					CrushName:     null.StringFrom("アリス"),
					CrushBirthday: null.StringFrom("1990-01-01"),
				}
				m.EXPECT().FindMatchingUser(mock.Anything, mock.Anything).Return(matchedUser, nil)
				m.EXPECT().
					Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
						return u.LineID == "U-alice"
					})).
					Return(nil)
				m.EXPECT().
					Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
						return u.LineID == "U-bob"
					})).
					Return(errors.New("db error"))
			},
			expectedMatched:  false,
			expectedError:    true,
			expectedErrorMsg: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := mocks.NewMockUserRepository(t)
			tt.mockSetup(mockUserRepo)

			service := NewMatchingService(mockUserRepo)
			matched, matchedUser, err := service.CheckAndUpdateMatch(context.Background(), tt.currentUser)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				assert.False(t, matched)
				assert.Nil(t, matchedUser)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMatched, matched)
				if tt.expectedMatched {
					assert.NotNil(t, matchedUser)
					assert.Equal(t, tt.expectedUserName, matchedUser.Name)
				} else {
					assert.Nil(t, matchedUser)
				}
			}
		})
	}
}

// ========================================
// UnmatchUsers のテスト
// ========================================

func TestMatchingService_UnmatchUsers(t *testing.T) {
	tests := []struct {
		name             string
		initiatorUserID  string
		partnerUserID    string
		mockSetup        func(*mocks.MockUserRepository)
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:            "正常系 - マッチング解除成功",
			initiatorUserID: "U-alice",
			partnerUserID:   "U-bob",
			mockSetup: func(m *mocks.MockUserRepository) {
				initiatorUser := &model.User{
					LineID:            "U-alice",
					Name:              "アリス",
					Birthday:          "1990-01-01",
					MatchedWithUserID: null.StringFrom("U-bob"),
				}
				partnerUser := &model.User{
					LineID:            "U-bob",
					Name:              "ボブ",
					Birthday:          "1995-05-05",
					MatchedWithUserID: null.StringFrom("U-alice"),
				}
				m.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(initiatorUser, nil)
				m.EXPECT().FindByLineID(mock.Anything, "U-bob").Return(partnerUser, nil)
				m.EXPECT().
					Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
						return u.LineID == "U-alice" && !u.MatchedWithUserID.Valid
					})).
					Return(nil)
				m.EXPECT().
					Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
						return u.LineID == "U-bob" && !u.MatchedWithUserID.Valid
					})).
					Return(nil)
			},
			expectedError: false,
		},
		{
			name:            "異常系 - initiatorユーザーが見つからない",
			initiatorUserID: "U-alice",
			partnerUserID:   "U-bob",
			mockSetup: func(m *mocks.MockUserRepository) {
				m.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(nil, nil)
			},
			expectedError:    true,
			expectedErrorMsg: "initiator user not found",
		},
		{
			name:            "異常系 - initiatorユーザー検索エラー",
			initiatorUserID: "U-alice",
			partnerUserID:   "U-bob",
			mockSetup: func(m *mocks.MockUserRepository) {
				m.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(nil, errors.New("db error"))
			},
			expectedError:    true,
			expectedErrorMsg: "failed to find initiator user",
		},
		{
			name:            "異常系 - partnerユーザーが見つからない",
			initiatorUserID: "U-alice",
			partnerUserID:   "U-bob",
			mockSetup: func(m *mocks.MockUserRepository) {
				initiatorUser := &model.User{
					LineID:            "U-alice",
					Name:              "アリス",
					Birthday:          "1990-01-01",
					MatchedWithUserID: null.StringFrom("U-bob"),
				}
				m.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(initiatorUser, nil)
				m.EXPECT().FindByLineID(mock.Anything, "U-bob").Return(nil, nil)
			},
			expectedError:    true,
			expectedErrorMsg: "partner user not found",
		},
		{
			name:            "異常系 - partnerユーザー検索エラー",
			initiatorUserID: "U-alice",
			partnerUserID:   "U-bob",
			mockSetup: func(m *mocks.MockUserRepository) {
				initiatorUser := &model.User{
					LineID:            "U-alice",
					Name:              "アリス",
					Birthday:          "1990-01-01",
					MatchedWithUserID: null.StringFrom("U-bob"),
				}
				m.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(initiatorUser, nil)
				m.EXPECT().FindByLineID(mock.Anything, "U-bob").Return(nil, errors.New("db error"))
			},
			expectedError:    true,
			expectedErrorMsg: "failed to find partner user",
		},
		{
			name:            "異常系 - initiatorユーザー更新エラー",
			initiatorUserID: "U-alice",
			partnerUserID:   "U-bob",
			mockSetup: func(m *mocks.MockUserRepository) {
				initiatorUser := &model.User{
					LineID:            "U-alice",
					Name:              "アリス",
					Birthday:          "1990-01-01",
					MatchedWithUserID: null.StringFrom("U-bob"),
				}
				partnerUser := &model.User{
					LineID:            "U-bob",
					Name:              "ボブ",
					Birthday:          "1995-05-05",
					MatchedWithUserID: null.StringFrom("U-alice"),
				}
				m.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(initiatorUser, nil)
				m.EXPECT().FindByLineID(mock.Anything, "U-bob").Return(partnerUser, nil)
				m.EXPECT().
					Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
						return u.LineID == "U-alice"
					})).
					Return(errors.New("db error"))
			},
			expectedError:    true,
			expectedErrorMsg: "failed to update initiator user",
		},
		{
			name:            "異常系 - partnerユーザー更新エラー",
			initiatorUserID: "U-alice",
			partnerUserID:   "U-bob",
			mockSetup: func(m *mocks.MockUserRepository) {
				initiatorUser := &model.User{
					LineID:            "U-alice",
					Name:              "アリス",
					Birthday:          "1990-01-01",
					MatchedWithUserID: null.StringFrom("U-bob"),
				}
				partnerUser := &model.User{
					LineID:            "U-bob",
					Name:              "ボブ",
					Birthday:          "1995-05-05",
					MatchedWithUserID: null.StringFrom("U-alice"),
				}
				m.EXPECT().FindByLineID(mock.Anything, "U-alice").Return(initiatorUser, nil)
				m.EXPECT().FindByLineID(mock.Anything, "U-bob").Return(partnerUser, nil)
				m.EXPECT().
					Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
						return u.LineID == "U-alice"
					})).
					Return(nil)
				m.EXPECT().
					Update(mock.Anything, mock.MatchedBy(func(u *model.User) bool {
						return u.LineID == "U-bob"
					})).
					Return(errors.New("db error"))
			},
			expectedError:    true,
			expectedErrorMsg: "failed to update partner user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := mocks.NewMockUserRepository(t)
			tt.mockSetup(mockUserRepo)

			service := NewMatchingService(mockUserRepo)
			updatedInitiator, updatedPartner, err := service.UnmatchUsers(context.Background(), tt.initiatorUserID, tt.partnerUserID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				assert.Nil(t, updatedInitiator)
				assert.Nil(t, updatedPartner)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updatedInitiator)
				assert.NotNil(t, updatedPartner)
				assert.False(t, updatedInitiator.MatchedWithUserID.Valid)
				assert.False(t, updatedPartner.MatchedWithUserID.Valid)
			}
		})
	}
}
