package liff

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Verifier struct {
	channelID string
}

func NewVerifier(channelID string) *Verifier {
	return &Verifier{channelID: channelID}
}

type VerifyResponse struct {
	ClientID string `json:"client_id"`
	Exp      int64  `json:"exp"`
}

type ProfileResponse struct {
	UserID        string `json:"userId"`
	DisplayName   string `json:"displayName"`
	PictureURL    string `json:"pictureUrl"`
	StatusMessage string `json:"statusMessage"`
}

type IDTokenVerifyResponse struct {
	ISS     string   `json:"iss"`
	Sub     string   `json:"sub"` // LINE user ID
	Aud     string   `json:"aud"` // Channel ID
	Exp     int64    `json:"exp"`
	Iat     int64    `json:"iat"`
	Nonce   string   `json:"nonce"`
	AmR     []string `json:"amr"`
	Name    string   `json:"name"`
	Picture string   `json:"picture"`
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

	// Get user profile using the access token
	profileURL := "https://api.line.me/v2/profile"
	profileReq, err := http.NewRequest("GET", profileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create profile request: %w", err)
	}
	profileReq.Header.Set("Authorization", "Bearer "+accessToken)

	profileResp, err := http.DefaultClient.Do(profileReq)
	if err != nil {
		return "", fmt.Errorf("failed to get profile: %w", err)
	}
	defer profileResp.Body.Close()

	if profileResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(profileResp.Body)
		return "", fmt.Errorf("profile request failed: %s", string(body))
	}

	var profile ProfileResponse
	if err := json.NewDecoder(profileResp.Body).Decode(&profile); err != nil {
		return "", fmt.Errorf("failed to decode profile response: %w", err)
	}

	return profile.UserID, nil
}

// VerifyIDToken verifies LIFF ID token and returns LINE user ID
func (v *Verifier) VerifyIDToken(idToken string) (string, error) {
	// Call LINE's ID token verification endpoint
	apiURL := "https://api.line.me/oauth2/v2.1/verify"

	// Prepare form data
	data := url.Values{}
	data.Set("id_token", idToken)
	data.Set("client_id", v.channelID)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to verify ID token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ID token verification failed: %s", string(body))
	}

	var verifyResp IDTokenVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Verify channel ID matches
	if verifyResp.Aud != v.channelID {
		return "", fmt.Errorf("channel ID mismatch")
	}

	return verifyResp.Sub, nil
}
