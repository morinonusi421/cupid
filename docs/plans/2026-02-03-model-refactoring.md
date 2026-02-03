# Model Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Refactor thin model layer by adding domain logic to models, introducing MatchingService, and improving dependency injection

**Architecture:** Follow hybrid approach without Value Objects. Add domain methods to User and Like models. Extract cross-aggregate matching logic to MatchingService. Keep UserService focused on orchestration.

**Tech Stack:** Go 1.25.5, testing/testify, SQLite with SQLBoiler ORM

---

## Task 1: Add User Domain Methods

**Files:**
- Modify: `internal/model/user.go:1-16`
- Create: `internal/model/user_test.go`

**Step 1: Write failing tests for User domain methods**

Create `internal/model/user_test.go`:

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_IsSamePerson(t *testing.T) {
	user := &User{
		Name:     "田中太郎",
		Birthday: "2000-01-15",
	}

	tests := []struct {
		name     string
		testName string
		birthday string
		want     bool
	}{
		{
			name:     "同じ名前と誕生日",
			testName: "田中太郎",
			birthday: "2000-01-15",
			want:     true,
		},
		{
			name:     "異なる名前",
			testName: "佐藤花子",
			birthday: "2000-01-15",
			want:     false,
		},
		{
			name:     "異なる誕生日",
			testName: "田中太郎",
			birthday: "2000-12-25",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := user.IsSamePerson(tt.testName, tt.birthday)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUser_CanRegisterCrush(t *testing.T) {
	tests := []struct {
		name             string
		registrationStep int
		want             bool
	}{
		{
			name:             "Step 0: 未登録",
			registrationStep: 0,
			want:             false,
		},
		{
			name:             "Step 1: ユーザー登録完了",
			registrationStep: 1,
			want:             true,
		},
		{
			name:             "Step 2: Crush登録完了",
			registrationStep: 2,
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{RegistrationStep: tt.registrationStep}
			got := user.CanRegisterCrush()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUser_CompleteCrushRegistration(t *testing.T) {
	user := &User{RegistrationStep: 1}
	user.CompleteCrushRegistration()
	assert.Equal(t, 2, user.RegistrationStep)
}

func TestUser_IsRegistrationComplete(t *testing.T) {
	tests := []struct {
		name             string
		registrationStep int
		want             bool
	}{
		{
			name:             "Step 0: 未登録",
			registrationStep: 0,
			want:             false,
		},
		{
			name:             "Step 1: ユーザー登録完了",
			registrationStep: 1,
			want:             true,
		},
		{
			name:             "Step 2: Crush登録完了",
			registrationStep: 2,
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{RegistrationStep: tt.registrationStep}
			got := user.IsRegistrationComplete()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUser_CompleteUserRegistration(t *testing.T) {
	user := &User{RegistrationStep: 0}
	user.CompleteUserRegistration()
	assert.Equal(t, 1, user.RegistrationStep)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/model -v`
Expected: FAIL with "undefined: User.IsSamePerson" etc.

**Step 3: Implement User domain methods**

Modify `internal/model/user.go`, add after line 15:

```go
// IsSamePerson は自己登録チェック用
func (u *User) IsSamePerson(name, birthday string) bool {
	return u.Name == name && u.Birthday == birthday
}

// CanRegisterCrush はcrush登録可能かチェック
func (u *User) CanRegisterCrush() bool {
	return u.RegistrationStep >= 1
}

// CompleteCrushRegistration はcrush登録完了時の状態遷移
func (u *User) CompleteCrushRegistration() {
	u.RegistrationStep = 2
}

// IsRegistrationComplete はユーザー登録が完了しているかチェック
func (u *User) IsRegistrationComplete() bool {
	return u.RegistrationStep >= 1
}

// CompleteUserRegistration はユーザー登録完了時の状態遷移
func (u *User) CompleteUserRegistration() {
	u.RegistrationStep = 1
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/model -v`
Expected: PASS (all User tests passing)

**Step 5: Commit**

```bash
git add internal/model/user.go internal/model/user_test.go
git commit -m "feat(model): add User domain methods

- Add IsSamePerson() for self-registration check
- Add CanRegisterCrush() for crush registration eligibility
- Add CompleteCrushRegistration() for state transition
- Add IsRegistrationComplete() for registration status
- Add CompleteUserRegistration() for user registration completion
- Add comprehensive tests for all methods"
```

---

## Task 2: Add Like Domain Methods

**Files:**
- Modify: `internal/model/like.go:1-45`
- Create: `internal/model/like_test.go`

**Step 1: Write failing tests for Like domain methods**

Create `internal/model/like_test.go`:

```go
package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLike_MarkAsMatched(t *testing.T) {
	like := &Like{Matched: false}
	like.MarkAsMatched()
	assert.True(t, like.Matched)
}

func TestLike_IsMatched(t *testing.T) {
	tests := []struct {
		name    string
		matched bool
		want    bool
	}{
		{
			name:    "Matched",
			matched: true,
			want:    true,
		},
		{
			name:    "Not matched",
			matched: false,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			like := &Like{Matched: tt.matched}
			got := like.IsMatched()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLike_MatchesCrush(t *testing.T) {
	like := &Like{
		ToName:     "山田花子",
		ToBirthday: "1995-05-20",
	}

	tests := []struct {
		name     string
		testName string
		birthday string
		want     bool
	}{
		{
			name:     "同じ名前と誕生日",
			testName: "山田花子",
			birthday: "1995-05-20",
			want:     true,
		},
		{
			name:     "異なる名前",
			testName: "佐藤花子",
			birthday: "1995-05-20",
			want:     false,
		},
		{
			name:     "異なる誕生日",
			testName: "山田花子",
			birthday: "1995-12-25",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := like.MatchesCrush(tt.testName, tt.birthday)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewLike(t *testing.T) {
	like := NewLike("U123", "田中太郎", "2000-01-15")

	assert.Equal(t, "U123", like.FromUserID)
	assert.Equal(t, "田中太郎", like.ToName)
	assert.Equal(t, "2000-01-15", like.ToBirthday)
	assert.False(t, like.Matched)
	assert.Equal(t, int64(0), like.ID)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/model -v`
Expected: FAIL with "undefined: Like.MarkAsMatched" etc.

**Step 3: Implement Like domain methods**

Modify `internal/model/like.go`, add after line 44:

```go

// MarkAsMatched はマッチング成立時にフラグを立てる
func (l *Like) MarkAsMatched() {
	l.Matched = true
}

// IsMatched はマッチング済みかチェック
func (l *Like) IsMatched() bool {
	return l.Matched
}

// MatchesCrush は指定された名前・誕生日と一致するかチェック
func (l *Like) MatchesCrush(name, birthday string) bool {
	return l.ToName == name && l.ToBirthday == birthday
}

// NewLike はLikeを生成するファクトリ関数
func NewLike(fromUserID, toName, toBirthday string) *Like {
	return &Like{
		FromUserID:  fromUserID,
		ToName:      toName,
		ToBirthday:  toBirthday,
		Matched:     false,
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/model -v`
Expected: PASS (all Like and User tests passing)

**Step 5: Commit**

```bash
git add internal/model/like.go internal/model/like_test.go
git commit -m "feat(model): add Like domain methods

- Add MarkAsMatched() for matching state update
- Add IsMatched() for matching status check
- Add MatchesCrush() for crush matching verification
- Add NewLike() factory function
- Add comprehensive tests for all methods"
```

---

## Task 3: Create MatchingService

**Files:**
- Create: `internal/service/matching_service.go`
- Create: `internal/service/matching_service_test.go`

**Step 1: Write failing test for MatchingService**

Create `internal/service/matching_service_test.go`:

```go
package service

import (
	"context"
	"testing"

	"github.com/morinonusi421/cupid/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMatchingUserRepository is a mock for UserRepository used in MatchingService tests
type MockMatchingUserRepository struct {
	mock.Mock
}

func (m *MockMatchingUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockMatchingUserRepository) FindByLineID(ctx context.Context, lineID string) (*model.User, error) {
	args := m.Called(ctx, lineID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockMatchingUserRepository) FindByNameAndBirthday(ctx context.Context, name, birthday string) (*model.User, error) {
	args := m.Called(ctx, name, birthday)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockMatchingUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// MockMatchingLikeRepository is a mock for LikeRepository used in MatchingService tests
type MockMatchingLikeRepository struct {
	mock.Mock
}

func (m *MockMatchingLikeRepository) Create(ctx context.Context, like *model.Like) error {
	args := m.Called(ctx, like)
	return args.Error(0)
}

func (m *MockMatchingLikeRepository) FindByFromUserID(ctx context.Context, fromUserID string) (*model.Like, error) {
	args := m.Called(ctx, fromUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Like), args.Error(1)
}

func (m *MockMatchingLikeRepository) FindMatchingLike(ctx context.Context, fromUserID, toName, toBirthday string) (*model.Like, error) {
	args := m.Called(ctx, fromUserID, toName, toBirthday)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Like), args.Error(1)
}

func (m *MockMatchingLikeRepository) UpdateMatched(ctx context.Context, likeID int64, matched bool) error {
	args := m.Called(ctx, likeID, matched)
	return args.Error(0)
}

func TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotRegistered(t *testing.T) {
	userRepo := new(MockMatchingUserRepository)
	likeRepo := new(MockMatchingLikeRepository)
	service := NewMatchingService(userRepo, likeRepo)

	currentUser := &model.User{
		LineID:   "U111",
		Name:     "田中太郎",
		Birthday: "2000-01-15",
	}
	currentLike := &model.Like{
		ID:         1,
		FromUserID: "U111",
		ToName:     "山田花子",
		ToBirthday: "1995-05-20",
	}

	// Crush not registered
	userRepo.On("FindByNameAndBirthday", mock.Anything, "山田花子", "1995-05-20").
		Return(nil, nil)

	matched, matchedName, err := service.CheckAndUpdateMatch(context.Background(), currentUser, currentLike)

	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Empty(t, matchedName)
	userRepo.AssertExpectations(t)
	likeRepo.AssertExpectations(t)
}

func TestMatchingService_CheckAndUpdateMatch_NoMatch_CrushNotLikeBack(t *testing.T) {
	userRepo := new(MockMatchingUserRepository)
	likeRepo := new(MockMatchingLikeRepository)
	service := NewMatchingService(userRepo, likeRepo)

	currentUser := &model.User{
		LineID:   "U111",
		Name:     "田中太郎",
		Birthday: "2000-01-15",
	}
	currentLike := &model.Like{
		ID:         1,
		FromUserID: "U111",
		ToName:     "山田花子",
		ToBirthday: "1995-05-20",
	}
	crushUser := &model.User{
		LineID:   "U222",
		Name:     "山田花子",
		Birthday: "1995-05-20",
	}

	// Crush is registered
	userRepo.On("FindByNameAndBirthday", mock.Anything, "山田花子", "1995-05-20").
		Return(crushUser, nil)
	// But crush doesn't like back
	likeRepo.On("FindMatchingLike", mock.Anything, "U222", "田中太郎", "2000-01-15").
		Return(nil, nil)

	matched, matchedName, err := service.CheckAndUpdateMatch(context.Background(), currentUser, currentLike)

	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Empty(t, matchedName)
	userRepo.AssertExpectations(t)
	likeRepo.AssertExpectations(t)
}

func TestMatchingService_CheckAndUpdateMatch_Match(t *testing.T) {
	userRepo := new(MockMatchingUserRepository)
	likeRepo := new(MockMatchingLikeRepository)
	service := NewMatchingService(userRepo, likeRepo)

	currentUser := &model.User{
		LineID:   "U111",
		Name:     "田中太郎",
		Birthday: "2000-01-15",
	}
	currentLike := &model.Like{
		ID:         1,
		FromUserID: "U111",
		ToName:     "山田花子",
		ToBirthday: "1995-05-20",
	}
	crushUser := &model.User{
		LineID:   "U222",
		Name:     "山田花子",
		Birthday: "1995-05-20",
	}
	reverseLike := &model.Like{
		ID:         2,
		FromUserID: "U222",
		ToName:     "田中太郎",
		ToBirthday: "2000-01-15",
	}

	// Crush is registered
	userRepo.On("FindByNameAndBirthday", mock.Anything, "山田花子", "1995-05-20").
		Return(crushUser, nil)
	// Crush likes back
	likeRepo.On("FindMatchingLike", mock.Anything, "U222", "田中太郎", "2000-01-15").
		Return(reverseLike, nil)
	// Update both matched flags
	likeRepo.On("UpdateMatched", mock.Anything, int64(1), true).Return(nil)
	likeRepo.On("UpdateMatched", mock.Anything, int64(2), true).Return(nil)

	matched, matchedName, err := service.CheckAndUpdateMatch(context.Background(), currentUser, currentLike)

	assert.NoError(t, err)
	assert.True(t, matched)
	assert.Equal(t, "山田花子", matchedName)
	userRepo.AssertExpectations(t)
	likeRepo.AssertExpectations(t)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestMatchingService -v`
Expected: FAIL with "undefined: NewMatchingService"

**Step 3: Implement MatchingService**

Create `internal/service/matching_service.go`:

```go
package service

import (
	"context"
	"fmt"

	"github.com/morinonusi421/cupid/internal/model"
	"github.com/morinonusi421/cupid/internal/repository"
)

// MatchingService はマッチング判定ロジックを提供する
type MatchingService struct {
	userRepo repository.UserRepository
	likeRepo repository.LikeRepository
}

// NewMatchingService はMatchingServiceの新しいインスタンスを作成
func NewMatchingService(
	userRepo repository.UserRepository,
	likeRepo repository.LikeRepository,
) *MatchingService {
	return &MatchingService{
		userRepo: userRepo,
		likeRepo: likeRepo,
	}
}

// CheckAndUpdateMatch はマッチング判定と状態更新を行う
//
// 処理フロー:
// 1. crushがusersテーブルに存在するか確認
// 2. crushも自分を登録しているか確認（双方向チェック）
// 3. マッチング成立なら両方のmatchedフラグを更新
//
// 戻り値:
// - matched: マッチング成立したか
// - matchedUserName: マッチングした相手の名前
// - error: エラー
func (m *MatchingService) CheckAndUpdateMatch(
	ctx context.Context,
	currentUser *model.User,
	currentLike *model.Like,
) (matched bool, matchedUserName string, err error) {

	// 1. crushがusersテーブルに存在するか確認
	crushUser, err := m.userRepo.FindByNameAndBirthday(
		ctx,
		currentLike.ToName,
		currentLike.ToBirthday,
	)
	if err != nil {
		return false, "", fmt.Errorf("failed to find crush user: %w", err)
	}
	if crushUser == nil {
		// 相手が未登録 → マッチング不可
		return false, "", nil
	}

	// 2. crushも自分を登録しているか確認（双方向チェック）
	reverseLike, err := m.likeRepo.FindMatchingLike(
		ctx,
		crushUser.LineID,
		currentUser.Name,
		currentUser.Birthday,
	)
	if err != nil {
		return false, "", fmt.Errorf("failed to find reverse like: %w", err)
	}
	if reverseLike == nil {
		// 相手は自分を登録していない → マッチング不可
		return false, "", nil
	}

	// 3. マッチング成立！両方のmatchedフラグを更新
	currentLike.MarkAsMatched()
	reverseLike.MarkAsMatched()

	if err := m.likeRepo.UpdateMatched(ctx, currentLike.ID, true); err != nil {
		return false, "", fmt.Errorf("failed to update current like: %w", err)
	}
	if err := m.likeRepo.UpdateMatched(ctx, reverseLike.ID, true); err != nil {
		return false, "", fmt.Errorf("failed to update reverse like: %w", err)
	}

	return true, currentLike.ToName, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestMatchingService -v`
Expected: PASS (all MatchingService tests passing)

**Step 5: Commit**

```bash
git add internal/service/matching_service.go internal/service/matching_service_test.go
git commit -m "feat(service): add MatchingService

- Extract matching logic from UserService
- Implement CheckAndUpdateMatch() for dual-check matching
- Add comprehensive tests (no match scenarios, match scenario)
- Improve testability by separating concerns"
```

---

## Task 4: Refactor UserService to use Model methods and MatchingService

**Files:**
- Modify: `internal/service/user_service.go:23-231`
- Modify: `internal/service/user_service_test.go`

**Step 1: Update UserService tests to use MatchingService mock**

Modify `internal/service/user_service_test.go`:

Add after existing mock definitions:

```go
// MockMatchingService is a mock for MatchingService
type MockMatchingService struct {
	mock.Mock
}

func (m *MockMatchingService) CheckAndUpdateMatch(ctx context.Context, currentUser *model.User, currentLike *model.Like) (matched bool, matchedUserName string, err error) {
	args := m.Called(ctx, currentUser, currentLike)
	return args.Bool(0), args.String(1), args.Error(2)
}
```

Find `TestUserService_RegisterFromLIFF` test and modify it to use `CompleteUserRegistration()`:

Replace line that sets `user.RegistrationStep = 1` in the mock expectation with verification that `CompleteUserRegistration()` was called:

```go
func TestUserService_RegisterFromLIFF(t *testing.T) {
	// ... existing setup ...

	userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.LineID == "U123" &&
			u.Name == "テスト太郎" &&
			u.Birthday == "2000-01-15" &&
			u.RegistrationStep == 1 // Verify CompleteUserRegistration() was called
	})).Return(nil)

	// ... rest of test ...
}
```

Find `TestUserService_RegisterCrush` tests and refactor them to use MatchingService mock:

Replace the entire `TestUserService_RegisterCrush_Match` test with:

```go
func TestUserService_RegisterCrush_Match(t *testing.T) {
	userRepo := new(MockUserRepository)
	likeRepo := new(MockLikeRepository)
	matchingService := new(MockMatchingService)
	service := NewUserService(userRepo, likeRepo, matchingService, nil, "")

	currentUser := &model.User{
		LineID:           "U111",
		Name:             "田中太郎",
		Birthday:         "2000-01-15",
		RegistrationStep: 1,
	}

	userRepo.On("FindByLineID", mock.Anything, "U111").Return(currentUser, nil)
	likeRepo.On("Create", mock.Anything, mock.MatchedBy(func(l *model.Like) bool {
		return l.FromUserID == "U111" &&
			l.ToName == "山田花子" &&
			l.ToBirthday == "1995-05-20" &&
			!l.Matched
	})).Return(nil)
	userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.RegistrationStep == 2 // Verify CompleteCrushRegistration() was called
	})).Return(nil)
	matchingService.On("CheckAndUpdateMatch", mock.Anything, currentUser, mock.AnythingOfType("*model.Like")).
		Return(true, "山田花子", nil)

	matched, matchedUserName, err := service.RegisterCrush(context.Background(), "U111", "山田花子", "1995-05-20")

	assert.NoError(t, err)
	assert.True(t, matched)
	assert.Equal(t, "山田花子", matchedUserName)
	userRepo.AssertExpectations(t)
	likeRepo.AssertExpectations(t)
	matchingService.AssertExpectations(t)
}
```

Replace the entire `TestUserService_RegisterCrush_NoMatch` test with:

```go
func TestUserService_RegisterCrush_NoMatch(t *testing.T) {
	userRepo := new(MockUserRepository)
	likeRepo := new(MockLikeRepository)
	matchingService := new(MockMatchingService)
	service := NewUserService(userRepo, likeRepo, matchingService, nil, "")

	currentUser := &model.User{
		LineID:           "U111",
		Name:             "田中太郎",
		Birthday:         "2000-01-15",
		RegistrationStep: 1,
	}

	userRepo.On("FindByLineID", mock.Anything, "U111").Return(currentUser, nil)
	likeRepo.On("Create", mock.Anything, mock.MatchedBy(func(l *model.Like) bool {
		return l.FromUserID == "U111" &&
			l.ToName == "山田花子" &&
			l.ToBirthday == "1995-05-20" &&
			!l.Matched
	})).Return(nil)
	userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		return u.RegistrationStep == 2
	})).Return(nil)
	matchingService.On("CheckAndUpdateMatch", mock.Anything, currentUser, mock.AnythingOfType("*model.Like")).
		Return(false, "", nil)

	matched, matchedUserName, err := service.RegisterCrush(context.Background(), "U111", "山田花子", "1995-05-20")

	assert.NoError(t, err)
	assert.False(t, matched)
	assert.Empty(t, matchedUserName)
	userRepo.AssertExpectations(t)
	likeRepo.AssertExpectations(t)
	matchingService.AssertExpectations(t)
}
```

Add new test for self-registration check using `IsSamePerson()`:

```go
func TestUserService_RegisterCrush_SelfRegistration(t *testing.T) {
	userRepo := new(MockUserRepository)
	likeRepo := new(MockLikeRepository)
	matchingService := new(MockMatchingService)
	service := NewUserService(userRepo, likeRepo, matchingService, nil, "")

	currentUser := &model.User{
		LineID:           "U111",
		Name:             "田中太郎",
		Birthday:         "2000-01-15",
		RegistrationStep: 1,
	}

	userRepo.On("FindByLineID", mock.Anything, "U111").Return(currentUser, nil)

	matched, matchedUserName, err := service.RegisterCrush(context.Background(), "U111", "田中太郎", "2000-01-15")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot register yourself")
	assert.False(t, matched)
	assert.Empty(t, matchedUserName)
	userRepo.AssertExpectations(t)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/service -v`
Expected: FAIL with compilation errors about NewUserService signature and missing MatchingService

**Step 3: Refactor UserService implementation**

Modify `internal/service/user_service.go`:

Update userService struct (line 23-28):

```go
type userService struct {
	userRepo        repository.UserRepository
	likeRepo        repository.LikeRepository
	matchingService *MatchingService
	liffVerifier    *liff.Verifier
	liffRegisterURL string
}
```

Update NewUserService (line 30-38):

```go
func NewUserService(
	userRepo repository.UserRepository,
	likeRepo repository.LikeRepository,
	matchingService *MatchingService,
	liffVerifier *liff.Verifier,
	liffRegisterURL string,
) UserService {
	return &userService{
		userRepo:        userRepo,
		likeRepo:        likeRepo,
		matchingService: matchingService,
		liffVerifier:    liffVerifier,
		liffRegisterURL: liffRegisterURL,
	}
}
```

Update RegisterFromLIFF (line 138-156) to use model methods:

```go
func (s *userService) RegisterFromLIFF(ctx context.Context, userID, name, birthday string) error {
	// Get or create user
	user, err := s.GetOrCreateUser(ctx, userID, "")
	if err != nil {
		return fmt.Errorf("failed to get or create user: %w", err)
	}

	// Update user info
	user.Name = name
	user.Birthday = birthday
	user.CompleteUserRegistration() // Use model method

	if err := s.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}
```

Update RegisterCrush (line 158-231) to use model methods and MatchingService:

```go
func (s *userService) RegisterCrush(ctx context.Context, userID, crushName, crushBirthday string) (matched bool, matchedUserName string, err error) {
	// 1. ユーザー取得
	currentUser, err := s.userRepo.FindByLineID(ctx, userID)
	if err != nil {
		return false, "", err
	}
	if currentUser == nil {
		return false, "", fmt.Errorf("user not found: %s", userID)
	}

	// 2. 自己登録チェック（Modelメソッド使用）
	if currentUser.IsSamePerson(crushName, crushBirthday) {
		return false, "", fmt.Errorf("cannot register yourself")
	}

	// 3. Like登録（ファクトリ関数使用）
	like := model.NewLike(userID, crushName, crushBirthday)
	if err := s.likeRepo.Create(ctx, like); err != nil {
		return false, "", err
	}

	// 4. 状態遷移（Modelメソッド使用）
	currentUser.CompleteCrushRegistration()
	if err := s.userRepo.Update(ctx, currentUser); err != nil {
		return false, "", err
	}

	// 5. マッチング判定（MatchingServiceに委譲）
	matched, name, err := s.matchingService.CheckAndUpdateMatch(ctx, currentUser, like)
	if err != nil {
		return false, "", err
	}

	return matched, name, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/service -v`
Expected: PASS (all service tests passing)

**Step 5: Commit**

```bash
git add internal/service/user_service.go internal/service/user_service_test.go
git commit -m "refactor(service): refactor UserService to use model methods and MatchingService

- Inject MatchingService into UserService
- Use User.IsSamePerson() for self-registration check
- Use User.CompleteUserRegistration() in RegisterFromLIFF
- Use User.CompleteCrushRegistration() in RegisterCrush
- Use model.NewLike() factory function
- Delegate matching logic to MatchingService
- Update tests to mock MatchingService
- Reduce RegisterCrush from 80 lines to 30 lines"
```

---

## Task 5: Update main.go DI

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: Update main.go to inject MatchingService**

Find the service initialization section in `cmd/server/main.go` and modify:

Before (approximately line 80-90):

```go
// Service層
userService := service.NewUserService(userRepo, likeRepo, liffVerifier, registerURL)
```

After:

```go
// Service層
matchingService := service.NewMatchingService(userRepo, likeRepo)
userService := service.NewUserService(userRepo, likeRepo, matchingService, liffVerifier, registerURL)
```

**Step 2: Build and verify compilation**

Run: `go build -o cupid cmd/server/main.go`
Expected: Successful compilation with no errors

**Step 3: Run all tests**

Run: `make test`
Expected: All tests passing

**Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "refactor(di): update main.go to inject MatchingService

- Create MatchingService instance
- Inject MatchingService into UserService
- Complete dependency injection refactoring"
```

---

## Task 6: Run full test suite and verify

**Files:**
- None (verification only)

**Step 1: Run all tests**

Run: `make test`
Expected: All tests in all packages passing

**Step 2: Verify test coverage**

Run: `go test -cover ./...`
Expected: Good coverage for model and service packages

**Step 3: Final verification commit**

```bash
git add -A
git commit -m "test: verify all tests pass after refactoring" --allow-empty
```

---

## Summary

This refactoring achieves:

1. **Model Layer Enrichment**: Added domain methods to User and Like models
2. **MatchingService Introduction**: Extracted cross-aggregate matching logic
3. **UserService Simplification**: Reduced RegisterCrush from 80 lines to 30 lines
4. **Improved Testability**: Model methods and MatchingService can be tested independently
5. **Better DI**: MatchingService properly injected through constructor

All changes maintain backward compatibility and follow TDD principles with comprehensive test coverage.
