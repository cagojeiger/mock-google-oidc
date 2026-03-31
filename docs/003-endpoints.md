## Spec 003: 엔드포인트 계약

이 문서는 mock-google-oidc가 외부 클라이언트와 맺는 HTTP 계약을 정의한다.
기준은 OIDC/OAuth 표준이지만, 경로는 Google 호환성을 우선한다.

## 전체 구조

```text
[Discovery]
   |
   v
[Authorization Endpoint]
   |
   v
[Token Endpoint] ----> [JWKS]
   |
   +----> [ID Token]
   |
   `----> [UserInfo Endpoint]
```

## 엔드포인트 목록

| Method | Path | 역할 | 비고 |
| --- | --- | --- | --- |
| `GET` | `/.well-known/openid-configuration` | OIDC Discovery | 표준 discovery |
| `GET` | `/o/oauth2/v2/auth` | 로그인 화면 | Google 스타일 authorize 경로 |
| `POST` | `/o/oauth2/v2/auth` | 로그인 처리 + code 발급 | mock 로그인 폼 제출 |
| `POST` | `/token` | code -> token 교환 | OAuth token endpoint |
| `GET` | `/v1/userinfo` | 사용자 정보 반환 | Google 스타일 userinfo 경로 |
| `GET` | `/oauth2/v3/certs` | JWKS 공개키 제공 | Google 스타일 certs 경로 |
| `GET` | `/health` | 상태 확인 | 구현 점검용 |

## 1. Discovery

경로:

```text
GET /.well-known/openid-configuration
```

필수 의미:
- OIDC 클라이언트가 issuer와 endpoint를 자동으로 찾을 수 있어야 한다.
- Google 호환 경로를 discovery에 실어준다.

핵심 반환 필드:

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
  "token_endpoint_auth_methods_supported": ["client_secret_post", "client_secret_basic"],
  "claims_supported": ["aud", "email", "email_verified", "exp", "family_name", "given_name", "iat", "iss", "name", "picture", "sub"],
  "code_challenge_methods_supported": ["plain", "S256"],
  "grant_types_supported": ["authorization_code"]
}
```

제약:
- `refresh_token`은 광고하지 않는다.
- JWKS는 RS256 검증에 필요한 공개키만 제공한다.

## 2. Authorization Endpoint

경로:

```text
GET  /o/oauth2/v2/auth
POST /o/oauth2/v2/auth
```

목적:
- 브라우저 기반 Google 로그인 시작점을 흉내낸다.
- 실제 패스워드 인증 대신 이메일/이름 입력으로 바로 code를 발급한다.

### GET /o/oauth2/v2/auth

주요 query 파라미터:

| 파라미터 | 요구사항 | 설명 |
| --- | --- | --- |
| `redirect_uri` | 필수 | callback URL |
| `state` | 필수 | 그대로 돌려줄 값 |
| `response_type` | `code`만 허용 | 그 외는 `unsupported_response_type` |
| `scope` | `openid` 포함 필수 | 없으면 `invalid_scope` |
| `client_id` | 선택 | token 단계 비교용 저장 |
| `nonce` | 선택 | id_token에 반영 |
| `login_hint` | 선택 | email 기본값으로 사용 |
| `code_challenge` | 선택 | PKCE challenge |
| `code_challenge_method` | 선택 | `S256` 또는 `plain` |
| `prompt` | `none` 거부 | mock은 항상 사용자 상호작용 필요 |

동작:

```text
request validation
   -> 로그인 HTML 렌더링
   -> hidden field로 OAuth/OIDC 파라미터 유지
```

### POST /o/oauth2/v2/auth

입력:

| 필드 | 요구사항 | 설명 |
| --- | --- | --- |
| `redirect_uri` | 필수 | hidden field |
| `state` | 필수 | hidden field |
| `email` | 필수 | mock 사용자 식별자 |
| `name` | 필수 | 표시 이름 |
| `client_id` | 선택 | code에 저장 |
| `nonce` | 선택 | code에 저장 |
| `scope` | 선택 | token 응답용 저장 |
| `code_challenge` | 선택 | PKCE 검증용 저장 |
| `code_challenge_method` | 선택 | PKCE 메서드 |
| `response_mode` | 선택 | mock 에러 제어 |

성공 응답:

```text
302 {redirect_uri}?code={code}&state={state}
```

response_mode:

| 값 | 의미 |
| --- | --- |
| `normal` | 정상 로그인 |
| `deny` | `access_denied` redirect |
| `token_error` | token endpoint에서 500 유도 |
| `userinfo_error` | userinfo endpoint에서 500 유도 |

## 3. Token Endpoint

경로:

```text
POST /token
```

지원하는 인증 방식:
- `client_secret_post`
- `client_secret_basic`

필수 의미:
- code는 1회용이다.
- `redirect_uri`는 authorization 단계와 문자열 완전 일치해야 한다.
- PKCE를 사용한 경우 `code_verifier`를 검증해야 한다.

주요 form 파라미터:

| 필드 | 요구사항 | 설명 |
| --- | --- | --- |
| `grant_type` | `authorization_code`만 허용 | 다른 값은 거부 |
| `code` | 필수 | 미사용 code |
| `client_id` | 필수 | authorize에 저장된 값과 비교 |
| `client_secret` | 필수 | 값 자체는 엄격 검증 안 함 |
| `redirect_uri` | 필수 | authorize 요청과 exact match |
| `code_verifier` | PKCE 시 필수 | challenge와 비교 |

PKCE 검증:

```text
S256  -> BASE64URL(SHA256(code_verifier)) == code_challenge
plain -> code_verifier == code_challenge
```

정상 응답:

```json
{
  "access_token": "ya29.xxxxx",
  "expires_in": 3920,
  "token_type": "Bearer",
  "scope": "openid email profile",
  "id_token": "eyJ..."
}
```

응답 헤더:
- `Cache-Control: no-store`
- `Pragma: no-cache`
- `Content-Type: application/json`

주요 에러:
- `unsupported_grant_type`
- `invalid_request`
- `invalid_grant`
- `server_error`

## 4. ID Token

token endpoint가 반환하는 `id_token`은 RS256 JWT이다.

주요 claims:

```json
{
  "iss": "http://localhost:9082",
  "sub": "c160f8cc69a4f0bf2b0c",
  "aud": "client_id_value",
  "azp": "client_id_value",
  "exp": 1353604926,
  "iat": 1353601026,
  "nonce": "abc123",
  "email": "alice@example.com",
  "email_verified": true,
  "name": "Alice Kim",
  "given_name": "Alice",
  "family_name": "Kim",
  "picture": ""
}
```

`sub` 규칙:

```text
sub = SHA256(email)[:10] -> 20자리 hex
같은 이메일 -> 같은 sub
다른 이메일 -> 다른 sub
```

## 5. UserInfo Endpoint

경로:

```text
GET /v1/userinfo
Authorization: Bearer {access_token}
```

정상 응답:

```json
{
  "sub": "c160f8cc69a4f0bf2b0c",
  "name": "Alice Kim",
  "given_name": "Alice",
  "family_name": "Kim",
  "picture": "",
  "email": "alice@example.com",
  "email_verified": true
}
```

보장:
- `sub`는 같은 flow에서 발급된 `id_token.sub`와 같아야 한다.
- 토큰이 없거나 잘못되면 `401 invalid_token`을 반환한다.

## 6. JWKS Endpoint

경로:

```text
GET /oauth2/v3/certs
```

응답 예시:

```json
{
  "keys": [{
    "kty": "RSA",
    "alg": "RS256",
    "use": "sig",
    "kid": "test-idp-key-1",
    "n": "{base64url}",
    "e": "{base64url}"
  }]
}
```

보장:
- `id_token` 헤더의 `kid`와 일치하는 키를 제공한다.
- OIDC 클라이언트가 JWKS를 읽고 RS256 서명을 검증할 수 있어야 한다.

## 7. Health Endpoint

경로:

```text
GET /health
```

응답:

```json
{
  "status": "ok",
  "version": "dev"
}
