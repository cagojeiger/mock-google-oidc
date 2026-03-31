# Test Spec: Discovery Endpoint

**엔드포인트**: `GET /.well-known/openid-configuration`
**준거 표준**: OIDC Discovery 1.0 §3

---

## 순수 함수 테스트

없음 (이 엔드포인트는 정적 JSON 반환)

---

## 핸들러 테스트

### T-DISC-01: 정상 응답

- **요청**: `GET /.well-known/openid-configuration`
- **기대**: 200, `Content-Type: application/json`
- **검증**:
  - `issuer` == `PUBLIC_URL`
  - `authorization_endpoint` == `{issuer}/o/oauth2/v2/auth`
  - `token_endpoint` == `{issuer}/token`
  - `userinfo_endpoint` == `{issuer}/v1/userinfo`
  - `jwks_uri` == `{issuer}/oauth2/v3/certs`

### T-DISC-02: REQUIRED 필드 존재

- **검증**: 아래 필드가 모두 존재하고 비어있지 않음
  - `issuer`
  - `authorization_endpoint`
  - `token_endpoint`
  - `jwks_uri`
  - `response_types_supported`
  - `subject_types_supported`
  - `id_token_signing_alg_values_supported`

### T-DISC-03: 지원 범위 값 검증

- **검증**:
  - `response_types_supported`에 `"code"` 포함
  - `subject_types_supported`에 `"public"` 포함
  - `id_token_signing_alg_values_supported`에 `"RS256"` 포함
  - `scopes_supported`에 `"openid"` 포함
  - `grant_types_supported`에 `"authorization_code"` 포함

---

## Conformance 테스트 (coreos/go-oidc)

### C-DISC-01: 표준 클라이언트 파싱

- **방법**: `oidc.NewProvider(ctx, issuerURL)`
- **기대**: 에러 없이 Provider 객체 반환
- **증명**: Discovery 문서가 표준 OIDC 클라이언트에서 파싱 가능

### C-DISC-02: Endpoint 추출

- **방법**: `provider.Endpoint()`
- **기대**: AuthURL, TokenURL이 discovery 문서의 값과 일치
