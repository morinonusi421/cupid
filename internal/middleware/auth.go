package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/morinonusi421/cupid/internal/liff"
	"github.com/morinonusi421/cupid/pkg/httputil"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// AuthMiddleware は LIFF ID Token を検証し、user_id を context に保存するミドルウェア
type AuthMiddleware struct {
	verifier liff.Verifier
}

func NewAuthMiddleware(verifier liff.Verifier) *AuthMiddleware {
	return &AuthMiddleware{
		verifier: verifier,
	}
}

// Authenticate は認証を行うミドルウェア関数
func (m *AuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Authorization ヘッダーからトークン取得
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			httputil.WriteJSONError(w, http.StatusUnauthorized, map[string]string{"error": "認証が必要です"})
			return
		}

		// "Bearer {token}" 形式からトークン抽出
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader { // Bearer プレフィックスがない
			httputil.WriteJSONError(w, http.StatusUnauthorized, map[string]string{"error": "無効な認証形式です"})
			return
		}

		// トークン検証して user_id 取得
		userID, err := m.verifier.VerifyIDToken(token)
		if err != nil {
			log.Printf("Token verification failed: %v", err)
			httputil.WriteJSONError(w, http.StatusUnauthorized, map[string]string{"error": "認証に失敗しました"})
			return
		}

		// user_id を context に保存
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// GetUserIDFromContext は context から user_id を取得する
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
