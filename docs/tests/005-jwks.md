# Test Spec: JWKS Endpoint

**엔드포인트**: `GET /oauth2/v3/certs`
**준거 표준**: RFC 7517 (JWK), OIDC Core 1.0 §10.1

---

## 순수 함수 테스트

없음 (KeyPair.JWKS()는 구조체 변환이므로 핸들러 테스트로 검증)

---

## 핸들러 테스트

### T-JWK-01: 정상 응답 구조

- **요청**: `GET /oauth2/v3/certs`
- **기대**: 200, `Content-Type: application/json`
- **검증**:
  - `keys` 배열 존재, 1개 이상의 키
  - 첫 번째 키에 `kty`, `alg`, `use`, `kid`, `n`, `e` 필드 존재

### T-JWK-02: 키 속성 값

- **검증**:
  - `kty` == `"RSA"`
  - `alg` == `"RS256"`
  - `use` == `"sig"`
  - `kid` == `"test-idp-key-1"`

### T-JWK-03: Cache-Control 헤더

- **검증**: `Cache-Control` 헤더가 `public`으로 시작

### T-JWK-04: kid 일치

- **검증**: JWKS의 `kid` == id_token JWT 헤더의 `kid`
- id_token 발급 후 헤더를 디코딩하여 `kid` 추출, JWKS의 kid와 비교

---

## Conformance 테스트 (coreos/go-oidc)

### C-JWK-01: 서명 검증

- **방법**:
  1. `oidc.NewProvider(ctx, issuerURL)` → JWKS 자동 로딩
  2. `provider.Verifier(&oidc.Config{ClientID: clientID})`
  3. `verifier.Verify(ctx, rawIDToken)`
- **기대**: 에러 없이 검증 성공
- **증명**: JWKS 포맷이 올바르고, 공개키로 id_token 서명 검증 가능
