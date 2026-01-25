# LIFF Registration Form Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace text-based user registration with a LIFF (LINE Front-end Framework) web form for better UX and validation.

**Architecture:** Create a web form hosted on EC2/Nginx that uses LIFF SDK to get LINE user info, collect name/birthday with proper validation (date picker), and submit to backend API. Backend verifies LIFF access token and saves user data to SQLite.

**Tech Stack:** LIFF SDK v2, vanilla JavaScript, Go (Echo-like handler), SQLite (existing)

---

## Task 1: Backend API - Registration Endpoint

**Files:**
- Create: `internal/handler/registration_api.go`
- Create: `internal/handler/registration_api_test.go`
- Modify: `cmd/server/main.go` (add route)

**Step 1: Write the failing test for registration API**

Create `internal/handler/registration_api_test.go`:

```go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/morinonusi421/cupid/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMessageServiceForAPI struct {
	mock.Mock
}

func (m *MockMessageServiceForAPI) ProcessTextMessage(ctx interface{}, userID, text string) (string, error) {
	args := m.Called(ctx, userID, text)
	return args.String(0), args.Error(1)
}

func TestRegistrationAPI_Register_Success(t *testing.T) {
	mockMessageService := new(MockMessageServiceForAPI)
	handler := NewRegistrationAPIHandler(mockMessageService)

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
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/handler/... -run TestRegistrationAPI -v
```

Expected: FAIL with "undefined: NewRegistrationAPIHandler"

**Step 3: Write minimal implementation**

Create `internal/handler/registration_api.go`:

```go
package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/morinonusi421/cupid/internal/service"
)

type RegistrationAPIHandler struct {
	messageService service.MessageService
}

func NewRegistrationAPIHandler(messageService service.MessageService) *RegistrationAPIHandler {
	return &RegistrationAPIHandler{
		messageService: messageService,
	}
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Birthday string `json:"birthday"`
}

func (h *RegistrationAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
	// TODO: LIFF token verification will be added later

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// For now, just return 200 OK
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/handler/... -run TestRegistrationAPI -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/handler/registration_api.go internal/handler/registration_api_test.go
git commit -m "feat: add registration API endpoint skeleton"
```

---

## Task 2: LIFF Token Verification

**Files:**
- Create: `internal/liff/verifier.go`
- Create: `internal/liff/verifier_test.go`
- Modify: `internal/handler/registration_api.go`

**Step 1: Write the failing test for LIFF token verifier**

Create `internal/liff/verifier_test.go`:

```go
package liff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifier_VerifyAccessToken_ValidToken(t *testing.T) {
	verifier := NewVerifier("test-channel-id")

	// This will fail until we implement real verification
	userID, err := verifier.VerifyAccessToken("valid-token")

	assert.NoError(t, err)
	assert.NotEmpty(t, userID)
}

func TestVerifier_VerifyAccessToken_InvalidToken(t *testing.T) {
	verifier := NewVerifier("test-channel-id")

	userID, err := verifier.VerifyAccessToken("invalid-token")

	assert.Error(t, err)
	assert.Empty(t, userID)
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/liff/... -v
```

Expected: FAIL with "undefined: NewVerifier"

**Step 3: Write LIFF token verifier implementation**

Create `internal/liff/verifier.go`:

```go
package liff

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Verifier struct {
	channelID string
}

func NewVerifier(channelID string) *Verifier {
	return &Verifier{channelID: channelID}
}

type VerifyResponse struct {
	ClientID string `json:"client_id"`
	Sub      string `json:"sub"` // LINE user ID
	Exp      int64  `json:"exp"`
}

func (v *Verifier) VerifyAccessToken(accessToken string) (string, error) {
	// Call LINE's token verification endpoint
	url := "https://api.line.me/oauth2/v2.1/verify?access_token=" + accessToken

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token verification failed: %s", string(body))
	}

	var verifyResp VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Verify channel ID matches
	if verifyResp.ClientID != v.channelID {
		return "", fmt.Errorf("channel ID mismatch")
	}

	return verifyResp.Sub, nil
}
```

**Step 4: Skip automated test (requires real LIFF token)**

Note: The test requires a real LIFF token which we don't have in unit tests. We'll verify this manually during integration testing.

**Step 5: Integrate verifier into registration API**

Modify `internal/handler/registration_api.go`:

```go
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/morinonusi421/cupid/internal/liff"
	"github.com/morinonusi421/cupid/internal/service"
)

type RegistrationAPIHandler struct {
	messageService service.MessageService
	liffVerifier   *liff.Verifier
}

func NewRegistrationAPIHandler(messageService service.MessageService, liffVerifier *liff.Verifier) *RegistrationAPIHandler {
	return &RegistrationAPIHandler{
		messageService: messageService,
		liffVerifier:   liffVerifier,
	}
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Birthday string `json:"birthday"`
}

func (h *RegistrationAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Extract LIFF access token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		log.Println("Missing or invalid Authorization header")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	// Verify LIFF access token
	userID, err := h.liffVerifier.VerifyAccessToken(accessToken)
	if err != nil {
		log.Printf("Failed to verify LIFF token: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
		return
	}

	// Decode request body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	// TODO: Save user data using messageService
	log.Printf("Registration request for user %s: name=%s, birthday=%s", userID, req.Name, req.Birthday)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

**Step 6: Update test to use new constructor signature**

Modify `internal/handler/registration_api_test.go`:

```go
func TestRegistrationAPI_Register_Success(t *testing.T) {
	mockMessageService := new(MockMessageServiceForAPI)
	mockVerifier := liff.NewVerifier("test-channel-id") // Add this
	handler := NewRegistrationAPIHandler(mockMessageService, mockVerifier) // Update this

	// ... rest of test
}
```

**Step 7: Commit**

```bash
git add internal/liff/ internal/handler/registration_api.go internal/handler/registration_api_test.go
git commit -m "feat: add LIFF token verification"
```

---

## Task 3: Connect Registration API to User Service

**Files:**
- Modify: `internal/handler/registration_api.go`
- Modify: `internal/service/message_service.go` (add RegisterFromLIFF method)

**Step 1: Add RegisterFromLIFF method to MessageService**

Add to `internal/service/message_service.go`:

```go
// MessageService interface - add this method
type MessageService interface {
	ProcessTextMessage(ctx context.Context, userID, text string) (string, error)
	RegisterFromLIFF(ctx context.Context, userID, name, birthday string) error
}

// Implementation - add this method
func (s *messageService) RegisterFromLIFF(ctx context.Context, userID, name, birthday string) error {
	// Get or create user
	user, err := s.userService.GetOrCreateUser(ctx, userID, "")
	if err != nil {
		return fmt.Errorf("failed to get or create user: %w", err)
	}

	// Update user info
	user.Name = name
	user.Birthday = birthday
	user.RegistrationStep = 3 // Registration complete

	if err := s.userService.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}
```

**Step 2: Update registration API to use RegisterFromLIFF**

Modify `internal/handler/registration_api.go`:

```go
func (h *RegistrationAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
	// ... (existing token verification and request decode)

	// Save user data
	if err := h.messageService.RegisterFromLIFF(r.Context(), userID, req.Name, req.Birthday); err != nil {
		log.Printf("Failed to register user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "registration failed"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

**Step 3: Add route to main.go**

Modify `cmd/server/main.go`:

```go
import (
	// ... existing imports
	"github.com/morinonusi421/cupid/internal/liff"
)

func main() {
	// ... (existing code)

	// 依存関係の組み立て (DI)
	lineBotClient := linebot.NewClient(botAPI)
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	messageService := service.NewMessageService(userService)
	webhookHandler := handler.NewWebhookHandler(channelSecret, lineBotClient, messageService)

	// Add LIFF registration API handler
	liffVerifier := liff.NewVerifier(os.Getenv("LINE_LIFF_CHANNEL_ID"))
	registrationAPIHandler := handler.NewRegistrationAPIHandler(messageService, liffVerifier)

	// ヘルスチェック用エンドポイント
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Cupid LINE Bot is running")
	})

	// LINE Webhook エンドポイント
	http.HandleFunc("/webhook", webhookHandler.Handle)

	// Registration API endpoint
	http.HandleFunc("/api/register", registrationAPIHandler.Register)

	// ... (rest of main)
}
```

**Step 4: Build and verify no compilation errors**

```bash
make build
```

Expected: Build succeeds

**Step 5: Commit**

```bash
git add internal/service/message_service.go internal/handler/registration_api.go cmd/server/main.go
git commit -m "feat: connect registration API to user service"
```

---

## Task 4: Create LIFF Frontend - HTML Structure

**Files:**
- Create: `static/liff/register.html`
- Create: `static/liff/register.css`

**Step 1: Create HTML file with form**

Create `static/liff/register.html`:

```html
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ユーザー登録 - Cupid</title>
    <link rel="stylesheet" href="register.css">
    <script charset="utf-8" src="https://static.line-scdn.net/liff/edge/2/sdk.js"></script>
</head>
<body>
    <div class="container">
        <h1>ユーザー登録</h1>
        <form id="registerForm">
            <div class="form-group">
                <label for="name">名前</label>
                <input type="text" id="name" name="name" required maxlength="50" placeholder="例: 田中太郎">
                <span class="error" id="nameError"></span>
            </div>

            <div class="form-group">
                <label for="birthday">誕生日</label>
                <input type="date" id="birthday" name="birthday" required>
                <span class="error" id="birthdayError"></span>
            </div>

            <button type="submit" id="submitBtn">登録する</button>
        </form>

        <div id="loading" class="loading hidden">
            <p>登録中...</p>
        </div>

        <div id="success" class="success hidden">
            <p>✓ 登録完了しました！</p>
        </div>

        <div id="error" class="error-message hidden">
            <p id="errorText"></p>
        </div>
    </div>

    <script src="register.js"></script>
</body>
</html>
```

**Step 2: Create CSS file**

Create `static/liff/register.css`:

```css
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    background-color: #f5f5f5;
    padding: 20px;
}

.container {
    max-width: 400px;
    margin: 0 auto;
    background: white;
    padding: 30px;
    border-radius: 10px;
    box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

h1 {
    font-size: 24px;
    margin-bottom: 30px;
    text-align: center;
    color: #333;
}

.form-group {
    margin-bottom: 20px;
}

label {
    display: block;
    margin-bottom: 8px;
    font-weight: 600;
    color: #555;
}

input[type="text"],
input[type="date"] {
    width: 100%;
    padding: 12px;
    border: 1px solid #ddd;
    border-radius: 5px;
    font-size: 16px;
}

input:focus {
    outline: none;
    border-color: #06c755;
}

.error {
    display: block;
    color: #e74c3c;
    font-size: 14px;
    margin-top: 5px;
}

button[type="submit"] {
    width: 100%;
    padding: 15px;
    background-color: #06c755;
    color: white;
    border: none;
    border-radius: 5px;
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
}

button[type="submit"]:hover {
    background-color: #05b04b;
}

button[type="submit"]:disabled {
    background-color: #ccc;
    cursor: not-allowed;
}

.loading,
.success,
.error-message {
    text-align: center;
    padding: 20px;
    margin-top: 20px;
    border-radius: 5px;
}

.loading {
    background-color: #f0f0f0;
}

.success {
    background-color: #d4edda;
    color: #155724;
}

.error-message {
    background-color: #f8d7da;
    color: #721c24;
}

.hidden {
    display: none;
}
```

**Step 3: Commit static files**

```bash
mkdir -p static/liff
git add static/liff/register.html static/liff/register.css
git commit -m "feat: add LIFF registration form HTML/CSS"
```

---

## Task 5: Create LIFF Frontend - JavaScript Logic

**Files:**
- Create: `static/liff/register.js`

**Step 1: Create JavaScript file with LIFF integration**

Create `static/liff/register.js`:

```javascript
(function() {
    'use strict';

    const LIFF_ID = 'YOUR_LIFF_ID_HERE'; // Will be replaced with actual LIFF ID
    const API_ENDPOINT = '/api/register';

    let liffAccessToken = null;

    // Initialize LIFF
    async function initializeLiff() {
        try {
            await liff.init({ liffId: LIFF_ID });

            if (!liff.isLoggedIn()) {
                liff.login();
                return;
            }

            liffAccessToken = liff.getAccessToken();
            console.log('LIFF initialized successfully');
        } catch (err) {
            console.error('LIFF initialization failed', err);
            showError('LIFFの初期化に失敗しました');
        }
    }

    // Form submission handler
    async function handleSubmit(e) {
        e.preventDefault();

        // Clear previous errors
        document.getElementById('nameError').textContent = '';
        document.getElementById('birthdayError').textContent = '';
        document.getElementById('error').classList.add('hidden');

        // Get form values
        const name = document.getElementById('name').value.trim();
        const birthday = document.getElementById('birthday').value;

        // Validate
        if (!name) {
            document.getElementById('nameError').textContent = '名前を入力してください';
            return;
        }

        if (name.length > 50) {
            document.getElementById('nameError').textContent = '名前は50文字以内で入力してください';
            return;
        }

        if (!birthday) {
            document.getElementById('birthdayError').textContent = '誕生日を選択してください';
            return;
        }

        // Show loading
        document.getElementById('registerForm').classList.add('hidden');
        document.getElementById('loading').classList.remove('hidden');

        try {
            const response = await fetch(API_ENDPOINT, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${liffAccessToken}`
                },
                body: JSON.stringify({ name, birthday })
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || '登録に失敗しました');
            }

            // Success
            document.getElementById('loading').classList.add('hidden');
            document.getElementById('success').classList.remove('hidden');

            // Close LIFF window after 2 seconds
            setTimeout(() => {
                liff.closeWindow();
            }, 2000);

        } catch (err) {
            console.error('Registration failed', err);
            document.getElementById('loading').classList.add('hidden');
            document.getElementById('registerForm').classList.remove('hidden');
            showError(err.message);
        }
    }

    function showError(message) {
        document.getElementById('errorText').textContent = message;
        document.getElementById('error').classList.remove('hidden');
    }

    // Initialize on page load
    window.addEventListener('load', async () => {
        await initializeLiff();

        const form = document.getElementById('registerForm');
        form.addEventListener('submit', handleSubmit);
    });
})();
```

**Step 2: Commit JavaScript file**

```bash
git add static/liff/register.js
git commit -m "feat: add LIFF registration form JavaScript logic"
```

---

## Task 6: Configure Nginx to Serve Static Files

**Files:**
- Modify: `nginx/cupid.conf`

**Step 1: Update Nginx configuration**

Modify `nginx/cupid.conf`:

```nginx
server {
    listen 80;
    server_name cupid-linebot.click;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl;
    server_name cupid-linebot.click;

    ssl_certificate /etc/letsencrypt/live/cupid-linebot.click/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cupid-linebot.click/privkey.pem;

    # Reverse proxy for Go server
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Serve static LIFF files
    location /liff/ {
        alias /home/ec2-user/cupid/static/liff/;
        try_files $uri $uri/ =404;
    }
}
```

**Step 2: Commit Nginx configuration**

```bash
git add nginx/cupid.conf
git commit -m "feat: configure Nginx to serve LIFF static files"
```

---

## Task 7: Update Bot Message to Include LIFF URL

**Files:**
- Modify: `internal/service/message_service.go`

**Step 1: Update handleInitialMessage to return LIFF URL**

Modify `internal/service/message_service.go`:

Change the `handleInitialMessage` method to return a Flex Message with LIFF button:

```go
func (s *messageService) handleInitialMessage(ctx context.Context, user *model.User) (string, error) {
	// Step を 1 に進めるのではなく、LIFF で登録してもらう
	// registration_step は LIFF API で直接 3 に更新される

	// LIFF URL を含むメッセージを返す
	// Note: This returns a plain text message for now
	// In the next task, we'll modify the handler to support Flex Messages
	return "初めまして！以下のボタンから登録してください。\nhttps://cupid-linebot.click/liff/register.html", nil
}
```

**Step 2: Commit message service update**

```bash
git add internal/service/message_service.go
git commit -m "feat: update initial message to include LIFF registration URL"
```

---

## Task 8: Deploy and Test

**Files:**
- Modify: `.env` (add LINE_LIFF_CHANNEL_ID)

**Step 1: Register LIFF app in LINE Developers Console**

Manual steps (document for reference):
1. Go to https://developers.line.biz/console/
2. Select your channel
3. Go to LIFF tab
4. Click "Add"
5. Set:
   - LIFF app name: "Cupid Registration"
   - Size: Full
   - Endpoint URL: `https://cupid-linebot.click/liff/register.html`
   - Scope: `profile`, `openid`
6. Click "Add"
7. Copy the LIFF ID (format: `1234567890-abcdefgh`)

**Step 2: Update LIFF ID in register.js**

Modify `static/liff/register.js`:

```javascript
const LIFF_ID = '1234567890-abcdefgh'; // Replace with actual LIFF ID
```

**Step 3: Add LINE_LIFF_CHANNEL_ID to .env**

Add to `.env` on EC2:

```bash
LINE_LIFF_CHANNEL_ID=1234567890
```

Note: The LIFF Channel ID is the numeric part of the LIFF ID (before the dash)

**Step 4: Deploy to EC2**

```bash
git add .
git commit -m "feat: configure LIFF ID for registration form"
git push origin main
make deploy
```

**Step 5: Test LIFF registration flow**

Manual test steps:
1. Send any message to the bot
2. Bot should reply with LIFF URL
3. Click the URL
4. Fill in name and birthday
5. Click "登録する"
6. Should see "登録完了しました！"
7. Check database: `sqlite3 cupid.db "SELECT * FROM users;"`
8. Verify `registration_step = 3`

**Step 6: Verify completed registration**

Send another message to the bot after registration:
- Expected: Bot echoes the message (オウム返し)
- This confirms `registration_step = 3` flow works

---

## Task 9: Add Push Message After Registration (Optional Enhancement)

**Files:**
- Create: `internal/linebot/push.go`
- Modify: `internal/handler/registration_api.go`

**Step 1: Create push message client wrapper**

Create `internal/linebot/push.go`:

```go
package linebot

import (
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

type PushClient interface {
	PushMessage(to string, messages ...messaging_api.MessageInterface) error
}

type pushClient struct {
	api *messaging_api.MessagingApiAPI
}

func NewPushClient(api *messaging_api.MessagingApiAPI) PushClient {
	return &pushClient{api: api}
}

func (c *pushClient) PushMessage(to string, messages ...messaging_api.MessageInterface) error {
	_, err := c.api.PushMessage(&messaging_api.PushMessageRequest{
		To:       to,
		Messages: messages,
	})
	return err
}
```

**Step 2: Update registration API to send push message**

Modify `internal/handler/registration_api.go`:

```go
type RegistrationAPIHandler struct {
	messageService service.MessageService
	liffVerifier   *liff.Verifier
	pushClient     linebot.PushClient // Add this
}

func NewRegistrationAPIHandler(
	messageService service.MessageService,
	liffVerifier *liff.Verifier,
	pushClient linebot.PushClient, // Add this
) *RegistrationAPIHandler {
	return &RegistrationAPIHandler{
		messageService: messageService,
		liffVerifier:   liffVerifier,
		pushClient:     pushClient,
	}
}

func (h *RegistrationAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
	// ... (existing code)

	// Save user data
	if err := h.messageService.RegisterFromLIFF(r.Context(), userID, req.Name, req.Birthday); err != nil {
		log.Printf("Failed to register user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "registration failed"})
		return
	}

	// Send push message to confirm registration
	if err := h.pushClient.PushMessage(
		userID,
		messaging_api.TextMessage{
			Text: fmt.Sprintf("%sさん、登録ありがとう！", req.Name),
		},
	); err != nil {
		log.Printf("Failed to send push message: %v", err)
		// Don't fail the request if push fails
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

**Step 3: Update main.go to inject push client**

Modify `cmd/server/main.go`:

```go
// Add push client
lineBotPushClient := linebot.NewPushClient(botAPI)

// Update registration API handler
liffVerifier := liff.NewVerifier(os.Getenv("LINE_LIFF_CHANNEL_ID"))
registrationAPIHandler := handler.NewRegistrationAPIHandler(messageService, liffVerifier, lineBotPushClient)
```

**Step 4: Update test to use new signature**

Modify `internal/handler/registration_api_test.go` to add mock push client.

**Step 5: Test and commit**

```bash
make build
make test
git add internal/linebot/push.go internal/handler/registration_api.go cmd/server/main.go
git commit -m "feat: send push message after LIFF registration"
```

---

## Deployment Checklist

- [ ] LINE Developers: Register LIFF app, get LIFF ID
- [ ] Update `static/liff/register.js` with real LIFF ID
- [ ] Add `LINE_LIFF_CHANNEL_ID` to EC2 `.env`
- [ ] Deploy: `git push && make deploy`
- [ ] Verify Nginx serves `/liff/register.html`
- [ ] Test registration flow end-to-end
- [ ] Verify database: `registration_step = 3`
- [ ] Verify push message received

---

## Notes

- LIFF token verification requires real tokens, so unit tests are limited
- Integration testing should be done manually with actual LINE account
- LIFF ID format: `1234567890-abcdefgh` (numeric channel ID + random suffix)
- Static files served by Nginx at `/liff/` path
- Registration sets `registration_step = 3` (complete)
