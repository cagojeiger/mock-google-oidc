# Test Spec: OIDC Conformance Tests

`coreos/go-oidc` 클라이언트 라이브러리를 사용하여 mock-google-oidc가
표준 OIDC 클라이언트와 호환되는지 검증한다.

**의존성**: `github.com/coreos/go-oidc/v3/oidc`, `golang.org/x/oauth2`
**테스트 파일**: `internal/oidc/conformance_test.go`

---

## 테스트 구성

```
httptest.NewServer → mock-google-oidc 서버 기동
                   ↓
        coreos/go-oidc 클라이언트가 서버에 접근
                   ↓
        Discovery → JWKS → Token 검증 → UserInfo
```

모든 테스트는 `httptest.Server`를 사용하여 인프로세스로 실행된다.
외부 네트워크 의존성 없음.

---

## C1: Discovery 파싱

- **목적**: Discovery 문서가 표준 OIDC 클라이언트에서 파싱 가능한지 검증
- **방법**:
  ```go
  provider, err := oidc.NewProvider(ctx, server.URL)
  ```
- **기대**: `err == nil`, `provider != nil`
- **실패 시 의미**: Discovery JSON 형식 또는 필수 필드 누락

## C2: JWKS 로딩

- **목적**: JWKS 엔드포인트에서 공개키를 파싱할 수 있는지 검증
- **방법**:
  ```go
  verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
  ```
- **기대**: verifier 생성 성공
- **실패 시 의미**: JWKS 형식 오류 또는 키 파싱 실패

## C3: id_token 서명 검증

- **목적**: 발급된 id_token이 JWKS의 공개키로 검증 가능한지
- **방법**:
  ```go
  idToken, err := verifier.Verify(ctx, rawIDToken)
  ```
- **기대**: `err == nil`
- **검증 항목** (coreos/go-oidc가 내부적으로 수행):
  - JWT 서명 검증 (RS256)
  - `iss` == issuer URL
  - `aud`에 clientID 포함
  - `exp` > 현재 시각
  - `iat` 존재
- **실패 시 의미**: 서명 알고리즘, issuer, audience, 만료 중 하나 불일치

## C4: id_token Claims 파싱

- **목적**: id_token에서 표준 claims를 추출할 수 있는지
- **방법**:
  ```go
  var claims struct {
      Sub           string `json:"sub"`
      Email         string `json:"email"`
      EmailVerified bool   `json:"email_verified"`
      Name          string `json:"name"`
      GivenName     string `json:"given_name"`
      FamilyName    string `json:"family_name"`
      Nonce         string `json:"nonce"`
  }
  idToken.Claims(&claims)
  ```
- **검증**:
  - `sub` 비어있지 않음
  - `email` 비어있지 않음
  - `email_verified` == true
  - `name` 비어있지 않음
  - `nonce` == 요청 시 전달한 nonce

## C5: sub 일관성

- **목적**: id_token의 sub와 UserInfo의 sub가 동일한지
- **방법**: C3에서 얻은 sub와 C6에서 얻은 sub 비교
- **기대**: 동일
- **스펙**: Core §5.3.2

## C6: UserInfo 호출

- **목적**: access_token으로 UserInfo 엔드포인트를 호출할 수 있는지
- **방법**:
  ```go
  userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
  ```
- **기대**: `err == nil`
- **검증**:
  - `sub` 비어있지 않음
  - claims 파싱 가능
- **실패 시 의미**: UserInfo 엔드포인트 형식 오류 또는 Bearer 토큰 처리 실패

## C7: 만료 토큰 거부

- **목적**: 만료된 id_token을 verifier가 거부하는지
- **방법**: 만료 시간이 과거인 id_token을 수동 생성 후 검증
  ```go
  _, err := verifier.Verify(ctx, expiredIDToken)
  ```
- **기대**: `err != nil` (토큰 만료 에러)
- **실패 시 의미**: exp claim이 누락되었거나 잘못된 형식

---

## 통합 플로우 테스트

### C-FLOW-01: 전체 Authorization Code Flow

1. `oidc.NewProvider(ctx, issuerURL)` — Discovery
2. `POST /o/oauth2/v2/auth` — 인가 코드 획득
3. `POST /token` — 코드 → 토큰 교환
4. `verifier.Verify(ctx, id_token)` — id_token 검증
5. `provider.UserInfo(ctx, tokenSource)` — UserInfo 호출
6. id_token sub == UserInfo sub — 일관성 확인

**이 테스트가 통과하면**: 표준 OIDC 클라이언트가 우리 provider를 사용하여
전체 인증 플로우를 완료할 수 있음이 증명된다.

### C-FLOW-02: PKCE + Conformance

1. PKCE code_challenge 생성 (S256)
2. 인가 코드 획득 (code_challenge 포함)
3. 토큰 교환 (code_verifier 포함)
4. id_token 검증
5. UserInfo 호출

---

## 테스트 ID 매핑

| Conformance ID | 관련 스펙 | 검증 내용 |
|----------------|----------|----------|
| C1 | Discovery §3 | Discovery 파싱 |
| C2 | JWK RFC 7517 | JWKS 공개키 로딩 |
| C3 | Core §3.1.3.7 | id_token 서명 + claims 검증 |
| C4 | Core §2 | 표준 claims 추출 |
| C5 | Core §5.3.2 | sub 일관성 |
| C6 | Core §5.3 | UserInfo 호환성 |
| C7 | Core §2 (exp) | 만료 토큰 거부 |

---

## 실행 방법

```bash
# conformance 테스트만 실행
go test ./internal/oidc/ -run TestConformance -v

# 전체 테스트
go test ./internal/oidc/ -v
```
