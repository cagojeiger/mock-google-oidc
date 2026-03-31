package oidc

import (
	"errors"
	"strings"
	"time"
)

// ValidateCodeTTL checks if an authorization code is still valid.
// Returns true if the code has not expired.
func ValidateCodeTTL(createdAt, now time.Time, ttl time.Duration) bool {
	return now.Sub(createdAt) < ttl
}

// MatchRedirectURI performs RFC 6749 simple string comparison.
// If stored is empty (not registered at auth time), validation is skipped.
func MatchRedirectURI(stored, provided string) bool {
	if stored == "" {
		return true
	}
	return stored == provided
}

// ValidateResponseType checks that response_type is "code".
// Only Authorization Code Flow is supported.
func ValidateResponseType(responseType string) error {
	if responseType != "code" {
		return errors.New("unsupported_response_type")
	}
	return nil
}

// RequireOpenIDScope checks that the scope string contains "openid" as a distinct token.
func RequireOpenIDScope(scope string) error {
	for _, s := range strings.Fields(scope) {
		if s == "openid" {
			return nil
		}
	}
	return errors.New("openid scope is required")
}

// ValidatePrompt checks the prompt parameter per OIDC Core 3.1.2.1.
// This mock always requires user interaction, so prompt=none returns an error.
func ValidatePrompt(prompt string) error {
	if prompt == "" {
		return nil
	}
	fields := strings.Fields(prompt)
	for _, f := range fields {
		if f == "none" {
			if len(fields) > 1 {
				return errors.New("none must not be combined with other prompt values")
			}
			return errors.New("login_required")
		}
	}
	return nil
}

// SplitName splits a full name into given_name and family_name.
// First token becomes given_name, the rest becomes family_name.
func SplitName(fullName string) (givenName, familyName string) {
	parts := strings.SplitN(strings.TrimSpace(fullName), " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", ""
	}
	givenName = parts[0]
	if len(parts) == 2 {
		familyName = parts[1]
	}
	return
}
