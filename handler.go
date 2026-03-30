package main

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
func RegisterHandlers(mux *http.ServeMux, publicURL string, keys *KeyPair, store *Store) {
	mux.HandleFunc("GET /.well-known/openid-configuration", handleDiscovery(publicURL))
	mux.HandleFunc("GET /o/oauth2/v2/auth", handleAuthorizeGET())
	mux.HandleFunc("POST /o/oauth2/v2/auth", handleAuthorizePOST(store))
	mux.HandleFunc("POST /token", handleToken(publicURL, keys, store))
	mux.HandleFunc("GET /v1/userinfo", handleUserinfo(store))
	mux.HandleFunc("GET /oauth2/v3/certs", handleCerts(keys))
	mux.HandleFunc("GET /health", handleHealth())
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

		email := "alice@example.com"
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

		responseMode := r.FormValue("response_mode")
		if responseMode == "" {
			responseMode = "normal"
		}

		// Deny mode: redirect with error, no code
		if responseMode == "deny" {
			u, _ := url.Parse(redirectURI)
			q := u.Query()
			q.Set("error", "access_denied")
			q.Set("error_description", "The user denied access")
			q.Set("state", state)
			u.RawQuery = q.Encode()
			http.Redirect(w, r, u.String(), http.StatusFound)
			return
		}

		code := randomCode()
		entry := &CodeEntry{
			Sub:                 DeterministicSub(email),
			Email:               email,
			Name:                name,
			Nonce:               r.FormValue("nonce"),
			Scope:               r.FormValue("scope"),
			ClientID:            r.FormValue("client_id"),
			ResponseMode:        responseMode,
			CodeChallenge:       r.FormValue("code_challenge"),
			CodeChallengeMethod: r.FormValue("code_challenge_method"),
		}
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

		accessToken := "ya29." + randomCode()
		store.SaveToken(accessToken, code)

		claims := map[string]any{
			"iss":            publicURL,
			"sub":            entry.Sub,
			"aud":            entry.ClientID,
			"email":          entry.Email,
			"email_verified": true,
			"name":           entry.Name,
			"given_name":     entry.Name,
			"family_name":    "",
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sub":            entry.Sub,
			"name":           entry.Name,
			"given_name":     entry.Name,
			"family_name":    "",
			"picture":        "",
			"email":          entry.Email,
			"email_verified": true,
		})
	}
}

func handleCerts(keys *KeyPair) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keys.JWKS())
	}
}

func handleHealth() http.HandlerFunc {
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

func jsonError(w http.ResponseWriter, status int, errCode, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": desc,
	})
}
