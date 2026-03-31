package oidc

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// RegisterHandlers registers all HTTP handlers on the given mux.
func RegisterHandlers(mux *http.ServeMux, publicURL string, keys *KeyPair, store *Store, version string) {
	mux.HandleFunc("GET /.well-known/openid-configuration", handleDiscovery(publicURL))
	mux.HandleFunc("GET /o/oauth2/v2/auth", handleAuthorizeGET())
	mux.HandleFunc("POST /o/oauth2/v2/auth", handleAuthorizePOST(store))
	mux.HandleFunc("POST /token", handleToken(publicURL, keys, store))
	mux.HandleFunc("GET /v1/userinfo", handleUserinfo(store))
	mux.HandleFunc("GET /oauth2/v3/certs", handleCerts(keys))
	mux.HandleFunc("GET /health", handleHealth(version))
}

func handleDiscovery(publicURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                publicURL,
			"authorization_endpoint":                publicURL + "/o/oauth2/v2/auth",
			"token_endpoint":                        publicURL + "/token",
			"userinfo_endpoint":                     publicURL + "/v1/userinfo",
			"jwks_uri":                              publicURL + "/oauth2/v3/certs",
			"response_types_supported":              []string{"code"},
			"response_modes_supported":              []string{"query"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"scopes_supported":                      []string{"openid", "email", "profile"},
			"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
			"claims_supported":                      []string{"aud", "email", "email_verified", "exp", "family_name", "given_name", "iat", "iss", "name", "picture", "sub"},
			"code_challenge_methods_supported":      []string{"plain", "S256"},
			"grant_types_supported":                 []string{"authorization_code"},
		})
	}
}

func handleAuthorizeGET() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")
		if redirectURI == "" || state == "" {
			http.Error(w, "missing redirect_uri or state", http.StatusBadRequest)
			return
		}

		// OIDC Core 3.1.2.1: response_type is REQUIRED
		if err := ValidateResponseType(r.URL.Query().Get("response_type")); err != nil {
			redirectWithError(w, r, redirectURI, state, "unsupported_response_type", "Only response_type=code is supported.")
			return
		}

		// OIDC Core 3.1.2.1: scope MUST contain "openid"
		if err := RequireOpenIDScope(r.URL.Query().Get("scope")); err != nil {
			redirectWithError(w, r, redirectURI, state, "invalid_scope", "The openid scope is required.")
			return
		}

		// OIDC Core 3.1.2.1: prompt=none requires no UI
		if err := ValidatePrompt(r.URL.Query().Get("prompt")); err != nil {
			redirectWithError(w, r, redirectURI, state, "login_required", "This mock provider always requires login.")
			return
		}

		email := "alice@gmail.com"
		if hint := r.URL.Query().Get("login_hint"); hint != "" {
			email = hint
		}

		data := LoginPageData{
			RedirectURI:         redirectURI,
			State:               state,
			Nonce:               r.URL.Query().Get("nonce"),
			Scope:               r.URL.Query().Get("scope"),
			ClientID:            r.URL.Query().Get("client_id"),
			CodeChallenge:       r.URL.Query().Get("code_challenge"),
			CodeChallengeMethod: r.URL.Query().Get("code_challenge_method"),
			Email:               email,
			Name:                "Alice",
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		loginTemplate.Execute(w, data)
	}
}

func handleAuthorizePOST(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		redirectURI := r.FormValue("redirect_uri")
		state := r.FormValue("state")
		if redirectURI == "" || state == "" {
			http.Error(w, "missing redirect_uri or state", http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		name := r.FormValue("name")
		if email == "" {
			http.Error(w, "missing email", http.StatusBadRequest)
			return
		}
		if name == "" {
			http.Error(w, "missing name", http.StatusBadRequest)
			return
		}

		responseMode := r.FormValue("response_mode")
		if responseMode == "" {
			responseMode = "normal"
		}

		// Deny mode: redirect with error, no code
		if responseMode == "deny" {
			redirectWithError(w, r, redirectURI, state, "access_denied", "The user denied access")
			return
		}

		givenName, familyName := SplitName(name)

		code := randomCode()
		entry := &CodeEntry{
			Sub:                 DeterministicSub(email),
			Email:               email,
			Name:                name,
			Nonce:               r.FormValue("nonce"),
			Scope:               r.FormValue("scope"),
			ClientID:            r.FormValue("client_id"),
			RedirectURI:         redirectURI,
			ResponseMode:        responseMode,
			CodeChallenge:       r.FormValue("code_challenge"),
			CodeChallengeMethod: r.FormValue("code_challenge_method"),
		}
		_ = givenName
		_ = familyName
		store.SaveCode(code, entry)

		u, _ := url.Parse(redirectURI)
		q := u.Query()
		q.Set("code", code)
		q.Set("state", state)
		u.RawQuery = q.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	}
}

func handleToken(publicURL string, keys *KeyPair, store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed.")
			return
		}
		r.ParseForm()

		clientID := r.FormValue("client_id")
		clientSecret := r.FormValue("client_secret")
		if basicClientID, basicClientSecret, ok := parseBasicClientAuth(r); ok {
			if clientID == "" {
				clientID = basicClientID
			}
			if clientSecret == "" {
				clientSecret = basicClientSecret
			}
		}

		grantType := r.FormValue("grant_type")
		if grantType != "authorization_code" {
			jsonError(w, http.StatusBadRequest, "unsupported_grant_type", "Only authorization_code is supported.")
			return
		}
		if clientID == "" {
			jsonError(w, http.StatusBadRequest, "invalid_request", "client_id is required.")
			return
		}
		if clientSecret == "" {
			jsonError(w, http.StatusBadRequest, "invalid_request", "client_secret is required.")
			return
		}
		if r.FormValue("redirect_uri") == "" {
			jsonError(w, http.StatusBadRequest, "invalid_request", "redirect_uri is required.")
			return
		}

		code := r.FormValue("code")
		if code == "" {
			jsonError(w, http.StatusBadRequest, "invalid_grant", "Code is required.")
			return
		}

		entry, ok := store.LoadCode(code)
		if !ok {
			jsonError(w, http.StatusBadRequest, "invalid_grant", "Code not found or already redeemed.")
			return
		}
		if entry.ClientID != "" && entry.ClientID != clientID {
			jsonError(w, http.StatusBadRequest, "invalid_grant", "client_id does not match the authorization code.")
			return
		}

		// RFC 6749 4.1.3: redirect_uri MUST match the one used in authorization
		if !MatchRedirectURI(entry.RedirectURI, r.FormValue("redirect_uri")) {
			jsonError(w, http.StatusBadRequest, "invalid_grant", "redirect_uri does not match the authorization request.")
			return
		}

		if entry.ResponseMode == "token_error" {
			jsonError(w, http.StatusInternalServerError, "server_error", "Internal server error.")
			return
		}

		// PKCE verification
		if entry.CodeChallenge != "" {
			verifier := r.FormValue("code_verifier")
			if verifier == "" {
				jsonError(w, http.StatusBadRequest, "invalid_grant", "code_verifier is required.")
				return
			}
			if !verifyPKCE(entry.CodeChallenge, entry.CodeChallengeMethod, verifier) {
				jsonError(w, http.StatusBadRequest, "invalid_grant", "PKCE verification failed.")
				return
			}
		}

		entry, ok = store.ConsumeCode(code)
		if !ok {
			jsonError(w, http.StatusBadRequest, "invalid_grant", "Code not found or already redeemed.")
			return
		}

		accessToken := "ya29." + randomCode()
		store.SaveToken(accessToken, entry)

		givenName, familyName := SplitName(entry.Name)

		claims := map[string]any{
			"iss":            publicURL,
			"sub":            entry.Sub,
			"aud":            entry.ClientID,
			"azp":            entry.ClientID,
			"email":          entry.Email,
			"email_verified": true,
			"name":           entry.Name,
			"given_name":     givenName,
			"family_name":    familyName,
			"picture":        "",
		}
		if entry.Nonce != "" {
			claims["nonce"] = entry.Nonce
		}

		idToken, err := keys.SignIDToken(claims)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "server_error", "Failed to sign id_token.")
			return
		}

		// OIDC Core 3.1.3.3: MUST include Cache-Control: no-store
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": accessToken,
			"expires_in":   3920,
			"token_type":   "Bearer",
			"scope":        entry.Scope,
			"id_token":     idToken,
		})
	}
}

func handleUserinfo(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			jsonError(w, http.StatusUnauthorized, "invalid_token", "The access token is invalid.")
			return
		}
		accessToken := strings.TrimPrefix(auth, "Bearer ")

		entry, ok := store.LoadCodeByToken(accessToken)
		if !ok {
			jsonError(w, http.StatusUnauthorized, "invalid_token", "The access token is invalid.")
			return
		}

		if entry.ResponseMode == "userinfo_error" {
			jsonError(w, http.StatusInternalServerError, "server_error", "Internal server error.")
			return
		}

		givenName, familyName := SplitName(entry.Name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sub":            entry.Sub,
			"name":           entry.Name,
			"given_name":     givenName,
			"family_name":    familyName,
			"picture":        "",
			"email":          entry.Email,
			"email_verified": true,
		})
	}
}

func handleCerts(keys *KeyPair) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		json.NewEncoder(w).Encode(keys.JWKS())
	}
}

func handleHealth(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": version,
		})
	}
}

func randomCode() string {
	b := make([]byte, 24)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func verifyPKCE(challenge, method, verifier string) bool {
	switch method {
	case "S256":
		h := sha256.Sum256([]byte(verifier))
		computed := base64.RawURLEncoding.EncodeToString(h[:])
		return computed == challenge
	case "plain", "":
		return verifier == challenge
	default:
		return false
	}
}

func parseBasicClientAuth(r *http.Request) (string, string, bool) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		return "", "", false
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
	if err != nil {
		return "", "", false
	}

	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}

func redirectWithError(w http.ResponseWriter, r *http.Request, redirectURI, state, errCode, desc string) {
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("error", errCode)
	q.Set("error_description", desc)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func jsonError(w http.ResponseWriter, status int, errCode, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": desc,
	})
}
