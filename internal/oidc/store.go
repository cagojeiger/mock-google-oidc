package oidc

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// CodeEntry holds the data associated with an authorization code.
type CodeEntry struct {
	Sub                 string
	Email               string
	Name                string
	Nonce               string
	Scope               string
	ClientID            string
	RedirectURI         string
	ResponseMode        string // normal, deny, token_error, userinfo_error
	CodeChallenge       string // PKCE code_challenge (S256 or plain)
	CodeChallengeMethod string // "S256" or "plain"
	CreatedAt           time.Time
}

// Store is an in-memory store for authorization codes and access tokens.
type Store struct {
	codes  sync.Map // code string → *CodeEntry
	tokens sync.Map // access_token string → *CodeEntry
}

func NewStore() *Store {
	return &Store{}
}

// SaveCode stores a CodeEntry keyed by the authorization code.
func (s *Store) SaveCode(code string, entry *CodeEntry) {
	entry.CreatedAt = time.Now()
	s.codes.Store(code, entry)
}

// LoadCode retrieves a CodeEntry by authorization code (without consuming it).
func (s *Store) LoadCode(code string) (*CodeEntry, bool) {
	v, ok := s.codes.Load(code)
	if !ok {
		return nil, false
	}
	return v.(*CodeEntry), true
}

// ConsumeCode retrieves and deletes a CodeEntry. Returns false if not found or expired.
// Authorization codes are single-use per OAuth spec.
func (s *Store) ConsumeCode(code string) (*CodeEntry, bool) {
	v, ok := s.codes.LoadAndDelete(code)
	if !ok {
		return nil, false
	}
	entry := v.(*CodeEntry)
	if !ValidateCodeTTL(entry.CreatedAt, time.Now(), 10*time.Minute) {
		return nil, false
	}
	return entry, true
}

// SaveToken maps an access token directly to a CodeEntry.
func (s *Store) SaveToken(accessToken string, entry *CodeEntry) {
	s.tokens.Store(accessToken, entry)
}

// LoadCodeByToken resolves an access token to its CodeEntry.
func (s *Store) LoadCodeByToken(accessToken string) (*CodeEntry, bool) {
	v, ok := s.tokens.Load(accessToken)
	if !ok {
		return nil, false
	}
	return v.(*CodeEntry), true
}

// DeterministicSub generates a stable sub from an email address.
// Same email always produces the same sub.
func DeterministicSub(email string) string {
	h := sha256.Sum256([]byte(email))
	return fmt.Sprintf("%x", h[:10]) // 20 hex chars
}
