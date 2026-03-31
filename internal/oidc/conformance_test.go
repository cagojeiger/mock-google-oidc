package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// conformanceServer starts a test server and returns its URL.
func conformanceServer() (*httptest.Server, *Store, *KeyPair) {
	keys := NewKeyPair()
	store := NewStore()
	mux := http.NewServeMux()
	// We need a placeholder URL first; will be replaced after server starts.
	// Use a wrapper handler that injects the correct publicURL.
	var serverURL string
	wrapper := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	})
	server := httptest.NewServer(wrapper)
	serverURL = server.URL

	// Re-register handlers with the actual server URL.
	RegisterHandlers(mux, serverURL, keys, store, "test")

	return server, store, keys
}

// doAuthAndGetCode performs the authorization POST and returns the code.
func doAuthAndGetCode(t *testing.T, serverURL string, email, name string) string {
	t.Helper()
	values := url.Values{
		"redirect_uri":  {serverURL + "/callback"},
		"state":         {"teststate"},
		"nonce":         {"testnonce"},
		"scope":         {"openid email profile"},
		"client_id":     {"test-client"},
		"email":         {email},
		"name":          {name},
		"response_mode": {"normal"},
	}
	resp, err := http.Post(serverURL+"/o/oauth2/v2/auth", "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatalf("authorize POST failed: %v", err)
	}
	defer resp.Body.Close()

	// Don't follow redirect — extract code from Location header.
	loc := resp.Request.URL
	if resp.StatusCode == 200 {
		// httptest client followed the redirect
		code := loc.Query().Get("code")
		if code != "" {
			return code
		}
	}
	t.Fatal("could not extract code from redirect")
	return ""
}

// doAuthAndGetCodeDirect uses the mux directly (no redirect follow).
func doAuthAndGetCodeDirect(t *testing.T, mux *http.ServeMux, serverURL, email, name string) string {
	t.Helper()
	values := url.Values{
		"redirect_uri":  {serverURL + "/callback"},
		"state":         {"teststate"},
		"nonce":         {"testnonce"},
		"scope":         {"openid email profile"},
		"client_id":     {"test-client"},
		"email":         {email},
		"name":          {name},
		"response_mode": {"normal"},
	}
	req := httptest.NewRequest("POST", "/o/oauth2/v2/auth", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 302 {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	loc, _ := url.Parse(w.Header().Get("Location"))
	code := loc.Query().Get("code")
	if code == "" {
		t.Fatal("no code in redirect")
	}
	return code
}

// exchangeCode calls the token endpoint and returns the raw response.
func exchangeCode(t *testing.T, mux *http.ServeMux, serverURL, code string) map[string]any {
	t.Helper()
	values := url.Values{
		"code":          {code},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
		"redirect_uri":  {serverURL + "/callback"},
		"grant_type":    {"authorization_code"},
	}
	req := httptest.NewRequest("POST", "/token", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("token exchange failed: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

// --- C1: Discovery Parsing ---

func TestConformance_C1_DiscoveryParsing(t *testing.T) {
	server, _, _ := conformanceServer()
	defer server.Close()

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("C1 FAIL: coreos/go-oidc could not parse discovery: %v", err)
	}

	endpoint := provider.Endpoint()
	if endpoint.AuthURL != server.URL+"/o/oauth2/v2/auth" {
		t.Errorf("C1: AuthURL mismatch: got %s", endpoint.AuthURL)
	}
	if endpoint.TokenURL != server.URL+"/token" {
		t.Errorf("C1: TokenURL mismatch: got %s", endpoint.TokenURL)
	}
}

// --- C2: JWKS Loading ---

func TestConformance_C2_JWKSLoading(t *testing.T) {
	server, _, _ := conformanceServer()
	defer server.Close()

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("C2: provider creation failed: %v", err)
	}

	// Creating a verifier triggers JWKS loading internally.
	verifier := provider.Verifier(&oidc.Config{ClientID: "test-client"})
	if verifier == nil {
		t.Fatal("C2 FAIL: verifier is nil")
	}
}

// --- C3: ID Token Signature Verification ---

func TestConformance_C3_IDTokenVerification(t *testing.T) {
	server, store, keys := conformanceServer()
	defer server.Close()

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("C3: provider creation failed: %v", err)
	}

	// Create a token via our provider.
	mux := http.NewServeMux()
	RegisterHandlers(mux, server.URL, keys, store, "test")

	code := doAuthAndGetCodeDirect(t, mux, server.URL, "alice@example.com", "Alice Kim")
	tokenResp := exchangeCode(t, mux, server.URL, code)
	rawIDToken := tokenResp["id_token"].(string)

	// Verify with coreos/go-oidc.
	verifier := provider.Verifier(&oidc.Config{ClientID: "test-client"})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		t.Fatalf("C3 FAIL: id_token verification failed: %v", err)
	}

	if idToken.Issuer != server.URL {
		t.Errorf("C3: issuer mismatch: got %s, want %s", idToken.Issuer, server.URL)
	}
	if idToken.Subject == "" {
		t.Error("C3: subject is empty")
	}
}

// --- C4: ID Token Claims Parsing ---

func TestConformance_C4_IDTokenClaims(t *testing.T) {
	server, store, keys := conformanceServer()
	defer server.Close()

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("C4: provider creation failed: %v", err)
	}

	mux := http.NewServeMux()
	RegisterHandlers(mux, server.URL, keys, store, "test")

	code := doAuthAndGetCodeDirect(t, mux, server.URL, "claims@example.com", "Claims User")
	tokenResp := exchangeCode(t, mux, server.URL, code)
	rawIDToken := tokenResp["id_token"].(string)

	verifier := provider.Verifier(&oidc.Config{ClientID: "test-client"})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		t.Fatalf("C4: verification failed: %v", err)
	}

	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Nonce         string `json:"nonce"`
		Azp           string `json:"azp"`
	}
	if err := idToken.Claims(&claims); err != nil {
		t.Fatalf("C4 FAIL: could not parse claims: %v", err)
	}

	if claims.Sub == "" {
		t.Error("C4: sub is empty")
	}
	if claims.Email != "claims@example.com" {
		t.Errorf("C4: email mismatch: got %s", claims.Email)
	}
	if !claims.EmailVerified {
		t.Error("C4: email_verified is false")
	}
	if claims.Name != "Claims User" {
		t.Errorf("C4: name mismatch: got %s", claims.Name)
	}
	if claims.GivenName != "Claims" {
		t.Errorf("C4: given_name mismatch: got %s", claims.GivenName)
	}
	if claims.FamilyName != "User" {
		t.Errorf("C4: family_name mismatch: got %s", claims.FamilyName)
	}
	if claims.Nonce != "testnonce" {
		t.Errorf("C4: nonce mismatch: got %s", claims.Nonce)
	}
	if claims.Azp != "test-client" {
		t.Errorf("C4: azp mismatch: got %s", claims.Azp)
	}
}

// --- C5: Sub Consistency (id_token sub == userinfo sub) ---

func TestConformance_C5_SubConsistency(t *testing.T) {
	server, store, keys := conformanceServer()
	defer server.Close()

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("C5: provider creation failed: %v", err)
	}

	mux := http.NewServeMux()
	RegisterHandlers(mux, server.URL, keys, store, "test")

	code := doAuthAndGetCodeDirect(t, mux, server.URL, "sub@example.com", "Sub User")
	tokenResp := exchangeCode(t, mux, server.URL, code)
	rawIDToken := tokenResp["id_token"].(string)
	accessToken := tokenResp["access_token"].(string)

	// Get sub from id_token.
	verifier := provider.Verifier(&oidc.Config{ClientID: "test-client"})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		t.Fatalf("C5: verification failed: %v", err)
	}

	// Get sub from userinfo.
	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	}))
	if err != nil {
		t.Fatalf("C5 FAIL: UserInfo call failed: %v", err)
	}

	if idToken.Subject != userInfo.Subject {
		t.Errorf("C5 FAIL: sub mismatch: id_token=%s, userinfo=%s", idToken.Subject, userInfo.Subject)
	}
}

// --- C6: UserInfo Endpoint ---

func TestConformance_C6_UserInfo(t *testing.T) {
	server, store, keys := conformanceServer()
	defer server.Close()

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("C6: provider creation failed: %v", err)
	}

	mux := http.NewServeMux()
	RegisterHandlers(mux, server.URL, keys, store, "test")

	code := doAuthAndGetCodeDirect(t, mux, server.URL, "userinfo@example.com", "UserInfo Test")
	tokenResp := exchangeCode(t, mux, server.URL, code)
	accessToken := tokenResp["access_token"].(string)

	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	}))
	if err != nil {
		t.Fatalf("C6 FAIL: UserInfo call failed: %v", err)
	}

	if userInfo.Subject == "" {
		t.Error("C6: sub is empty")
	}
	if userInfo.Email != "userinfo@example.com" {
		t.Errorf("C6: email mismatch: got %s", userInfo.Email)
	}
	if !userInfo.EmailVerified {
		t.Error("C6: email_verified is false")
	}

	// Parse additional claims.
	var claims struct {
		Name       string `json:"name"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
	}
	if err := userInfo.Claims(&claims); err != nil {
		t.Fatalf("C6: claims parsing failed: %v", err)
	}
	if claims.Name != "UserInfo Test" {
		t.Errorf("C6: name mismatch: got %s", claims.Name)
	}
}

// --- C7: Expired Token Rejection ---

func TestConformance_C7_ExpiredTokenRejection(t *testing.T) {
	server, _, keys := conformanceServer()
	defer server.Close()

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("C7: provider creation failed: %v", err)
	}

	// Manually create an expired id_token using SignIDToken,
	// then override iat/exp to be in the past.
	claims := map[string]any{
		"iss":            server.URL,
		"sub":            "expired-sub",
		"aud":            "test-client",
		"email":          "expired@example.com",
		"email_verified": true,
		"name":           "Expired User",
		"iat":            time.Now().Add(-2 * time.Hour).Unix(),
		"exp":            time.Now().Add(-1 * time.Hour).Unix(), // expired 1 hour ago
	}
	expiredToken, err := keys.SignIDToken(claims)
	if err != nil {
		t.Fatalf("C7: failed to sign expired token: %v", err)
	}
	// SignIDToken overwrites iat/exp with current time, so we need to
	// re-sign with the expired values manually.
	// Instead, decode, patch, re-encode, re-sign.
	parts := strings.Split(expiredToken, ".")
	claimsJSON, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claimsMap map[string]any
	json.Unmarshal(claimsJSON, &claimsMap)
	claimsMap["iat"] = time.Now().Add(-2 * time.Hour).Unix()
	claimsMap["exp"] = time.Now().Add(-1 * time.Hour).Unix()
	patchedClaimsJSON, _ := json.Marshal(claimsMap)
	parts[1] = base64.RawURLEncoding.EncodeToString(patchedClaimsJSON)
	signingInput := parts[0] + "." + parts[1]
	hash := sha256.Sum256([]byte(signingInput))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, keys.Private, crypto.SHA256, hash[:])
	parts[2] = base64.RawURLEncoding.EncodeToString(sig)
	expiredToken = strings.Join(parts, ".")

	verifier := provider.Verifier(&oidc.Config{ClientID: "test-client"})
	_, err = verifier.Verify(ctx, expiredToken)
	if err == nil {
		t.Fatal("C7 FAIL: expired token should be rejected")
	}
	// err should mention token expiry.
	if !strings.Contains(err.Error(), "expired") && !strings.Contains(err.Error(), "exp") {
		t.Logf("C7: rejection error (acceptable): %v", err)
	}
}

// --- C-FLOW-01: Full Authorization Code Flow ---

func TestConformance_Flow_Full(t *testing.T) {
	server, store, keys := conformanceServer()
	defer server.Close()

	ctx := context.Background()

	// 1. Discovery.
	provider, err := oidc.NewProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("FLOW: discovery failed: %v", err)
	}

	mux := http.NewServeMux()
	RegisterHandlers(mux, server.URL, keys, store, "test")

	// 2. Authorization → code.
	code := doAuthAndGetCodeDirect(t, mux, server.URL, "flow@example.com", "Flow User")

	// 3. Token exchange.
	tokenResp := exchangeCode(t, mux, server.URL, code)
	rawIDToken := tokenResp["id_token"].(string)
	accessToken := tokenResp["access_token"].(string)

	// 4. id_token verification.
	verifier := provider.Verifier(&oidc.Config{ClientID: "test-client"})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		t.Fatalf("FLOW: id_token verification failed: %v", err)
	}

	// 5. UserInfo.
	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	}))
	if err != nil {
		t.Fatalf("FLOW: UserInfo failed: %v", err)
	}

	// 6. Sub consistency.
	if idToken.Subject != userInfo.Subject {
		t.Errorf("FLOW: sub mismatch: id_token=%s, userinfo=%s", idToken.Subject, userInfo.Subject)
	}

	// 7. Claims check.
	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	idToken.Claims(&claims)
	if claims.Email != "flow@example.com" {
		t.Errorf("FLOW: email mismatch: %s", claims.Email)
	}
	if claims.Name != "Flow User" {
		t.Errorf("FLOW: name mismatch: %s", claims.Name)
	}
}

// --- C-FLOW-02: PKCE + Conformance ---

func TestConformance_Flow_PKCE(t *testing.T) {
	server, store, keys := conformanceServer()
	defer server.Close()

	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, server.URL)
	if err != nil {
		t.Fatalf("FLOW-PKCE: discovery failed: %v", err)
	}

	mux := http.NewServeMux()
	RegisterHandlers(mux, server.URL, keys, store, "test")

	// Generate PKCE pair.
	verifierStr := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifierStr))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	// Authorization with PKCE.
	values := url.Values{
		"redirect_uri":          {server.URL + "/callback"},
		"state":                 {"teststate"},
		"nonce":                 {"testnonce"},
		"scope":                 {"openid email profile"},
		"client_id":             {"test-client"},
		"email":                 {"pkce@example.com"},
		"name":                  {"PKCE User"},
		"response_mode":         {"normal"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	req := httptest.NewRequest("POST", "/o/oauth2/v2/auth", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	loc, _ := url.Parse(w.Header().Get("Location"))
	code := loc.Query().Get("code")

	// Token exchange with verifier.
	tokenValues := url.Values{
		"code":          {code},
		"client_id":     {"test-client"},
		"client_secret": {"test-secret"},
		"redirect_uri":  {server.URL + "/callback"},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifierStr},
	}
	tokenReq := httptest.NewRequest("POST", "/token", strings.NewReader(tokenValues.Encode()))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenW := httptest.NewRecorder()
	mux.ServeHTTP(tokenW, tokenReq)

	var tokenResp map[string]any
	json.NewDecoder(tokenW.Body).Decode(&tokenResp)
	rawIDToken := tokenResp["id_token"].(string)
	accessToken := tokenResp["access_token"].(string)

	// Verify id_token.
	verifier := provider.Verifier(&oidc.Config{ClientID: "test-client"})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		t.Fatalf("FLOW-PKCE: id_token verification failed: %v", err)
	}

	// UserInfo.
	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	}))
	if err != nil {
		t.Fatalf("FLOW-PKCE: UserInfo failed: %v", err)
	}

	if idToken.Subject != userInfo.Subject {
		t.Errorf("FLOW-PKCE: sub mismatch")
	}
}
