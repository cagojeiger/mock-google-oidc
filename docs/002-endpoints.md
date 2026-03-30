# Spec 002: 엔드포인트

## 엔드포인트 목록

Google OIDC와 호환되는 경로와 형식을 사용한다.

| Method | Path | Google 대응 | 설명 |
|--------|------|------------|------|
| GET | `/.well-known/openid-configuration` | `accounts.google.com/.well-known/openid-configuration` | OIDC Discovery |
| GET | `/o/oauth2/v2/auth` | `accounts.google.com/o/oauth2/v2/auth` | 로그인 화면 표시 |
| POST | `/o/oauth2/v2/auth` | (Google은 내부 처리) | 로그인 폼 제출 |
| POST | `/token` | `oauth2.googleapis.com/token` | code → token 교환 |
| GET | `/v1/userinfo` | `openidconnect.googleapis.com/v1/userinfo` | 사용자 정보 |
| GET | `/oauth2/v3/certs` | `www.googleapis.com/oauth2/v3/certs` | JWKS 공개키 |
| GET | `/health` | — | 헬스 체크 |

---

## GET /.well-known/openid-configuration

OIDC Discovery 문서.

**응답:**

```json
{
  "issuer": "http://localhost:9082",
  "authorization_endpoint": "http://localhost:9082/o/oauth2/v2/auth",
  "token_endpoint": "http://localhost:9082/token",
  "userinfo_endpoint": "http://localhost:9082/v1/userinfo",
  "jwks_uri": "http://localhost:9082/oauth2/v3/certs",
  "response_types_supported": ["code"],
  "response_modes_supported": ["query"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"],
  "scopes_supported": ["openid", "email", "profile"],
  "token_endpoint_auth_methods_supported": ["client_secret_post"],
  "claims_supported": ["aud", "email", "email_verified", "exp", "family_name", "given_name", "iat", "iss", "name", "picture", "sub"],
  "code_challenge_methods_supported": ["plain", "S256"],
  "grant_types_supported": ["authorization_code"]
}
```

주의:
- `grant_types_supported`에 `refresh_token` 없음 (미지원)
- `token_endpoint_auth_methods_supported`에 `client_secret_post`만 (basic 미지원)
- `code_challenge_methods_supported`에 `plain`과 `S256` 모두 광고하며 실제 검증함

---

## GET /o/oauth2/v2/auth — 로그인 화면

**Query Parameters:**

| 파라미터 | 필수 | 설명 |
|---------|------|------|
| `redirect_uri` | **필수** | callback URL |
| `state` | **필수** | CSRF 방지 토큰 |
| `client_id` | 선택 | 저장 (검증 안 함) |
| `scope` | 선택 | 저장 (표시용) |
| `response_type` | 선택 | 항상 `code` |
| `nonce` | 선택 | id_token에 포함 |
| `code_challenge` | 선택 | PKCE challenge |
| `code_challenge_method` | 선택 | `S256` 또는 `plain` |
| `login_hint` | 선택 | email 기본값으로 사용 |
| `access_type` | 선택 | 무시 |
| `prompt` | 선택 | 무시 |
| `hd` | 선택 | 무시 |

**에러:**
- `redirect_uri` 또는 `state` 누락 → 400

---

## POST /o/oauth2/v2/auth — 로그인 처리

**Form Body:**

| 필드 | 필수 | 설명 |
|------|------|------|
| `redirect_uri` | 필수 | hidden field |
| `state` | 필수 | hidden field |
| `email` | 필수 | 사용자 입력 |
| `name` | 필수 | 사용자 입력 (HTML required) |
| `nonce` | 선택 | hidden field |
| `scope` | 선택 | hidden field |
| `client_id` | 선택 | hidden field |
| `code_challenge` | 선택 | hidden field (PKCE) |
| `code_challenge_method` | 선택 | hidden field (PKCE) |
| `response_mode` | 선택 | 기본 `normal` |

**응답 모드:**

| response_mode | 동작 |
|---------------|------|
| `normal` (기본) | code 생성 → `302 redirect_uri?code={code}&state={state}` |
| `deny` | `302 redirect_uri?error=access_denied&error_description=The+user+denied+access&state={state}` |
| `token_error` | code 생성 (marker) → 정상 리다이렉트, POST /token에서 500 |
| `userinfo_error` | code 생성 (marker) → 정상 리다이렉트, GET /userinfo에서 500 |

**sub 생성 규칙:**
```
sub = fmt.Sprintf("%x", sha256(email)[:10])
```
- 20자리 hex 문자열 (예: `c160f8cc69a4f0bf2b0c`)
- 같은 이메일 → 항상 같은 sub
- 다른 이메일 → 다른 sub

---

## POST /token — 토큰 교환

**Content-Type:** `application/x-www-form-urlencoded`

**Request Body:**

| 필드 | 필수 | 설명 |
|------|------|------|
| `grant_type` | 필수 | `authorization_code` (다른 값 → 400 `unsupported_grant_type`) |
| `code` | 필수 | authorization code (1회용) |
| `client_id` | 필수 | 클라이언트 ID (검증 안 함) |
| `client_secret` | 필수 | 클라이언트 시크릿 (검증 안 함) |
| `redirect_uri` | 필수 | (검증 안 함) |
| `code_verifier` | PKCE 시 필수 | PKCE verifier |

**PKCE 검증:**
- code에 `code_challenge`가 저장되어 있으면 `code_verifier` 필수
- S256: `base64url(sha256(code_verifier)) == code_challenge`
- plain: `code_verifier == code_challenge`
- 불일치 시 400 `invalid_grant`

**code 1회용:**
- code는 토큰 교환 시 소비(삭제)됨
- 같은 code로 재요청 시 400 `invalid_grant`

**응답:**

정상 (200):
```json
{
  "access_token": "ya29.xxxxxxxxxxxx",
  "expires_in": 3920,
  "token_type": "Bearer",
  "scope": "openid email profile",
  "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6Ii4uLiJ9..."
}
```

code 미존재/재사용 (400):
```json
{
  "error": "invalid_grant",
  "error_description": "Code not found or already redeemed."
}
```

grant_type 오류 (400):
```json
{
  "error": "unsupported_grant_type",
  "error_description": "Only authorization_code is supported."
}
```

PKCE 실패 (400):
```json
{
  "error": "invalid_grant",
  "error_description": "PKCE verification failed."
}
```

token_error 모드 (500):
```json
{
  "error": "server_error",
  "error_description": "Internal server error."
}
```

**id_token JWT 클레임:**
```json
{
  "iss": "http://localhost:9082",
  "sub": "c160f8cc69a4f0bf2b0c",
  "aud": "client_id_value",
  "exp": 1353604926,
  "iat": 1353601026,
  "nonce": "abc123",
  "email": "alice@example.com",
  "email_verified": true,
  "name": "Alice",
  "given_name": "Alice",
  "family_name": "",
  "picture": ""
}
```

---

## GET /v1/userinfo — 사용자 정보

**Headers:**
```
Authorization: Bearer {access_token}
```

**정상 응답 (200):**
```json
{
  "sub": "c160f8cc69a4f0bf2b0c",
  "name": "Alice",
  "given_name": "Alice",
  "family_name": "",
  "picture": "",
  "email": "alice@example.com",
  "email_verified": true
}
```

토큰 없음/유효하지 않음 (401):
```json
{
  "error": "invalid_token",
  "error_description": "The access token is invalid."
}
```

---

## GET /oauth2/v3/certs — JWKS

```json
{
  "keys": [
    {
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "kid": "test-idp-key-1",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

서버 시작 시 RSA 2048비트 키 쌍 생성. 재시작 시 키 변경.

---

## GET /health

```json
{
  "status": "ok",
  "version": "v0.1.0"
}
```
