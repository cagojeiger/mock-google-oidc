package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const testPublicURL = "http://localhost:8082"

func setupTestServer() (*http.ServeMux, *Store, *KeyPair) {
	keys := NewKeyPair()
	store := NewStore()
	mux := http.NewServeMux()
	RegisterHandlers(mux, testPublicURL, keys, store)
	return mux, store, keys
}

// --- Discovery ---

func TestDiscovery(t *testing.T) {
	mux, _, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/.well-known/openid-configuration", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var doc map[string]any
	json.NewDecoder(w.Body).Decode(&doc)

	checks := map[string]string{
		"issuer":                 testPublicURL,
		"authorization_endpoint": testPublicURL + "/o/oauth2/v2/auth",
		"token_endpoint":         testPublicURL + "/token",
		"userinfo_endpoint":      testPublicURL + "/v1/userinfo",
		"jwks_uri":               testPublicURL + "/oauth2/v3/certs",
	}
	for k, want := range checks {
		got, _ := doc[k].(string)
		if got != want {
			t.Errorf("%s: got %q, want %q", k, got, want)
		}
	}
}

// --- Authorize GET ---

func TestAuthorizeGET_Normal(t *testing.T) {
	mux, _, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/o/oauth2/v2/auth?redirect_uri=http://example.com/cb&state=abc&client_id=app1&scope=openid", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "alice@example.com") {
		t.Error("expected default email in form")
	}
	if !strings.Contains(body, "http://example.com/cb") {
		t.Error("expected redirect_uri in form")
	}
}

func TestAuthorizeGET_LoginHint(t *testing.T) {
	mux, _, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/o/oauth2/v2/auth?redirect_uri=http://example.com/cb&state=abc&login_hint=bob@test.com", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "bob@test.com") {
		t.Error("expected login_hint email in form")
	}
}

func TestAuthorizeGET_MissingRedirectURI(t *testing.T) {
	mux, _, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/o/oauth2/v2/auth?state=abc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAuthorizeGET_MissingState(t *testing.T) {
	mux, _, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/o/oauth2/v2/auth?redirect_uri=http://example.com/cb", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- Authorize POST ---

func postAuthorize(mux *http.ServeMux, values url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/o/oauth2/v2/auth", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestAuthorizePOST_Normal(t *testing.T) {
	mux, _, _ := setupTestServer()
	w := postAuthorize(mux, url.Values{
		"redirect_uri": {"http://example.com/cb"},
		"state":        {"abc"},
		"email":        {"alice@example.com"},
		"name":         {"Alice"},
	})

	if w.Code != 302 {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	loc, _ := url.Parse(w.Header().Get("Location"))
	if loc.Query().Get("code") == "" {
		t.Error("expected code in redirect")
	}
	if loc.Query().Get("state") != "abc" {
		t.Error("expected state=abc in redirect")
	}
}

func TestAuthorizePOST_Deny(t *testing.T) {
	mux, _, _ := setupTestServer()
	w := postAuthorize(mux, url.Values{
		"redirect_uri":  {"http://example.com/cb"},
		"state":         {"abc"},
		"email":         {"alice@example.com"},
		"name":          {"Alice"},
		"response_mode": {"deny"},
	})

	if w.Code != 302 {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	loc, _ := url.Parse(w.Header().Get("Location"))
	if loc.Query().Get("error") != "access_denied" {
		t.Error("expected error=access_denied")
	}
	if loc.Query().Get("code") != "" {
		t.Error("expected no code in deny mode")
	}
}

func TestAuthorizePOST_MissingEmail(t *testing.T) {
	mux, _, _ := setupTestServer()
	w := postAuthorize(mux, url.Values{
		"redirect_uri": {"http://example.com/cb"},
		"state":        {"abc"},
		"name":         {"Alice"},
	})

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- Token ---

func postToken(mux *http.ServeMux, values url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/token", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestToken_Normal(t *testing.T) {
	mux, store, _ := setupTestServer()
	store.SaveCode("test-code-1", &CodeEntry{
		Sub:          DeterministicSub("alice@example.com"),
		Email:        "alice@example.com",
		Name:         "Alice",
		ClientID:     "app1",
		Scope:        "openid email profile",
		ResponseMode: "normal",
	})

	w := postToken(mux, url.Values{
		"code":          {"test-code-1"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
	})

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["access_token"] == nil {
		t.Error("expected access_token")
	}
	if resp["id_token"] == nil {
		t.Error("expected id_token")
	}
	if resp["token_type"] != "Bearer" {
		t.Errorf("expected token_type Bearer, got %v", resp["token_type"])
	}
	if resp["expires_in"] != float64(3920) {
		t.Errorf("expected expires_in 3920, got %v", resp["expires_in"])
	}
}

func TestToken_InvalidCode(t *testing.T) {
	mux, _, _ := setupTestServer()
	w := postToken(mux, url.Values{
		"code":          {"nonexistent"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
	})

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "invalid_grant" {
		t.Errorf("expected invalid_grant, got %s", resp["error"])
	}
}

func TestToken_EmptyCode(t *testing.T) {
	mux, _, _ := setupTestServer()
	w := postToken(mux, url.Values{
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
	})

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestToken_ErrorMode(t *testing.T) {
	mux, store, _ := setupTestServer()
	store.SaveCode("err-code", &CodeEntry{
		Sub:          "sub1",
		Email:        "err@example.com",
		ResponseMode: "token_error",
	})

	w := postToken(mux, url.Values{
		"code":          {"err-code"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
	})

	if w.Code != 500 {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "server_error" {
		t.Errorf("expected server_error, got %s", resp["error"])
	}
}

func TestToken_IDTokenClaims(t *testing.T) {
	mux, store, _ := setupTestServer()
	store.SaveCode("claims-code", &CodeEntry{
		Sub:          DeterministicSub("claims@example.com"),
		Email:        "claims@example.com",
		Name:         "Claims User",
		Nonce:        "nonce123",
		ClientID:     "app1",
		Scope:        "openid email profile",
		ResponseMode: "normal",
	})

	w := postToken(mux, url.Values{
		"code":          {"claims-code"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
	})

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	idTokenStr, ok := resp["id_token"].(string)
	if !ok {
		t.Fatal("id_token not a string")
	}

	parts := strings.Split(idTokenStr, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode claims: %v", err)
	}
	var claims map[string]any
	json.Unmarshal(claimsJSON, &claims)

	checks := map[string]any{
		"iss":            testPublicURL,
		"sub":            DeterministicSub("claims@example.com"),
		"aud":            "app1",
		"email":          "claims@example.com",
		"email_verified": true,
		"name":           "Claims User",
		"nonce":          "nonce123",
	}
	for k, want := range checks {
		got := claims[k]
		if got != want {
			t.Errorf("claim %s: got %v, want %v", k, got, want)
		}
	}

	if claims["iat"] == nil || claims["exp"] == nil {
		t.Error("expected iat and exp claims")
	}
}

func TestToken_CodeSingleUse(t *testing.T) {
	mux, store, _ := setupTestServer()
	store.SaveCode("once-code", &CodeEntry{
		Sub:          "sub1",
		Email:        "once@example.com",
		ResponseMode: "normal",
	})

	// First use — should succeed
	w1 := postToken(mux, url.Values{
		"code":          {"once-code"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
	})
	if w1.Code != 200 {
		t.Fatalf("first use: expected 200, got %d", w1.Code)
	}

	// Second use — should fail
	w2 := postToken(mux, url.Values{
		"code":          {"once-code"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
	})
	if w2.Code != 400 {
		t.Fatalf("second use: expected 400, got %d", w2.Code)
	}
}

func TestToken_WrongGrantType(t *testing.T) {
	mux, store, _ := setupTestServer()
	store.SaveCode("gt-code", &CodeEntry{
		Sub:          "sub1",
		Email:        "gt@example.com",
		ResponseMode: "normal",
	})

	w := postToken(mux, url.Values{
		"code":          {"gt-code"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"refresh_token"},
	})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "unsupported_grant_type" {
		t.Errorf("expected unsupported_grant_type, got %s", resp["error"])
	}
}

// --- Token PKCE ---

func TestToken_PKCE_S256_Valid(t *testing.T) {
	mux, store, _ := setupTestServer()
	// S256: challenge = base64url(sha256(verifier))
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	store.SaveCode("pkce-code", &CodeEntry{
		Sub:                 "sub1",
		Email:               "pkce@example.com",
		Name:                "PKCE User",
		ClientID:            "app1",
		ResponseMode:        "normal",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	})

	w := postToken(mux, url.Values{
		"code":          {"pkce-code"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifier},
	})

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestToken_PKCE_S256_Invalid(t *testing.T) {
	mux, store, _ := setupTestServer()
	verifier := "correct-verifier"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	store.SaveCode("pkce-bad", &CodeEntry{
		Sub:                 "sub1",
		Email:               "pkce@example.com",
		ResponseMode:        "normal",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	})

	w := postToken(mux, url.Values{
		"code":          {"pkce-bad"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
		"code_verifier": {"wrong-verifier"},
	})

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "invalid_grant" {
		t.Errorf("expected invalid_grant, got %s", resp["error"])
	}
}

func TestToken_PKCE_MissingVerifier(t *testing.T) {
	mux, store, _ := setupTestServer()
	store.SaveCode("pkce-noverify", &CodeEntry{
		Sub:                 "sub1",
		Email:               "pkce@example.com",
		ResponseMode:        "normal",
		CodeChallenge:       "some-challenge",
		CodeChallengeMethod: "S256",
	})

	w := postToken(mux, url.Values{
		"code":          {"pkce-noverify"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
	})

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestToken_PKCE_Plain_Valid(t *testing.T) {
	mux, store, _ := setupTestServer()
	verifier := "plain-verifier-value"

	store.SaveCode("pkce-plain", &CodeEntry{
		Sub:                 "sub1",
		Email:               "pkce@example.com",
		ResponseMode:        "normal",
		CodeChallenge:       verifier,
		CodeChallengeMethod: "plain",
	})

	w := postToken(mux, url.Values{
		"code":          {"pkce-plain"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifier},
	})

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestToken_NoPKCE_StillWorks(t *testing.T) {
	mux, store, _ := setupTestServer()
	store.SaveCode("no-pkce", &CodeEntry{
		Sub:          "sub1",
		Email:        "nopkce@example.com",
		ResponseMode: "normal",
	})

	w := postToken(mux, url.Values{
		"code":          {"no-pkce"},
		"client_id":     {"app1"},
		"client_secret": {"secret"},
		"redirect_uri":  {"http://example.com/cb"},
		"grant_type":    {"authorization_code"},
	})

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Userinfo ---

func TestUserinfo_Normal(t *testing.T) {
	mux, store, _ := setupTestServer()
	entry := &CodeEntry{
		Sub:          DeterministicSub("ui@example.com"),
		Email:        "ui@example.com",
		Name:         "UI User",
		ResponseMode: "normal",
	}
	store.SaveToken("ya29.test-token", entry)

	req := httptest.NewRequest("GET", "/v1/userinfo", nil)
	req.Header.Set("Authorization", "Bearer ya29.test-token")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var info map[string]any
	json.NewDecoder(w.Body).Decode(&info)

	if info["sub"] != DeterministicSub("ui@example.com") {
		t.Errorf("wrong sub: %v", info["sub"])
	}
	if info["email"] != "ui@example.com" {
		t.Errorf("wrong email: %v", info["email"])
	}
	if info["name"] != "UI User" {
		t.Errorf("wrong name: %v", info["name"])
	}
	if info["email_verified"] != true {
		t.Error("expected email_verified=true")
	}
}

func TestUserinfo_NoAuth(t *testing.T) {
	mux, _, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/v1/userinfo", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUserinfo_InvalidToken(t *testing.T) {
	mux, _, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/v1/userinfo", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUserinfo_ErrorMode(t *testing.T) {
	mux, store, _ := setupTestServer()
	errEntry := &CodeEntry{
		Sub:          "sub1",
		Email:        "err@example.com",
		ResponseMode: "userinfo_error",
	}
	store.SaveToken("ya29.err-token", errEntry)

	req := httptest.NewRequest("GET", "/v1/userinfo", nil)
	req.Header.Set("Authorization", "Bearer ya29.err-token")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- JWKS ---

func TestCerts(t *testing.T) {
	mux, _, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/oauth2/v3/certs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var jwks map[string]any
	json.NewDecoder(w.Body).Decode(&jwks)

	keys, ok := jwks["keys"].([]any)
	if !ok || len(keys) == 0 {
		t.Fatal("expected keys array with at least one key")
	}

	key := keys[0].(map[string]any)
	if key["kty"] != "RSA" {
		t.Errorf("expected kty=RSA, got %v", key["kty"])
	}
	if key["alg"] != "RS256" {
		t.Errorf("expected alg=RS256, got %v", key["alg"])
	}
	if key["kid"] != "test-idp-key-1" {
		t.Errorf("expected kid=test-idp-key-1, got %v", key["kid"])
	}
}

// --- Health ---

func TestHealth(t *testing.T) {
	mux, _, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", resp["status"])
	}
}
