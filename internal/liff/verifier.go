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

	// Debug logging
	fmt.Printf("DEBUG: VerifyResponse: ClientID=%s, Sub=%s, Exp=%d, ExpectedChannelID=%s\n",
		verifyResp.ClientID, verifyResp.Sub, verifyResp.Exp, v.channelID)

	// Verify channel ID matches
	if verifyResp.ClientID != v.channelID {
		return "", fmt.Errorf("channel ID mismatch")
	}

	return verifyResp.Sub, nil
}
