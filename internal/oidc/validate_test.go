package oidc

import (
	"testing"
	"time"
)

// --- ValidateCodeTTL ---

func TestValidateCodeTTL_Valid(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 5, 0, 0, time.UTC)
	createdAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) // 5 min ago
	ttl := 10 * time.Minute

	if !ValidateCodeTTL(createdAt, now, ttl) {
		t.Error("code created 5m ago with 10m TTL should be valid")
	}
}

func TestValidateCodeTTL_Expired(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 11, 0, 0, time.UTC)
	createdAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) // 11 min ago
	ttl := 10 * time.Minute

	if ValidateCodeTTL(createdAt, now, ttl) {
		t.Error("code created 11m ago with 10m TTL should be expired")
	}
}

func TestValidateCodeTTL_ExactBoundary(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 10, 0, 0, time.UTC)
	createdAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) // exactly 10 min
	ttl := 10 * time.Minute

	if ValidateCodeTTL(createdAt, now, ttl) {
		t.Error("code at exact TTL boundary should be expired")
	}
}

func TestValidateCodeTTL_JustCreated(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	createdAt := now
	ttl := 10 * time.Minute

	if !ValidateCodeTTL(createdAt, now, ttl) {
		t.Error("code just created should be valid")
	}
}

// --- MatchRedirectURI ---

func TestMatchRedirectURI_ExactMatch(t *testing.T) {
	if !MatchRedirectURI("http://localhost:9090/callback", "http://localhost:9090/callback") {
		t.Error("identical URIs should match")
	}
}

func TestMatchRedirectURI_DifferentPath(t *testing.T) {
	if MatchRedirectURI("http://localhost:9090/callback", "http://localhost:9090/other") {
		t.Error("different paths should not match")
	}
}

func TestMatchRedirectURI_TrailingSlash(t *testing.T) {
	if MatchRedirectURI("http://localhost:9090/callback", "http://localhost:9090/callback/") {
		t.Error("trailing slash difference should not match (strict string comparison)")
	}
}

func TestMatchRedirectURI_DifferentScheme(t *testing.T) {
	if MatchRedirectURI("http://localhost/cb", "https://localhost/cb") {
		t.Error("different schemes should not match")
	}
}

func TestMatchRedirectURI_ExtraQueryParam(t *testing.T) {
	if MatchRedirectURI("http://localhost/cb", "http://localhost/cb?extra=1") {
		t.Error("extra query params should not match")
	}
}

func TestMatchRedirectURI_CaseSensitive(t *testing.T) {
	if MatchRedirectURI("http://localhost/Callback", "http://localhost/callback") {
		t.Error("path case difference should not match")
	}
}

func TestMatchRedirectURI_BothEmpty(t *testing.T) {
	if !MatchRedirectURI("", "") {
		t.Error("both empty should match")
	}
}

func TestMatchRedirectURI_StoredEmpty(t *testing.T) {
	if !MatchRedirectURI("", "http://localhost/cb") {
		t.Error("empty stored URI should skip validation (match)")
	}
}

// --- ValidateResponseType ---

func TestValidateResponseType_Code(t *testing.T) {
	if err := ValidateResponseType("code"); err != nil {
		t.Errorf("response_type=code should be valid, got: %v", err)
	}
}

func TestValidateResponseType_Token(t *testing.T) {
	if err := ValidateResponseType("token"); err == nil {
		t.Error("response_type=token should be rejected (implicit flow not supported)")
	}
}

func TestValidateResponseType_IDToken(t *testing.T) {
	if err := ValidateResponseType("id_token"); err == nil {
		t.Error("response_type=id_token should be rejected")
	}
}

func TestValidateResponseType_Empty(t *testing.T) {
	if err := ValidateResponseType(""); err == nil {
		t.Error("empty response_type should be rejected")
	}
}

func TestValidateResponseType_CodeIDToken(t *testing.T) {
	if err := ValidateResponseType("code id_token"); err == nil {
		t.Error("hybrid response_type should be rejected")
	}
}

// --- RequireOpenIDScope ---

func TestRequireOpenIDScope_Present(t *testing.T) {
	if err := RequireOpenIDScope("openid email profile"); err != nil {
		t.Errorf("scope containing openid should be valid, got: %v", err)
	}
}

func TestRequireOpenIDScope_OnlyOpenID(t *testing.T) {
	if err := RequireOpenIDScope("openid"); err != nil {
		t.Errorf("scope=openid should be valid, got: %v", err)
	}
}

func TestRequireOpenIDScope_Missing(t *testing.T) {
	if err := RequireOpenIDScope("email profile"); err == nil {
		t.Error("scope without openid should be rejected")
	}
}

func TestRequireOpenIDScope_Empty(t *testing.T) {
	if err := RequireOpenIDScope(""); err == nil {
		t.Error("empty scope should be rejected")
	}
}

func TestRequireOpenIDScope_Substring(t *testing.T) {
	if err := RequireOpenIDScope("notopenid email"); err == nil {
		t.Error("openid as substring should not match")
	}
}

func TestRequireOpenIDScope_MiddlePosition(t *testing.T) {
	if err := RequireOpenIDScope("email openid profile"); err != nil {
		t.Errorf("openid in middle should be valid, got: %v", err)
	}
}

// --- ValidatePrompt ---

func TestValidatePrompt_Empty(t *testing.T) {
	if err := ValidatePrompt(""); err != nil {
		t.Errorf("empty prompt should be valid, got: %v", err)
	}
}

func TestValidatePrompt_Consent(t *testing.T) {
	if err := ValidatePrompt("consent"); err != nil {
		t.Errorf("prompt=consent should be valid, got: %v", err)
	}
}

func TestValidatePrompt_Login(t *testing.T) {
	if err := ValidatePrompt("login"); err != nil {
		t.Errorf("prompt=login should be valid, got: %v", err)
	}
}

func TestValidatePrompt_SelectAccount(t *testing.T) {
	if err := ValidatePrompt("select_account"); err != nil {
		t.Errorf("prompt=select_account should be valid, got: %v", err)
	}
}

func TestValidatePrompt_None(t *testing.T) {
	err := ValidatePrompt("none")
	if err == nil {
		t.Error("prompt=none should return error (mock always requires interaction)")
	}
}

func TestValidatePrompt_NoneWithOthers(t *testing.T) {
	err := ValidatePrompt("none login")
	if err == nil {
		t.Error("prompt=none combined with other values should return error")
	}
}

// --- SplitName ---

func TestSplitName_TwoParts(t *testing.T) {
	given, family := SplitName("Alice Kim")
	if given != "Alice" {
		t.Errorf("given_name: got %q, want %q", given, "Alice")
	}
	if family != "Kim" {
		t.Errorf("family_name: got %q, want %q", family, "Kim")
	}
}

func TestSplitName_SingleName(t *testing.T) {
	given, family := SplitName("Alice")
	if given != "Alice" {
		t.Errorf("given_name: got %q, want %q", given, "Alice")
	}
	if family != "" {
		t.Errorf("family_name: got %q, want %q", family, "")
	}
}

func TestSplitName_ThreeParts(t *testing.T) {
	given, family := SplitName("Alice Bob Kim")
	if given != "Alice" {
		t.Errorf("given_name: got %q, want %q", given, "Alice")
	}
	if family != "Bob Kim" {
		t.Errorf("family_name: got %q, want %q", family, "Bob Kim")
	}
}

func TestSplitName_Empty(t *testing.T) {
	given, family := SplitName("")
	if given != "" {
		t.Errorf("given_name: got %q, want %q", given, "")
	}
	if family != "" {
		t.Errorf("family_name: got %q, want %q", family, "")
	}
}
