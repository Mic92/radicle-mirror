package github

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

func (c *Client) Token() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokenAge.Add(50 * time.Minute).Before(time.Now()) {
		slog.Debug("refreshing access token")
		token, err := c.requestAccessToken()
		if err != nil {
			return "", fmt.Errorf("cannot request access token: %v", err)
		}
		c._token = token
		c.tokenAge = time.Now()
	}
	return c._token, nil
}

func (c *Client) generateJwt() (string, error) {
	jwtIatDrift := 60
	jwtExpDelta := 600
	iat := time.Now().Unix() - int64(jwtIatDrift)
	jwtPayload, err := json.Marshal(map[string]int64{
		"iat": iat,
		"exp": iat + int64(jwtExpDelta),
		"iss": int64(c.appId),
	})
	if err != nil {
		return "", fmt.Errorf("cannot encode JWT payload: %v", err)
	}

	jwtHeaders, err := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", fmt.Errorf("cannot encode JWT headers: %v", err)
	}
	jwtPayloadBase64 := base64.RawURLEncoding.EncodeToString(jwtPayload)
	jwtHeadersBase64 := base64.RawURLEncoding.EncodeToString(jwtHeaders)

	encodedJwtParts := jwtHeadersBase64 + "." + jwtPayloadBase64
	hashed := sha256.Sum256([]byte(encodedJwtParts))
	encodedMac, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("cannot sign JWT: %v", err)
	}
	generatedJwt := encodedJwtParts + "." + base64.RawURLEncoding.EncodeToString(encodedMac)
	return generatedJwt, nil
}

func (c *Client) requestAccessToken() (string, error) {
	generatedJwt, err := c.generateJwt()
	if err != nil {
		return "", fmt.Errorf("cannot generate JWT: %v", err)
	}
	resp, err := c.appInstallations(generatedJwt)
	if err != nil {
		return "", fmt.Errorf("cannot get app installations: %v", err)
	}
	for _, item := range resp {
		if item.AppId != c.appId {
			continue
		}
		installationId := item.Id
		response, err := c.createInstallationAccessToken(installationId, generatedJwt)
		if err != nil {
			return "", fmt.Errorf("cannot create installation access token: %v", err)
		}
		return response, nil
	}
	return "", fmt.Errorf("installation not found for app id %d", c.appId)
}
