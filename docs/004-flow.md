# Spec 004: 전체 흐름

## Google OIDC 호환 플로우

mock-google-oidc는 Google OIDC Authorization Code + PKCE 플로우를 구현한다.
유일한 차이는 로그인 화면에서 패스워드가 없다는 것.

## 시퀀스 다이어그램

```mermaid
sequenceDiagram
    participant U as 브라우저
    participant App as Relying Party
    participant IDP as mock-google-oidc

    Note over U,IDP: 1. 로그인 시작
    U->>App: 로그인 클릭
    App->>App: PKCE 생성 (code_verifier → code_challenge)
    App->>U: 302 → IDP /o/oauth2/v2/auth?client_id=...&code_challenge=...&state=...

    Note over U,IDP: 2. 로그인 화면
    U->>IDP: GET /o/oauth2/v2/auth?...
    IDP->>U: 200 로그인 화면 (이메일/이름 입력)

    Note over U,IDP: 3. 사용자 인증 (패스워드 없음)
    U->>U: 이메일/이름 입력 + Login 클릭
    U->>IDP: POST /o/oauth2/v2/auth (email, name, code_challenge, ...)
    IDP->>IDP: code 생성 + code_challenge 저장
    IDP->>U: 302 → redirect_uri?code={code}&state={state}

    Note over U,IDP: 4. 토큰 교환 (+ PKCE 검증)
    U->>App: GET /callback?code={code}&state={state}
    App->>IDP: POST /token (code, code_verifier, client_id, client_secret)
    IDP->>IDP: PKCE 검증: base64url(sha256(verifier)) == challenge
    IDP-->>App: {"access_token", "id_token", "expires_in", "token_type"}

    Note over U,IDP: 5. 사용자 정보 조회
    App->>IDP: GET /v1/userinfo (Authorization: Bearer ...)
    IDP-->>App: {"sub", "email", "name", "email_verified": true}

    Note over U,IDP: 6. 완료
    App->>U: 로그인 완료
```

## Google 플로우와 비교

| 단계 | Google | mock-google-oidc | 차이 |
|------|--------|-----------------|------|
| Authorization URL | `accounts.google.com/o/oauth2/v2/auth` | `localhost:8082/o/oauth2/v2/auth` | 호스트 |
| 로그인 화면 | 이메일 → 패스워드 → 동의 (3단계) | 이메일 + 이름 → Login (1단계) | **유일한 UX 차이** |
| PKCE | S256, plain 지원 | S256, plain 지원 | 없음 |
| Callback | `redirect_uri?code=...&state=...` | 동일 | 없음 |
| Token 요청 | POST + code + code_verifier | 동일 형식 | 없음 |
| Token 응답 | `{access_token, id_token, ...}` | 동일 형식 | 없음 |
| id_token | RS256 JWT | RS256 JWT | 키만 다름 |
| UserInfo | `{sub, email, name, ...}` | 동일 형식 | 없음 |
| refresh_token | 지원 | **미지원** | |
| client_secret 검증 | 검증함 | **검증 안 함** | |
| HTTPS | 필수 | HTTP (로컬) | |

## sub 규칙

```
sub = fmt.Sprintf("%x", sha256("alice@example.com")[:10])
    = "e3b0c44298fc1c149afb"  // 20자리 hex
```

| 동작 | 설명 |
|------|------|
| 같은 이메일로 재로그인 | 같은 sub → Relying Party가 기존 유저로 인식 |
| 다른 이메일로 로그인 | 다른 sub → Relying Party가 새 유저로 인식 |

## 에러 플로우

### Deny
```
IDP → 302 → redirect_uri?error=access_denied&error_description=The+user+denied+access&state=...
```

### Token Error
```
Relying Party → POST /token → 500 {"error": "server_error"}
```

### Userinfo Error
```
Relying Party → POST /token → 200 정상
Relying Party → GET /v1/userinfo → 500 {"error": "server_error"}
```

## 환경 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `LISTEN_ADDR` | `:8082` | 서버 바인딩 주소 |
| `PUBLIC_URL` | `http://localhost:8082` | issuer, discovery URL |
