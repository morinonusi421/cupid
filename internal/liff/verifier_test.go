package liff

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifier_VerifyAccessToken_ValidToken(t *testing.T) {
	accessToken := os.Getenv("LIFF_ACCESS_TOKEN")
	channelID := os.Getenv("LIFF_CHANNEL_ID")

	if accessToken == "" || channelID == "" {
		t.Skip("Skipping: LIFF_ACCESS_TOKEN and LIFF_CHANNEL_ID env vars not set")
	}

	verifier := NewVerifier(channelID)

	userID, err := verifier.VerifyAccessToken(accessToken)

	assert.NoError(t, err)
	assert.NotEmpty(t, userID)
}

func TestVerifier_VerifyAccessToken_InvalidToken(t *testing.T) {
	channelID := os.Getenv("LIFF_CHANNEL_ID")

	if channelID == "" {
		t.Skip("Skipping: LIFF_CHANNEL_ID env var not set")
	}

	verifier := NewVerifier(channelID)

	userID, err := verifier.VerifyAccessToken("invalid-token-12345")

	assert.Error(t, err)
	assert.Empty(t, userID)
}
