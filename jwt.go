package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"time"
)

// KeyPair holds RSA keys for signing id_tokens.
type KeyPair struct {
	Private *rsa.PrivateKey
	KID     string
}

// NewKeyPair generates a fresh RSA key pair. Keys are regenerated on every server start.
func NewKeyPair() *KeyPair {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("failed to generate RSA key: " + err.Error())
	}
	return &KeyPair{Private: key, KID: "test-idp-key-1"}
}

// JWKS returns the JSON Web Key Set containing the public key.
func (kp *KeyPair) JWKS() map[string]any {
	pub := &kp.Private.PublicKey
	return map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"kid": kp.KID,
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	}
}

// SignIDToken creates a signed JWT id_token.
func (kp *KeyPair) SignIDToken(claims map[string]any) (string, error) {
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kp.KID,
	}

	now := time.Now()
	claims["iat"] = now.Unix()
	claims["exp"] = now.Add(1 * time.Hour).Unix()

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64

	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, kp.Private, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return signingInput + "." + sigB64, nil
}
