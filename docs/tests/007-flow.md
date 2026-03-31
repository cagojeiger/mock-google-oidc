# Test Spec: Full Flow Integration Tests

전체 OIDC 플로우를 엔드투엔드로 검증하는 통합 테스트.
**테스트 파일**: `internal/oidc/flow_test.go`

---

## 정상 플로우

### T-FLOW-01: 전체 Authorization Code Flow

1. `GET /o/oauth2/v2/auth` → 200, 로그인 폼
2. `POST /o/oauth2/v2/auth` → 302, code + state
3. `POST /token` → 200, access_token + id_token
4. id_token claims 검증 (sub, email, name, nonce, azp)
5. id_token 서명 검증 (JWKS 공개키)
6. `GET /v1/userinfo` → 200, user claims
7. id_token sub == userinfo sub

### T-FLOW-02: 동일 이메일 → 동일 sub

- 같은 이메일로 두 번 로그인
- 두 id_token의 `sub`이 동일해야 함
- **스펙**: Core §2 — sub는 issuer 내에서 고유하고 재할당 안 됨

### T-FLOW-03: 다른 이메일 → 다른 sub

- 다른 이메일로 두 번 로그인
- 두 id_token의 `sub`이 달라야 함

---

## 에러 플로우

### T-FLOW-04: deny 모드

1. `POST /o/oauth2/v2/auth` response_mode=deny
2. redirect에 `error=access_denied`, code 없음

### T-FLOW-05: token_error 모드

1. 인가 성공 (code 발급됨)
2. `POST /token` → 500, `error=server_error`

### T-FLOW-06: userinfo_error 모드

1. 인가 성공
2. 토큰 교환 성공
3. `GET /v1/userinfo` → 500, `error=server_error`

---

## PKCE 플로우

### T-FLOW-07: PKCE S256 전체 플로우

1. code_challenge = BASE64URL(SHA256(verifier)), method=S256
2. 인가 → code
3. 토큰 교환 (code_verifier 포함) → 성공
4. userinfo → 성공

### T-FLOW-08: PKCE 잘못된 verifier

1. 인가 → code
2. 토큰 교환 (wrong verifier) → 400

---

## 이름 분리 플로우

### T-FLOW-09: given_name / family_name 검증

1. name="Alice Kim"으로 로그인
2. id_token: `given_name=Alice`, `family_name=Kim`
3. userinfo: `given_name=Alice`, `family_name=Kim`
4. 양쪽 일치 확인
