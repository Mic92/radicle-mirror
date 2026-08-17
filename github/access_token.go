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
	"strings"
	"time"
)

func (c *Client) installationList() ([]appInstallations, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.instAge.Add(10 * time.Minute).After(time.Now()) {
		return c.installations, nil
	}
	jwt, err := c.generateJwt()
	if err != nil {
		return nil, fmt.Errorf("cannot generate JWT: %v", err)
	}
	all, err := c.appInstallations(jwt)
	if err != nil {
		return nil, err
	}
	installations := make([]appInstallations, 0, len(all))
	for _, item := range all {
		if item.AppId == c.appId {
			installations = append(installations, item)
		}
	}
	c.installations = installations
	c.instAge = time.Now()
	return installations, nil
}

func (c *Client) tokenForInstallation(installationId int) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.tokens[installationId]; ok && t.age.Add(50*time.Minute).After(time.Now()) {
		return t.token, nil
	}
	slog.Debug("refreshing access token", "installation", installationId)
	jwt, err := c.generateJwt()
	if err != nil {
		return "", fmt.Errorf("cannot generate JWT: %v", err)
	}
	token, err := c.createInstallationAccessToken(installationId, jwt)
	if err != nil {
		return "", fmt.Errorf("cannot create installation access token: %v", err)
	}
	c.tokens[installationId] = &cachedToken{token: token, age: time.Now()}
	return token, nil
}

func (c *Client) TokenForOwner(owner string) (string, error) {
	installations, err := c.installationList()
	if err != nil {
		return "", err
	}
	for _, inst := range installations {
		if strings.EqualFold(inst.Account.Login, owner) {
			return c.tokenForInstallation(inst.Id)
		}
	}
	return "", fmt.Errorf("no installation found for owner %q", owner)
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
