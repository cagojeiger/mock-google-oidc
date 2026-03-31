# Test Spec: UserInfo Endpoint

**엔드포인트**: `GET /v1/userinfo`
**준거 표준**: OIDC Core 1.0 §5.3

---

## 순수 함수 테스트

없음 (SplitName은 003-token에서 테스트)

---

## 핸들러 테스트

### T-UI-01: 정상 응답

- **요청**: `GET /v1/userinfo`, `Authorization: Bearer ya29.{valid}`
- **기대**: 200, `Content-Type: application/json`
- **검증**:
  - `sub` == DeterministicSub(email)
  - `email` == 요청 시 이메일
  - `name` == 전체 이름
  - `given_name` == 첫 번째 이름
  - `family_name` == 나머지 이름
  - `email_verified` == true

### T-UI-02: Bearer 토큰 없음

- **요청**: `GET /v1/userinfo` (Authorization 헤더 없음)
- **기대**: 401, `error=invalid_token`
- **스펙**: Core §5.3.3

### T-UI-03: 유효하지 않은 토큰

- **요청**: `Authorization: Bearer invalid-token`
- **기대**: 401, `error=invalid_token`

### T-UI-04: userinfo_error 모드

- **설정**: response_mode=userinfo_error로 인가 후 토큰 발급
- **요청**: 해당 access_token으로 userinfo
- **기대**: 500, `error=server_error`

### T-UI-05: sub 일관성

- **검증**: userinfo의 `sub` == 동일 세션 id_token의 `sub`
- **스펙**: Core §5.3.2 — UserInfo sub MUST match id_token sub

---

## Conformance 테스트 (coreos/go-oidc)

### C-UI-01: 표준 클라이언트 UserInfo 호출

- **방법**: `provider.UserInfo(ctx, oauth2TokenSource)`
- **기대**: 에러 없이 UserInfo 반환
- **검증**: `sub` 필드 존재
