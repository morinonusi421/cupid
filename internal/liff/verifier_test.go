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
