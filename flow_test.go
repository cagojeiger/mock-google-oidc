package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// flowServer creates a test server and returns (mux, store, keys).
func flowServer() (*http.ServeMux, *Store, *KeyPair) {
	return setupTestServer()
}

// doAuthorizeGET performs the GET /o/oauth2/v2/auth step and returns the response.
func doAuthorizeGET(t *testing.T, mux *http.ServeMux, redirectURI, state string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/o/oauth2/v2/auth?redirect_uri="+url.QueryEscape(redirectURI)+"&state="+state+"&client_id=test-app&scope=openid+email+profile&nonce=testnonce", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("authorize GET: expected 200, got %d", w.Code)
	}
	return w
}

// doAuthorizePOSTWithPKCE performs the POST with PKCE parameters.
func doAuthorizePOSTWithPKCE(t *testing.T, mux *http.ServeMux, email, name, state, redirectURI, responseMode, codeChallenge, codeChallengeMethod string) *url.URL {
	t.Helper()
	values := url.Values{
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"nonce":                 {"testnonce"},
		"scope":                 {"openid email profile"},
		"client_id":             {"test-app"},
		"email":                 {email},
		"name":                  {name},
		"response_mode":         {responseMode},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {codeChallengeMethod},
	}
	req := httptest.NewRequest("POST", "/o/oauth2/v2/auth", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 302 {
		t.Fatalf("authorize POST: expected 302, got %d: %s", w.Code, w.Body.String())
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("invalid redirect Location: %v", err)
	}
	return loc
}

// doTokenWithVerifier performs POST /token with a code_verifier.
func doTokenWithVerifier(t *testing.T, mux *http.ServeMux, code, verifier string) (int, map[string]any) {
	t.Helper()
	values := url.Values{
		"code":          {code},
		"client_id":     {"test-app"},
		"client_secret": {"test-secret"},
		"redirect_uri":  {"http://localhost:9090/callback"},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifier},
	}
	req := httptest.NewRequest("POST", "/token", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return w.Code, resp
}

// doAuthorizePOST performs the POST /o/oauth2/v2/auth step and returns the redirect Location.
func doAuthorizePOST(t *testing.T, mux *http.ServeMux, email, name, state, redirectURI, responseMode string) *url.URL {
	t.Helper()
	values := url.Values{
		"redirect_uri":  {redirectURI},
		"state":         {state},
		"nonce":         {"testnonce"},
		"scope":         {"openid email profile"},
		"client_id":     {"test-app"},
		"email":         {email},
		"name":          {name},
		"response_mode": {responseMode},
	}
	req := httptest.NewRequest("POST", "/o/oauth2/v2/auth", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 302 {
		t.Fatalf("authorize POST: expected 302, got %d: %s", w.Code, w.Body.String())
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("invalid redirect Location: %v", err)
	}
	return loc
}

// doToken performs the POST /token step and returns the parsed response.
func doToken(t *testing.T, mux *http.ServeMux, code string) (int, map[string]any) {
	t.Helper()
	values := url.Values{
		"code":          {code},
		"client_id":     {"test-app"},
		"client_secret": {"test-secret"},
		"redirect_uri":  {"http://localhost:9090/callback"},
		"grant_type":    {"authorization_code"},
	}
	req := httptest.NewRequest("POST", "/token", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return w.Code, resp
}

// doUserinfo performs the GET /v1/userinfo step and returns the parsed response.
func doUserinfo(t *testing.T, mux *http.ServeMux, accessToken string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("GET", "/v1/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return w.Code, resp
}

// decodeJWTClaims extracts the claims from a JWT without verifying the signature.
func decodeJWTClaims(t *testing.T, jwt string) map[string]any {
	t.Helper()
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode JWT claims: %v", err)
	}
	var claims map[string]any
	json.Unmarshal(b, &claims)
	return claims
}

// verifyJWTSignature verifies the RS256 signature of a JWT using the provided public key.
func verifyJWTSignature(t *testing.T, jwt string, pub *rsa.PublicKey) {
	t.Helper()
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}
	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	hash := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hash[:], sig); err != nil {
		t.Fatalf("JWT signature verification failed: %v", err)
	}
}

// fetchJWKSPublicKey fetches the public key from /oauth2/v3/certs and parses it.
func fetchJWKSPublicKey(t *testing.T, mux *http.ServeMux) *rsa.PublicKey {
	t.Helper()
	req := httptest.NewRequest("GET", "/oauth2/v3/certs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("certs: expected 200, got %d", w.Code)
	}
	var jwks struct {
		Keys []struct {
			N string `json:"n"`
			E string `json:"e"`
		} `json:"keys"`
	}
	json.NewDecoder(w.Body).Decode(&jwks)
	if len(jwks.Keys) == 0 {
		t.Fatal("no keys in JWKS")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(jwks.Keys[0].N)
	if err != nil {
		t.Fatalf("decode N: %v", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwks.Keys[0].E)
	if err != nil {
		t.Fatalf("decode E: %v", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}
}

// === Full Flow Tests ===

func TestFullFlow_Normal(t *testing.T) {
	mux, _, _ := flowServer()
	redirectURI := "http://localhost:9090/callback"
	state := "state123"

	// 1. GET authorize — render login page
	doAuthorizeGET(t, mux, redirectURI, state)

	// 2. POST authorize — login
	loc := doAuthorizePOST(t, mux, "alice@example.com", "Alice", state, redirectURI, "normal")
	code := loc.Query().Get("code")
	if code == "" {
		t.Fatal("expected code in redirect")
	}
	if loc.Query().Get("state") != state {
		t.Fatal("expected state preserved")
	}

	// 3. POST /token
	status, tokenResp := doToken(t, mux, code)
	if status != 200 {
		t.Fatalf("token: expected 200, got %d", status)
	}
	accessToken, _ := tokenResp["access_token"].(string)
	idToken, _ := tokenResp["id_token"].(string)
	if accessToken == "" || idToken == "" {
		t.Fatal("expected access_token and id_token")
	}

	// 4. Verify id_token claims
	claims := decodeJWTClaims(t, idToken)
	if claims["sub"] != DeterministicSub("alice@example.com") {
		t.Errorf("id_token sub: got %v, want %v", claims["sub"], DeterministicSub("alice@example.com"))
	}
	if claims["email"] != "alice@example.com" {
		t.Errorf("id_token email: got %v", claims["email"])
	}
	if claims["name"] != "Alice" {
		t.Errorf("id_token name: got %v", claims["name"])
	}
	if claims["nonce"] != "testnonce" {
		t.Errorf("id_token nonce: got %v", claims["nonce"])
	}

	// 5. Verify id_token signature using JWKS
	pub := fetchJWKSPublicKey(t, mux)
	verifyJWTSignature(t, idToken, pub)

	// 6. GET /v1/userinfo
	uiStatus, userinfo := doUserinfo(t, mux, accessToken)
	if uiStatus != 200 {
		t.Fatalf("userinfo: expected 200, got %d", uiStatus)
	}
	if userinfo["sub"] != DeterministicSub("alice@example.com") {
		t.Errorf("userinfo sub mismatch")
	}
	if userinfo["email"] != "alice@example.com" {
		t.Errorf("userinfo email mismatch")
	}

	// 7. sub consistency: id_token sub == userinfo sub
	if claims["sub"] != userinfo["sub"] {
		t.Errorf("sub mismatch: id_token=%v, userinfo=%v", claims["sub"], userinfo["sub"])
	}
}

func TestFullFlow_SameEmailSameSub(t *testing.T) {
	mux, _, _ := flowServer()
	redirectURI := "http://localhost:9090/callback"

	// First login
	loc1 := doAuthorizePOST(t, mux, "repeat@example.com", "Repeat", "s1", redirectURI, "normal")
	_, resp1 := doToken(t, mux, loc1.Query().Get("code"))
	claims1 := decodeJWTClaims(t, resp1["id_token"].(string))

	// Second login — same email
	loc2 := doAuthorizePOST(t, mux, "repeat@example.com", "Repeat", "s2", redirectURI, "normal")
	_, resp2 := doToken(t, mux, loc2.Query().Get("code"))
	claims2 := decodeJWTClaims(t, resp2["id_token"].(string))

	if claims1["sub"] != claims2["sub"] {
		t.Errorf("same email should produce same sub: %v != %v", claims1["sub"], claims2["sub"])
	}
}

func TestFullFlow_DifferentEmailDifferentSub(t *testing.T) {
	mux, _, _ := flowServer()
	redirectURI := "http://localhost:9090/callback"

	loc1 := doAuthorizePOST(t, mux, "user-a@example.com", "A", "s1", redirectURI, "normal")
	_, resp1 := doToken(t, mux, loc1.Query().Get("code"))
	claims1 := decodeJWTClaims(t, resp1["id_token"].(string))

	loc2 := doAuthorizePOST(t, mux, "user-b@example.com", "B", "s2", redirectURI, "normal")
	_, resp2 := doToken(t, mux, loc2.Query().Get("code"))
	claims2 := decodeJWTClaims(t, resp2["id_token"].(string))

	if claims1["sub"] == claims2["sub"] {
		t.Errorf("different emails should produce different sub: both %v", claims1["sub"])
	}
}

func TestFullFlow_Deny(t *testing.T) {
	mux, _, _ := flowServer()
	redirectURI := "http://localhost:9090/callback"

	loc := doAuthorizePOST(t, mux, "alice@example.com", "Alice", "s1", redirectURI, "deny")

	if loc.Query().Get("error") != "access_denied" {
		t.Errorf("expected error=access_denied, got %v", loc.Query().Get("error"))
	}
	if loc.Query().Get("code") != "" {
		t.Error("expected no code in deny mode")
	}
}

func TestFullFlow_TokenError(t *testing.T) {
	mux, _, _ := flowServer()
	redirectURI := "http://localhost:9090/callback"

	// Authorize succeeds with code
	loc := doAuthorizePOST(t, mux, "alice@example.com", "Alice", "s1", redirectURI, "token_error")
	code := loc.Query().Get("code")
	if code == "" {
		t.Fatal("expected code even in token_error mode")
	}

	// Token exchange fails
	status, resp := doToken(t, mux, code)
	if status != 500 {
		t.Fatalf("expected 500, got %d", status)
	}
	if resp["error"] != "server_error" {
		t.Errorf("expected server_error, got %v", resp["error"])
	}
}

func TestFullFlow_UserinfoError(t *testing.T) {
	mux, _, _ := flowServer()
	redirectURI := "http://localhost:9090/callback"

	// Authorize succeeds
	loc := doAuthorizePOST(t, mux, "alice@example.com", "Alice", "s1", redirectURI, "userinfo_error")
	code := loc.Query().Get("code")

	// Token exchange succeeds
	status, tokenResp := doToken(t, mux, code)
	if status != 200 {
		t.Fatalf("token: expected 200, got %d", status)
	}
	accessToken := tokenResp["access_token"].(string)

	// Userinfo fails
	uiStatus, uiResp := doUserinfo(t, mux, accessToken)
	if uiStatus != 500 {
		t.Fatalf("userinfo: expected 500, got %d", uiStatus)
	}
	if uiResp["error"] != "server_error" {
		t.Errorf("expected server_error, got %v", uiResp["error"])
	}
}

func TestFullFlow_PKCE_S256(t *testing.T) {
	mux, _, _ := flowServer()
	redirectURI := "http://localhost:9090/callback"

	// Generate PKCE pair
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	// 1. Authorize with code_challenge
	loc := doAuthorizePOSTWithPKCE(t, mux, "pkce@example.com", "PKCE", "s1", redirectURI, "normal", challenge, "S256")
	code := loc.Query().Get("code")
	if code == "" {
		t.Fatal("expected code")
	}

	// 2. Token with correct verifier — should succeed
	status, resp := doTokenWithVerifier(t, mux, code, verifier)
	if status != 200 {
		t.Fatalf("expected 200, got %d: %v", status, resp)
	}
	if resp["access_token"] == nil || resp["id_token"] == nil {
		t.Error("expected access_token and id_token")
	}

	// 3. Verify userinfo works
	accessToken := resp["access_token"].(string)
	uiStatus, userinfo := doUserinfo(t, mux, accessToken)
	if uiStatus != 200 {
		t.Fatalf("userinfo: expected 200, got %d", uiStatus)
	}
	if userinfo["email"] != "pkce@example.com" {
		t.Errorf("wrong email: %v", userinfo["email"])
	}
}

func TestFullFlow_PKCE_S256_WrongVerifier(t *testing.T) {
	mux, _, _ := flowServer()
	redirectURI := "http://localhost:9090/callback"

	verifier := "correct-verifier"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	loc := doAuthorizePOSTWithPKCE(t, mux, "pkce@example.com", "PKCE", "s1", redirectURI, "normal", challenge, "S256")
	code := loc.Query().Get("code")

	// Wrong verifier — should fail
	status, resp := doTokenWithVerifier(t, mux, code, "wrong-verifier")
	if status != 400 {
		t.Fatalf("expected 400, got %d", status)
	}
	if resp["error"] != "invalid_grant" {
		t.Errorf("expected invalid_grant, got %v", resp["error"])
	}
}
