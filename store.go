package main

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
	ResponseMode        string // normal, deny, token_error, userinfo_error
	CodeChallenge       string // PKCE code_challenge (S256 or plain)
	CodeChallengeMethod string // "S256" or "plain"
	CreatedAt           time.Time
}

// Store is an in-memory store for authorization codes and access tokens.
type Store struct {
	codes  sync.Map // code string → *CodeEntry
	tokens sync.Map // access_token string → code string
}

func NewStore() *Store {
	return &Store{}
}

// SaveCode stores a CodeEntry keyed by the authorization code.
func (s *Store) SaveCode(code string, entry *CodeEntry) {
	s.codes.Store(code, entry)
}

// LoadCode retrieves a CodeEntry by authorization code.
func (s *Store) LoadCode(code string) (*CodeEntry, bool) {
	v, ok := s.codes.Load(code)
	if !ok {
		return nil, false
	}
	return v.(*CodeEntry), true
}

// SaveToken maps an access token to its originating authorization code.
func (s *Store) SaveToken(accessToken, code string) {
	s.tokens.Store(accessToken, code)
}

// LoadCodeByToken resolves an access token to its CodeEntry.
func (s *Store) LoadCodeByToken(accessToken string) (*CodeEntry, bool) {
	v, ok := s.tokens.Load(accessToken)
	if !ok {
		return nil, false
	}
	return s.LoadCode(v.(string))
}

// DeterministicSub generates a stable sub from an email address.
// Same email always produces the same sub.
func DeterministicSub(email string) string {
	h := sha256.Sum256([]byte(email))
	return fmt.Sprintf("%x", h[:10]) // 20 hex chars
}
