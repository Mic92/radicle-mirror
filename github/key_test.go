package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestParseRSAKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("cannot generate key: %v", err)
	}
	pkcs1 := x509.MarshalPKCS1PrivateKey(key)
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("cannot marshal pkcs8: %v", err)
	}

	cases := map[string][]byte{
		"pkcs1 der": pkcs1,
		"pkcs8 der": pkcs8,
		"pkcs1 pem": pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: pkcs1}),
		"pkcs8 pem": pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}),
	}
	for name, content := range cases {
		parsed, err := parseRSAKey(content)
		if err != nil {
			t.Errorf("parseRSAKey(%s): %v", name, err)
			continue
		}
		if !parsed.Equal(key) {
			t.Errorf("parseRSAKey(%s): key mismatch", name)
		}
	}
}
